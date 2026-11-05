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
// limit that lasts longer is not retried, so that stencil does not look hung while
// it waits, and the error says when the limit resets.
const maxRateLimitWait = time.Minute

// retryAfterError is an error that says how long to wait before the next attempt,
// see retry.RetryAfter.
type retryAfterError struct {
	error

	after time.Duration
}

func (e retryAfterError) Unwrap() error             { return e.error }
func (e retryAfterError) RetryAfter() time.Duration { return e.after }

// fetchRelease is release.Fetch, retried when GitHub or the network fails in a way
// that could go away, see classifyGitHubError. Only the request is retried, not the
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
// servers are retried, rate limits are waited out, and anything else is permanent.
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

// afterRateLimit marks err, a rate limit error, to be retried after wait, unless
// that is longer than maxRateLimitWait. A wait that has passed is shorter than
// the backoff, so retry.Do uses the backoff.
func afterRateLimit(err error, wait time.Duration) error {
	if wait > maxRateLimitWait {
		return retry.Permanent(err)
	}
	return retryAfterError{err, wait}
}
