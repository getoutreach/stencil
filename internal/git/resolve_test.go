// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for resolving the version of a git repository.

package git_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
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

var fastRetry = retry.WithBackoff(retry.Constant(time.Millisecond))

// statusServer serves a repository that does not exist. It answers 404, except that
// the first request gets a 503 when transientFirst is set.
func statusServer(t *testing.T, transientFirst bool) (repoURL string, requests *atomic.Int32) {
	t.Helper()

	requests = new(atomic.Int32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		status := http.StatusNotFound
		if requests.Add(1) == 1 && transientFirst {
			status = http.StatusServiceUnavailable
		}
		http.Error(w, "test error", status)
	}))
	t.Cleanup(srv.Close)

	return srv.URL + "/repo.git", requests
}

func TestResolveVersionRetriesTransientErrors(t *testing.T) {
	repoURL, requests := statusServer(t, true)

	_, err := stgit.ResolveVersion(t.Context(), logrus.New(), "", resolver.Criteria{URL: repoURL}, fastRetry)

	assert.ErrorIs(t, err, transport.ErrRepositoryNotFound, "expected the error of the second attempt")
	assert.Equal(t, requests.Load(), int32(2))
}

func TestResolveVersionDoesNotRetryOtherErrors(t *testing.T) {
	repoURL, requests := statusServer(t, false)

	_, err := stgit.ResolveVersion(t.Context(), logrus.New(), "", resolver.Criteria{URL: repoURL}, fastRetry)

	assert.ErrorIs(t, err, transport.ErrRepositoryNotFound)
	assert.Equal(t, requests.Load(), int32(1), "expected no retry, because asking again gives the same answer")
}

func TestResolveVersionDoesNotLogCredentials(t *testing.T) {
	repoURL, _ := statusServer(t, true)
	var logged bytes.Buffer
	log := logrus.New()
	log.Out = &logged

	withToken := "http://ghp_secret@" + strings.TrimPrefix(repoURL, "http://")
	_, err := stgit.ResolveVersion(t.Context(), log, "", resolver.Criteria{URL: withToken}, fastRetry)

	assert.Assert(t, strings.Contains(logged.String(), "retrying"), "expected a warning for the retry")
	assert.Assert(t, !strings.Contains(logged.String()+err.Error(), "ghp_secret"), "the log or the error has the token:\n%s\n%v", &logged, err)
}
