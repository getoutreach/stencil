// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file provides a minimal retry helper until gobox provides one.

// Package retry retries a function with backoff.
//
// It is a small subset of a retry helper that is not public. Port more of its
// features here when a caller needs them. Delete this package once gobox provides
// one.
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/getoutreach/gobox/pkg/async"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// defaultMaxAttempts is the most attempts that Do and DoValue make without WithMaxAttempts.
const defaultMaxAttempts = 3

// config holds the settings that Options change.
type config struct {
	// maxAttempts is the maximum total number of calls to fn, including the first.
	maxAttempts int

	// retryable classifies an error from fn: true retries, false stops. A nil
	// retryable retries every error.
	retryable func(error) bool

	// strategy computes the delay before each retry.
	strategy Strategy

	// onRetry, if set, runs immediately before each sleep between attempts.
	onRetry func(ctx context.Context, attempt int, err error, wait time.Duration)

	// logRetry, if set, runs immediately before each sleep between attempts, before onRetry.
	logRetry func(attempt int, err error, wait time.Duration)
}

// Option configures Do and DoValue.
type Option func(*config)

// WithMaxAttempts sets the maximum total number of calls to fn, including the
// first. The default is 3. It panics if n is below 1.
func WithMaxAttempts(n int) Option {
	if n < 1 {
		panic(fmt.Sprintf("retry: WithMaxAttempts: n must be at least 1, got %d", n))
	}
	return func(c *config) { c.maxAttempts = n }
}

// WithRetryable sets the function that decides whether an error from fn is
// retried: true retries, false returns the error at once. The default retries
// every error. Permanent errors are never passed to fn, and neither is any
// error once ctx is done.
func WithRetryable(fn func(error) bool) Option {
	return func(c *config) { c.retryable = fn }
}

// WithBackoff sets the Strategy used to compute the delay before each retry.
// The default is Exponential(1s, 5m, 1.6, 0.2).
func WithBackoff(s Strategy) Option {
	return func(c *config) { c.strategy = s }
}

// WithOnRetry registers fn to run immediately before each sleep between
// attempts. attempt is the 1-indexed attempt that just failed, err is the
// error it returned, and wait is the delay before the next attempt.
func WithOnRetry(fn func(ctx context.Context, attempt int, err error, wait time.Duration)) Option {
	return func(c *config) { c.onRetry = fn }
}

// WithLogging makes Do log a warning to log before each sleep between attempts, like
// "Failed to clone x (attempt 1), retrying in 1s", with the error in the "error"
// field. operation is a verb phrase that completes "Failed to ...". The log line is
// written before the WithOnRetry hook runs, and neither replaces the other. The
// operation and the error are logged as they are, so a caller must redact any URL in
// them first.
func WithLogging(log logrus.FieldLogger, operation string) Option {
	return func(c *config) {
		c.logRetry = func(attempt int, err error, wait time.Duration) {
			log.WithError(err).Warnf("Failed to %s (attempt %d), retrying in %s", operation, attempt, wait)
		}
	}
}

// Strategy describes a backoff curve: how long to wait before a given retry.
// It is opaque so it can only be obtained from Exponential or Constant.
type Strategy interface {
	// delay returns the wait before the attempt-th retry: delay(1) is the wait
	// after the first call to fn fails.
	delay(attempt int) time.Duration
}

// exponential is the Strategy that Exponential returns.
type exponential struct {
	initial, maxDelay time.Duration
	multiplier        float64
	jitter            float64
}

// Exponential returns a Strategy whose delay grows from initial by multiplier
// each attempt, capped at maxDelay, randomized by +-jitter (0.2 means +-20%, 0
// means no jitter). It panics if initial or multiplier is not positive, initial
// exceeds maxDelay, or jitter is outside [0, 1].
func Exponential(initial, maxDelay time.Duration, multiplier, jitter float64) Strategy {
	if initial <= 0 || multiplier <= 0 || initial > maxDelay || jitter < 0 || jitter > 1 {
		panic("retry: Exponential: invalid arguments")
	}
	return exponential{initial, maxDelay, multiplier, jitter}
}

func (e exponential) delay(attempt int) time.Duration {
	d := min(float64(e.initial)*math.Pow(e.multiplier, float64(attempt-1)), float64(e.maxDelay))
	delta := e.jitter * d
	return time.Duration(d - delta + rand.Float64()*2*delta) //nolint:gosec // Why: jitter does not need a secure source.
}

// constant is the Strategy that Constant returns.
type constant time.Duration

// Constant returns a Strategy that waits d before every retry.
func Constant(d time.Duration) Strategy { return constant(d) }

func (c constant) delay(int) time.Duration { return time.Duration(c) }

// RetryAfter is implemented by an error that says how long to wait before the
// next attempt, such as a rate limit error. Do waits at least that long when an
// error in the chain returned by fn implements it, even if the Strategy's delay
// is shorter.
//
//nolint:revive // Why: The name stutters with the package, but it matches the helper that this package is a subset of.
type RetryAfter interface {
	RetryAfter() time.Duration
}

// permanentError marks the wrapped error as not retryable.
type permanentError struct{ err error }

func (p *permanentError) Error() string { return p.err.Error() }
func (p *permanentError) Unwrap() error { return p.err }

// Permanent wraps err so Do and DoValue stop immediately and return err unchanged
// (only Permanent's own marker is removed). Wrap err with any extra context
// before calling Permanent, not after.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// Do calls fn until it succeeds, returns an error wrapped by Permanent, ctx is
// done, WithRetryable returns false, or the attempt budget (3 by default, see
// WithMaxAttempts) is used up, in which case it returns the last error from fn.
// Each error from fn is handled in this order:
//
//  1. ctx is done: stop and return ctx.Err().
//  2. The error is wrapped by Permanent: stop and return the unwrapped error.
//  3. WithRetryable is set and returns false: stop and return the error.
//  4. The attempt budget is exhausted: stop and return the error.
//  5. Otherwise wait for the Strategy's delay (at least the error's RetryAfter,
//     if it implements one), then try again.
func Do(ctx context.Context, fn func(context.Context) error, opts ...Option) error {
	c := &config{
		maxAttempts: defaultMaxAttempts,
		strategy:    Exponential(time.Second, 5*time.Minute, 1.6, 0.2),
	}
	for _, opt := range opts {
		opt(c)
	}

	for attempt := 1; ; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		var perm *permanentError
		if errors.As(err, &perm) {
			return perm.err
		}

		if c.retryable != nil && !c.retryable(err) {
			return err
		}

		if attempt >= c.maxAttempts {
			return err
		}

		wait := c.strategy.delay(attempt)
		var ra RetryAfter
		if errors.As(err, &ra) {
			wait = max(wait, ra.RetryAfter())
		}
		if c.logRetry != nil {
			c.logRetry(attempt, err, wait)
		}
		if c.onRetry != nil {
			c.onRetry(ctx, attempt, err, wait)
		}
		async.Sleep(ctx, wait)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
}

// DoValue is Do for a function that also returns a value. On failure it returns
// the zero value of T, even if fn returned a value alongside its error.
func DoValue[T any](ctx context.Context, fn func(context.Context) (T, error), opts ...Option) (T, error) {
	var value T
	err := Do(ctx, func(ctx context.Context) error {
		v, err := fn(ctx)
		if err != nil {
			return err
		}
		value = v
		return nil
	}, opts...)
	return value, err
}
