// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for retrying the download of a release.

package extensions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/getoutreach/gobox/pkg/cli/updater/release"
	"github.com/getoutreach/stencil/internal/retry"
	gogithub "github.com/google/go-github/v84/github"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

var (
	errConnectionRefused = errors.New("connection refused")
	errAssetNotFound     = errors.New("failed to find asset")
)

// request is set on the fake responses because the Error methods of go-github read
// Response.Request.
var request = &http.Request{Method: http.MethodGet, URL: &url.URL{}}

// githubError is the error of a request that GitHub answered with status.
func githubError(status int) error {
	return &gogithub.ErrorResponse{Response: &http.Response{StatusCode: status, Request: request}}
}

// abuseLimited is the error of a secondary rate limit, with the wait that it asks for, if any.
func abuseLimited(retryAfter *time.Duration) error {
	return &gogithub.AbuseRateLimitError{
		Response:   &http.Response{StatusCode: http.StatusForbidden, Request: request},
		RetryAfter: retryAfter,
	}
}

// attempts returns how many times retry.Do calls an operation that always fails with
// err. The error goes through classifyGitHubError, like the error of release.Fetch.
func attempts(t *testing.T, err error) (calls int) {
	t.Helper()

	_ = retry.Do(t.Context(), func(context.Context) error {
		calls++
		return classifyGitHubError(err)
	}, retry.WithBackoff(retry.Constant(time.Millisecond)))
	return calls
}

// waitFor returns the wait that classifyGitHubError asks retry.Do for when err fails.
func waitFor(t *testing.T, err error) time.Duration {
	t.Helper()

	var retryAfter retry.RetryAfter
	assert.Assert(t, errors.As(classifyGitHubError(err), &retryAfter), "expected a wait for %v", err)
	return retryAfter.RetryAfter()
}

// rateLimited is the error of a primary rate limit that resets at reset.
func rateLimited(reset time.Time) error {
	return &gogithub.RateLimitError{
		Rate:     gogithub.Rate{Reset: gogithub.Timestamp{Time: reset}},
		Response: &http.Response{StatusCode: http.StatusForbidden, Request: request},
	}
}

// TestClassifyGitHubError checks that retry.Do calls the operation 3 times for an
// error that is retried, and once for any other error.
func TestClassifyGitHubError(t *testing.T) {
	const retried, notRetried = 3, 1

	for _, tc := range []struct {
		name  string
		err   error
		calls int
	}{
		{"no error", nil, notRetried},
		{"server error", githubError(http.StatusServiceUnavailable), retried},
		{"too many requests", githubError(http.StatusTooManyRequests), retried},
		{"wrapped server error", fmt.Errorf("failed to get release: %w", githubError(http.StatusBadGateway)), retried},
		{"connection error", &net.OpError{Op: "dial", Err: errConnectionRefused}, retried},
		{"abuse rate limit without a wait", abuseLimited(nil), notRetried},
		{"abuse rate limit that says to retry now", abuseLimited(gogithub.Ptr(time.Duration(0))), retried},
		{"release not found", githubError(http.StatusNotFound), notRetried},
		{"bad credentials", githubError(http.StatusUnauthorized), notRetried},
		{"error that is not known", errAssetNotFound, notRetried},
		{"rate limit that lasts too long", rateLimited(time.Now().Add(time.Hour)), notRetried},
		{"abuse rate limit that lasts too long", abuseLimited(gogithub.Ptr(time.Hour)), notRetried},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, attempts(t, tc.err), tc.calls)
		})
	}
}

func TestClassifyGitHubErrorAsksToWaitForARateLimit(t *testing.T) {
	t.Run("rate limit", func(t *testing.T) {
		wait := waitFor(t, rateLimited(time.Now().Add(30*time.Second)))
		assert.Assert(t, wait > 0 && wait <= 30*time.Second, "wait: %v", wait)
	})

	t.Run("abuse rate limit", func(t *testing.T) {
		assert.Equal(t, waitFor(t, abuseLimited(gogithub.Ptr(20*time.Millisecond))), 20*time.Millisecond)
	})

	t.Run("rate limit that reset", func(t *testing.T) {
		// A wait of zero or less makes retry.Do use its backoff.
		assert.Assert(t, waitFor(t, rateLimited(time.Now().Add(-time.Minute))) <= 0)
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// withTransport makes the default HTTP transport answer every request with status,
// until the end of the test. release.Fetch has no token here, so it uses that
// transport. It is a global, so the tests that call it must not run in parallel.
func withTransport(t *testing.T, status int) *atomic.Int32 {
	t.Helper()

	requests := new(atomic.Int32)
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requests.Add(1)
		return &http.Response{
			StatusCode: status,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"message":"test error"}`)),
			Request:    r,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = original })

	return requests
}

// fetchWithStatus runs fetchRelease against a GitHub that answers every request
// with status, and returns the number of requests and the error. Its errors come
// from the go-github that gobox uses, so the tests that call it fail if
// classifyGitHubError imports another version of go-github.
func fetchWithStatus(t *testing.T, status int) (int32, error) {
	t.Helper()

	requests := withTransport(t, status)

	_, _, err := fetchRelease(t.Context(), logrus.New(), "",
		&release.FetchOptions{RepoURL: "https://github.com/example/repo", Tag: "v1.0.0", AssetName: "asset"},
		retry.WithBackoff(retry.Constant(time.Millisecond)))
	return requests.Load(), err
}

func TestFetchReleaseRetriesTransientErrors(t *testing.T) {
	requests, err := fetchWithStatus(t, http.StatusServiceUnavailable)

	assert.Assert(t, err != nil)
	assert.Equal(t, requests, int32(3), "expected the 3 attempts of retry.Do")
}

func TestFetchReleaseDoesNotRetryPermanentErrors(t *testing.T) {
	requests, err := fetchWithStatus(t, http.StatusNotFound)

	assert.Assert(t, err != nil)
	assert.Equal(t, requests, int32(1), "expected no retry, because asking again gives the same answer")
}

func TestFetchReleaseDoesNotLogCredentials(t *testing.T) {
	var logged bytes.Buffer
	log := logrus.New()
	log.Out = &logged
	withTransport(t, http.StatusServiceUnavailable)

	_, _, err := fetchRelease(t.Context(), log, "",
		&release.FetchOptions{RepoURL: "https://ghp_secret@github.com/example/repo", Tag: "v1.0.0", AssetName: "asset"},
		retry.WithBackoff(retry.Constant(time.Millisecond)))

	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(logged.String(), "retrying"), "expected a warning for the retry")
	assert.Assert(t, !strings.Contains(logged.String()+err.Error(), "ghp_secret"), "the log or the error has the token:\n%s\n%v", &logged, err)
}

func TestFetchReleaseDoesNotReturnCredentials(t *testing.T) {
	// gobox puts the whole URL in its error when the path is not /org/repo.
	_, _, err := fetchRelease(t.Context(), logrus.New(), "",
		&release.FetchOptions{RepoURL: "https://ghp_secret@github.com/example/repo/extra", Tag: "v1.0.0", AssetName: "asset"})

	assert.ErrorContains(t, err, "invalid Github URL")
	assert.Assert(t, !strings.Contains(err.Error(), "ghp_secret"), "the error has the token: %v", err)
}
