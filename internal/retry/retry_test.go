// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for the retry package.

package retry_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/getoutreach/stencil/internal/retry"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/assert/cmp"
)

var (
	errBoom      = errors.New("boom")
	errTransient = errors.New("transient")
)

// fast waits exactly 1ms before every retry.
func fast() retry.Option {
	return retry.WithBackoff(retry.Exponential(time.Millisecond, time.Millisecond, 1, 0))
}

func TestDoValueRetriesUntilSuccess(t *testing.T) {
	calls := 0
	v, err := retry.DoValue(t.Context(), func(context.Context) (string, error) {
		calls++
		if calls < 3 {
			return "", errBoom
		}
		return "ok", nil
	}, fast())
	assert.NilError(t, err)
	assert.Equal(t, v, "ok")
	assert.Equal(t, calls, 3)
}

func TestDoValueStopsAfterThreeAttempts(t *testing.T) {
	calls := 0
	v, err := retry.DoValue(t.Context(), func(context.Context) (string, error) {
		calls++
		return "partial", errBoom
	}, fast())
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, v, "", "expected the zero value on failure")
	assert.Equal(t, calls, 3)
}

func TestDoValuePermanentStopsImmediately(t *testing.T) {
	calls := 0
	_, err := retry.DoValue(t.Context(), func(context.Context) (string, error) {
		calls++
		return "", retry.Permanent(errBoom)
	}, fast())
	assert.Equal(t, calls, 1)
	assert.Equal(t, err, errBoom, "only Permanent's own marker is removed")
}

func TestDoValueReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	_, err := retry.DoValue(ctx, func(context.Context) (string, error) {
		calls++
		cancel()
		return "", errBoom
	}, fast())
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, calls, 1)
}

func TestDoValueOnRetryReceivesAttemptErrorAndWait(t *testing.T) {
	var attempts []int
	_, err := retry.DoValue(t.Context(), func(context.Context) (string, error) { return "", errBoom },
		fast(),
		retry.WithOnRetry(func(_ context.Context, attempt int, err error, wait time.Duration) {
			assert.ErrorIs(t, err, errBoom)
			assert.Equal(t, wait, time.Millisecond)
			attempts = append(attempts, attempt)
		}))
	assert.ErrorIs(t, err, errBoom, "expected the last error once the attempts are used up")
	assert.DeepEqual(t, attempts, []int{1, 2})
}

func TestDoWithMaxAttempts(t *testing.T) {
	for _, n := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			calls, retries := 0, 0
			err := retry.Do(t.Context(), func(context.Context) error {
				calls++
				return errBoom
			}, fast(), retry.WithMaxAttempts(n),
				retry.WithOnRetry(func(context.Context, int, error, time.Duration) { retries++ }))
			assert.ErrorIs(t, err, errBoom)
			assert.Equal(t, calls, n, "expected n calls in total, including the first")
			assert.Equal(t, retries, n-1, "expected no wait after the last call")
		})
	}
}

func TestDoWithMaxAttemptsRejectsLessThanOne(t *testing.T) {
	for _, n := range []int{0, -1} {
		calls := 0
		err := retry.Do(t.Context(), func(context.Context) error {
			calls++
			return nil
		}, retry.WithMaxAttempts(n))
		assert.ErrorIs(t, err, retry.ErrInvalidMaxAttempts, "expected a configuration error for n=%d", n)
		assert.Equal(t, calls, 0, "expected fn not to be called")
	}
}

func TestDoLastWithMaxAttemptsWins(t *testing.T) {
	calls := 0
	err := retry.Do(t.Context(), func(context.Context) error {
		calls++
		return nil
	}, retry.WithMaxAttempts(0), retry.WithMaxAttempts(5))
	assert.NilError(t, err, "expected a valid option to clear an earlier invalid one")
	assert.Equal(t, calls, 1)
}

func TestDoValueWithMaxAttempts(t *testing.T) {
	calls := 0
	v, err := retry.DoValue(t.Context(), func(context.Context) (string, error) {
		calls++
		return "partial", errBoom
	}, fast(), retry.WithMaxAttempts(4))
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, v, "", "expected the zero value on failure")
	assert.Equal(t, calls, 4)
}

// retryAfterError is an error that asks for a minimum wait before the next attempt.
type retryAfterError struct {
	err   error
	after time.Duration
}

func (e *retryAfterError) Error() string             { return e.err.Error() }
func (e *retryAfterError) Unwrap() error             { return e.err }
func (e *retryAfterError) RetryAfter() time.Duration { return e.after }

