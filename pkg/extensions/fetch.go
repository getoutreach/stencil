// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file retries the download of a release from GitHub.

package extensions

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/updater/release"
	stgit "github.com/getoutreach/stencil/internal/git"
	"github.com/getoutreach/stencil/internal/retry"
	gogithub "github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
)

// maxRateLimitWait is the longest that a rate limit can make a retry wait. A rate
// limit that lasts longer is not retried, so that stencil does not look hung.
const maxRateLimitWait = time.Minute

// retryAfterError wraps an error with the wait that retry.Do honors. See
// retry.RetryAfter.
type retryAfterError struct {
	error

	after time.Duration
}

func (e retryAfterError) Unwrap() error             { return e.error }
func (e retryAfterError) RetryAfter() time.Duration { return e.after }

// fetchRelease is release.Fetch, retried when GitHub or the network fails in a way
// that could go away. See classifyGitHubError. Only the request is retried, not the
// reading of the asset. retryOpts are applied after the defaults.
func fetchRelease(ctx context.Context, log logrus.FieldLogger, token cfg.SecretData,
	opts *release.FetchOptions, retryOpts ...retry.Option) (io.ReadCloser, string, error) {
	var asset io.ReadCloser
	var name string
	err := retry.Do(ctx, func(ctx context.Context) error {
		var err error
		asset, name, _, err = release.Fetch(ctx, token, opts)
		return classifyGitHubError(stgit.RedactError(err))
	}, append([]retry.Option{retry.WithLogging(log, "fetch the release "+opts.Tag+" of "+stgit.RedactURL(opts.RepoURL))}, retryOpts...)...)

	return asset, name, err
}

// classifyGitHubError marks err for retry.Do: network errors and busy or broken
// servers are retried, rate limits that reset within maxRateLimitWait are waited out,
// and anything else is permanent.
func classifyGitHubError(err error) error {
	if err == nil {
		return nil
	}

	var rateLimit *gogithub.RateLimitError
	var abuse *gogithub.AbuseRateLimitError
	var response *gogithub.ErrorResponse
	switch {
	case errors.As(err, &rateLimit):
		return afterRateLimit(err, time.Until(rateLimit.Rate.Reset.Time))
	case errors.As(err, &abuse):
		// Without a wait, GitHub asks for at least a minute. That is the longest that
		// stencil waits, so the error is not retried.
		if abuse.RetryAfter == nil {
			return retry.Permanent(err)
		}
		return afterRateLimit(err, abuse.GetRetryAfter())
	case errors.As(err, &response) && response.Response != nil:
		if stgit.IsTransientStatus(response.Response.StatusCode) {
			return err
		}
	case stgit.IsTransientError(err):
		return err
	}

	return retry.Permanent(err)
}

// afterRateLimit marks err, a rate limit error, to be retried after wait. It is
// permanent when wait is longer than maxRateLimitWait. When the limit has already
// reset, wait is not positive and retry.Do uses its backoff.
func afterRateLimit(err error, wait time.Duration) error {
	if wait > maxRateLimitWait {
		return retry.Permanent(err)
	}
	return retryAfterError{err, wait}
}
