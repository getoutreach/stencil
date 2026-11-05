// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file classifies the errors of git remote operations.

package git

import (
	"errors"
	"io"
	"net"
	"net/http"

	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
)

// IsTransientError reports whether err from an operation on a git remote, such as
// a clone or a ls-remote, could go away if the operation is tried again. That is a
// connection error other than a missing host, a timeout, a response that ended
// early (io.EOF or io.ErrUnexpectedEOF), or an HTTP status that says the server is
// busy or broken (408, 429 and 5xx).
//
// It is an allowlist: an error that it does not know, such as a version that does
// not exist or bad credentials, is not transient, so a retry does not wait for an
// answer that will not change.
func IsTransientError(err error) bool {
	var dnsErr *net.DNSError
	var opErr *net.OpError
	var netErr net.Error
	switch {
	case errors.As(err, &dnsErr) && dnsErr.IsNotFound:
		return false
	case errors.As(err, &opErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
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

// IsTransientStatus reports whether an HTTP response with status says that the
// server is busy or broken, so that the same request could work later: 408, 429
// and 5xx.
func IsTransientStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}
