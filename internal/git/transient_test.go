// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for the classification of git remote errors.

package git_test

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	stgit "github.com/getoutreach/stencil/internal/git"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
	"gotest.tools/v3/assert"
)

var (
	errOther = errors.New("other")

	// These have the messages of the errors that net/http returns for HTTP/2.
	errStreamReset     = errors.New("stream error: stream ID 1; INTERNAL_ERROR; received from peer")
	errServerGoAway    = errors.New(`http2: server sent GOAWAY and closed the connection; LastStreamID=1, ErrCode=NO_ERROR, debug=""`)
	errTransportGoAway = errors.New("http2: Transport received GOAWAY from server ErrCode:NO_ERROR")
	errConnectionLost  = errors.New("http2: client connection lost")
)

// timeoutError is a net.Error that says whether it is a timeout.
type timeoutError struct{ timeout bool }

func (e timeoutError) Error() string   { return "timeout test error" }
func (e timeoutError) Timeout() bool   { return e.timeout }
func (e timeoutError) Temporary() bool { return false }

// lsRemote lists the refs of the repository at url, like the resolver does.
func lsRemote(ctx context.Context, repoURL string) error {
	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{Name: "origin", URLs: []string{repoURL}})
	_, err := remote.ListContext(ctx, &git.ListOptions{})
	return err
}

func TestIsTransientError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other error", errOther, false},
		{"wrapped connection error", fmt.Errorf("ls-remote: %w", &net.OpError{Op: "read"}), true},
		{"missing host", &net.OpError{Op: "dial", Err: &net.DNSError{Err: "no such host", IsNotFound: true}}, false},
		{"DNS error that is not a missing host", &net.OpError{Op: "dial", Err: &net.DNSError{Err: "server misbehaving"}}, true},
		{"timeout", timeoutError{timeout: true}, true},
		{"network error that is not a timeout", timeoutError{timeout: false}, false},
		{"missing host without a connection error", &net.DNSError{Err: "no such host", IsNotFound: true}, false},
		{"response that ended early", fmt.Errorf("read: %w", io.ErrUnexpectedEOF), true},
		{"connection that a server closed", &url.Error{Op: "Get", URL: "https://example.test", Err: io.EOF}, true},
		{"HTTP/2 stream reset", &url.Error{Op: "Get", URL: "https://example.test", Err: errStreamReset}, true},
		{"HTTP/2 GOAWAY", &url.Error{Op: "Get", URL: "https://example.test", Err: errServerGoAway}, true},
		{"HTTP/2 GOAWAY during a request", &url.Error{Op: "Get", URL: "https://example.test", Err: errTransportGoAway}, true},
		{"HTTP/2 connection lost", &url.Error{Op: "Get", URL: "https://example.test", Err: errConnectionLost}, true},
		{"certificate that no authority signed", &url.Error{Op: "Get", URL: "https://example.test", Err: x509.UnknownAuthorityError{}}, false},
		{"context canceled", context.Canceled, false},
		{"context deadline exceeded", context.DeadlineExceeded, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, stgit.IsTransientError(tc.err), tc.want)
		})
	}
}

// TestIsTransientErrorFromGoGit checks the errors that go-git returns, because
// go-git wraps some of them in ways that errors.As does not see through.
func TestIsTransientErrorFromGoGit(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   bool
	}{
		{http.StatusInternalServerError, true},
		{http.StatusNotImplemented, false},
		{http.StatusRequestTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusUnauthorized, false},
		{http.StatusNotFound, false},
	} {
		t.Run(fmt.Sprintf("HTTP %d", tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "test error", tc.status)
			}))
			t.Cleanup(srv.Close)

			err := lsRemote(t.Context(), srv.URL+"/repo.git")
			assert.Assert(t, err != nil)
			assert.Equal(t, stgit.IsTransientError(err), tc.want, "error: %v", err)
		})
	}

	t.Run("nothing is listening", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		repoURL := srv.URL + "/repo.git"
		srv.Close()

		err := lsRemote(t.Context(), repoURL)
		assert.Assert(t, err != nil)
		assert.Assert(t, stgit.IsTransientError(err), "error: %v", err)
	})
}

// TestIsTransientErrorFromHTTP2 checks the error that net/http returns when a server
// resets an HTTP/2 stream, because its type is not exported.
func TestIsTransientErrorFromHTTP2(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	srv.EnableHTTP2 = true
	srv.StartTLS()
	t.Cleanup(srv.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, http.NoBody)
	assert.NilError(t, err)
	resp, err := srv.Client().Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	assert.ErrorContains(t, err, "stream error", "expected the server to reset the stream")
	assert.Assert(t, stgit.IsTransientError(err), "error: %v", err)
}

func TestIsTransientStatus(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   bool
	}{
		{http.StatusNotFound, false},
		{http.StatusRequestTimeout, true},
		{http.StatusTooManyRequests, true},
		{http.StatusInternalServerError, true},
		{http.StatusNotImplemented, false},
		{http.StatusServiceUnavailable, true},
		{http.StatusHTTPVersionNotSupported, false},
		{520, true}, // 520 is not an official status. Cloudflare uses it for an unknown error.
	} {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			assert.Equal(t, stgit.IsTransientStatus(tc.status), tc.want)
		})
	}
}
