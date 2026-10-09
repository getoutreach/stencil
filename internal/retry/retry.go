// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file provides a minimal retry helper until gobox provides one.

// Package retry retries a function with backoff.
//
// It is intentionally the smallest API that stencil needs (DoValue, WithBackoff,
// WithOnRetry, Permanent and Exponential), using functional options so it is
// easy to replace with a shared retry helper later. Delete this package once
// gobox provides one. The attempt cap is fixed at 3; a caller that needs another
// count should add a WithMaxAttempts option rather than change the constant.
package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"

	"github.com/getoutreach/gobox/pkg/async"
	"github.com/pkg/errors"
)

// maxAttempts is the maximum total number of calls to fn, including the first.
const maxAttempts = 3

// config is the resolved configuration for DoValue.
type config struct {
	// strategy computes the delay before each retry.
	strategy Strategy

	// onRetry, if set, runs immediately before each sleep between attempts.
	onRetry func(ctx context.Context, attempt int, err error, wait time.Duration)
}

// Option configures DoValue.
type Option func(*config)

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

// Strategy describes a backoff curve: how long to wait before a given retry.
// It is opaque so it can only be obtained from Exponential.
type Strategy interface {
	// delay returns the wait before the attempt-th retry: delay(1) is the wait
	// after the first call to fn fails.
	delay(attempt int) time.Duration
}

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

// permanentError marks the wrapped error as not retryable.
type permanentError struct{ err error }

func (p *permanentError) Error() string { return p.err.Error() }
func (p *permanentError) Unwrap() error { return p.err }

// Permanent wraps err so DoValue stops immediately and returns err unchanged
// (only Permanent's own marker is removed). Wrap err with any extra context
// before calling Permanent, not after.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &permanentError{err: err}
}

// DoValue calls fn until it succeeds, returns an error wrapped by Permanent, ctx
// is done, or 3 attempts have been made, in which case it returns the last error
// from fn. On failure it returns the zero value. Each error from fn is handled
// in this order:
//
//  1. ctx is done: stop and return ctx.Err().
//  2. The error is wrapped by Permanent: stop and return the unwrapped error.
//  3. The attempt budget is exhausted: stop and return the error.
//  4. Otherwise wait for the Strategy's delay, then try again.
func DoValue[T any](ctx context.Context, fn func(context.Context) (T, error), opts ...Option) (T, error) {
	c := &config{strategy: Exponential(time.Second, 5*time.Minute, 1.6, 0.2)}
	for _, opt := range opts {
		opt(c)
	}

	var zero T
	for attempt := 1; ; attempt++ {
		v, err := fn(ctx)
		if err == nil {
			return v, nil
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}

		var perm *permanentError
		if errors.As(err, &perm) {
			return zero, perm.err
		}

		if attempt >= maxAttempts {
			return zero, err
		}

		wait := c.strategy.delay(attempt)
		if c.onRetry != nil {
			c.onRetry(ctx, attempt, err, wait)
		}
		async.Sleep(ctx, wait)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
	}
}
