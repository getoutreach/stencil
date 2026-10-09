// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for the retry package.

package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/getoutreach/stencil/internal/retry"
	"gotest.tools/v3/assert"
)

var errBoom = errors.New("boom")

// fast waits exactly 1ms before every retry.
func fast() retry.Option {
	return retry.WithBackoff(retry.Exponential(time.Millisecond, time.Millisecond, 1, 0))
}

func TestDoValueRetriesUntilSuccess(t *testing.T) {
	calls := 0
	v, err := retry.DoValue(context.Background(), func(context.Context) (string, error) {
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
	v, err := retry.DoValue(context.Background(), func(context.Context) (string, error) {
		calls++
		return "partial", errBoom
	}, fast())
	assert.ErrorIs(t, err, errBoom)
	assert.Equal(t, v, "", "expected the zero value on failure")
	assert.Equal(t, calls, 3)
}

func TestDoValuePermanentStopsImmediately(t *testing.T) {
	calls := 0
	_, err := retry.DoValue(context.Background(), func(context.Context) (string, error) {
		calls++
		return "", retry.Permanent(errBoom)
	}, fast())
	assert.Equal(t, calls, 1)
	assert.Equal(t, err, errBoom, "only Permanent's own marker is removed")
}

func TestDoValueReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
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
	_, _ = retry.DoValue(context.Background(), func(context.Context) (string, error) { return "", errBoom },
		fast(),
		retry.WithOnRetry(func(_ context.Context, attempt int, err error, wait time.Duration) {
			assert.ErrorIs(t, err, errBoom)
			assert.Equal(t, wait, time.Millisecond)
			attempts = append(attempts, attempt)
		}))
	assert.DeepEqual(t, attempts, []int{1, 2})
}

func TestExponentialPanicsOnInvalidArguments(t *testing.T) {
	assert.Assert(t, func() (panicked bool) {
		defer func() { panicked = recover() != nil }()
		retry.Exponential(time.Minute, time.Second, 2, 0)
		return false
	}())
}
