// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file classifies the errors of git remote operations.

package git

import (
	"errors"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// IsTransientError reports whether err from an operation on a git remote, such as a
// clone or an ls-remote, could go away if the operation is tried again. That is a
// connection error (except an unknown host), a timeout, a response that ended early
// (io.EOF or io.ErrUnexpectedEOF), an HTTP/2 stream reset, GOAWAY or lost connection,
// or an HTTP status that says the server is busy or
// broken (see IsTransientStatus). It is an allowlist: any other error, such as a
// missing version or bad credentials, is not transient.
func IsTransientError(err error) bool {
	var dnsErr *net.DNSError
	var opErr *net.OpError
	var netErr net.Error
	switch {
	case errors.As(err, &dnsErr) && dnsErr.IsNotFound:
		return false
	case errors.As(err, &opErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return true
	case isHTTP2ConnectionError(err):
		return true
	case errors.As(err, &netErr):
		return netErr.Timeout()
	}

	// go-git wraps the HTTP error, and its wrapper has no Unwrap method.
	var unexpected *plumbing.UnexpectedError
	if errors.As(err, &unexpected) {
		err = unexpected.Err
	}
	var httpErr *githttp.Err
	if errors.As(err, &httpErr) {
		return IsTransientStatus(httpErr.StatusCode())
	}

	return false
}

// isHTTP2ConnectionError reports whether err is an HTTP/2 stream reset, a GOAWAY or
// a lost connection. net/http bundles its HTTP/2 client and does not export the
// error types, so only the start of the message can tell them apart from other
// net.Error values.
func isHTTP2ConnectionError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return slices.ContainsFunc([]string{
		"stream error: stream ID ",
		"http2: client connection lost",
		"http2: server sent GOAWAY",
		"http2: Transport received GOAWAY",
	}, func(text string) bool { return strings.Contains(msg, text) })
}

// IsTransientStatus reports whether an HTTP status says that the server is busy or
// broken: 408, 429 and 5xx, except 501 and 505, which a retry does not fix.
func IsTransientStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	case http.StatusNotImplemented, http.StatusHTTPVersionNotSupported:
		return false
	}
	return status >= http.StatusInternalServerError
}