// waits runs fn under fast() with extra options, and returns the delay before each retry.
func waits(t *testing.T, fn func(context.Context) error, opts ...retry.Option) []time.Duration {
	t.Helper()
	var got []time.Duration
	opts = append([]retry.Option{fast(), retry.WithOnRetry(func(_ context.Context, _ int, _ error, wait time.Duration) {
		got = append(got, wait)
	})}, opts...)
	_ = retry.Do(t.Context(), fn, opts...)
	return got
}

func TestDoWithRetryableStopsOnFalse(t *testing.T) {
	calls := 0
	err := retry.Do(t.Context(), func(context.Context) error {
		calls++
		return errBoom
	}, fast(), retry.WithRetryable(func(error) bool { return false }))
	assert.Equal(t, err, errBoom)
	assert.Equal(t, calls, 1, "expected no retry for an error that is not retryable")
}

func TestDoWithRetryableRetriesOnTrue(t *testing.T) {
	calls := 0
	err := retry.Do(t.Context(), func(context.Context) error {
		calls++
		if calls < 3 {
			return fmt.Errorf("attempt %d: %w", calls, errTransient)
		}
		return errBoom
	}, fast(), retry.WithRetryable(func(err error) bool { return errors.Is(err, errTransient) }))
	assert.Equal(t, err, errBoom, "expected the first error that is not retryable")
	assert.Equal(t, calls, 3)
}

func TestDoWithRetryableIsNotAskedAboutPermanentOrContextErrors(t *testing.T) {
	asked := 0
	retryable := retry.WithRetryable(func(error) bool {
		asked++
		return true
	})

	err := retry.Do(t.Context(), func(context.Context) error { return retry.Permanent(errBoom) }, fast(), retryable)
	assert.Equal(t, err, errBoom)

	ctx, cancel := context.WithCancel(t.Context())
	err = retry.Do(ctx, func(context.Context) error {
		cancel()
		return errBoom
	}, fast(), retryable)
	assert.ErrorIs(t, err, context.Canceled)

	assert.Equal(t, asked, 0)
}

func TestWithLoggingLogsEachRetryAndKeepsTheOnRetryHook(t *testing.T) {
	log, hook := logtest.NewNullLogger()
	hookCalls := 0

	err := retry.Do(t.Context(), func(context.Context) error { return errBoom }, fast(),
		retry.WithLogging(log, "do the thing"),
		retry.WithOnRetry(func(context.Context, int, error, time.Duration) { hookCalls++ }))
	assert.ErrorIs(t, err, errBoom)

	entries := hook.AllEntries()
	assert.Equal(t, len(entries), 2, "expected one log entry for each of the 2 retries")
	assert.Equal(t, entries[0].Level, logrus.WarnLevel)
	assert.Equal(t, entries[0].Message, "Failed to do the thing (attempt 1), retrying in 1ms")
	assert.Equal(t, entries[1].Message, "Failed to do the thing (attempt 2), retrying in 1ms")
	assert.Equal(t, entries[0].Data[logrus.ErrorKey], errBoom)
	assert.Equal(t, hookCalls, 2, "expected WithOnRetry to run as well")
}

func TestConstantWaitsTheSameBeforeEveryRetry(t *testing.T) {
	got := waits(t, func(context.Context) error { return errBoom },
		retry.WithBackoff(retry.Constant(2*time.Millisecond)), retry.WithMaxAttempts(4))
	assert.DeepEqual(t, got, []time.Duration{2 * time.Millisecond, 2 * time.Millisecond, 2 * time.Millisecond})
}

func TestRetryAfterSetsAMinimumWait(t *testing.T) {
	for _, tc := range []struct {
		name        string
		after, want time.Duration
	}{
		{"raises the wait", 20 * time.Millisecond, 20 * time.Millisecond},
		{"never shortens the wait", time.Microsecond, time.Millisecond}, // fast() waits 1ms
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := waits(t, func(context.Context) error {
				// Wrapped, to show that RetryAfter is found anywhere in the chain.
				return fmt.Errorf("calling upstream: %w", &retryAfterError{err: errBoom, after: tc.after})
			}, retry.WithMaxAttempts(2))
			assert.DeepEqual(t, got, []time.Duration{tc.want})
		})
	}
}

func TestExponentialPanicsOnInvalidArguments(t *testing.T) {
	assert.Assert(t, cmp.Panics(func() { retry.Exponential(time.Minute, time.Second, 2, 0) }))
}
