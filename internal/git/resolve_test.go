// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for resolving the version of a git repository.

package git_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	stgit "github.com/getoutreach/stencil/internal/git"
	"github.com/getoutreach/stencil/internal/retry"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

// resolveWith resolves the version of a repository that does not exist. The
// server answers 404, but with 503 first if transientFirst. It returns the number
// of requests and the error.
func resolveWith(t *testing.T, transientFirst bool) (int32, error) {
	t.Helper()

	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusNotFound
		if requests.Add(1) == 1 && transientFirst {
			status = http.StatusServiceUnavailable
		}
		http.Error(w, "test error", status)
	}))
	t.Cleanup(srv.Close)

	_, err := stgit.ResolveVersion(t.Context(), logrus.New(), "", resolver.Criteria{URL: srv.URL + "/repo.git"},
		retry.WithBackoff(retry.Constant(time.Millisecond)))
	return requests.Load(), err
}

func TestResolveVersionRetriesTransientErrors(t *testing.T) {
	requests, err := resolveWith(t, true)

	assert.ErrorIs(t, err, transport.ErrRepositoryNotFound, "expected the error of the second attempt")
	assert.Equal(t, requests, int32(2))
}

func TestResolveVersionDoesNotRetryOtherErrors(t *testing.T) {
	requests, err := resolveWith(t, false)

	assert.ErrorIs(t, err, transport.ErrRepositoryNotFound)
	assert.Equal(t, requests, int32(1), "expected no retry, because asking again gives the same answer")
}
