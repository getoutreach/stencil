// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for retrying the resolution of an extension version.

package extensions

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestGetVersionWithCommitRetriesTransientErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		tag  string
	}{
		{"channel", "unstable"},
		{"constraint", "v1.0.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// The server fails once, and then says that the repository does not exist.
			var requests atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				status := http.StatusNotFound
				if requests.Add(1) == 1 {
					status = http.StatusServiceUnavailable
				}
				http.Error(w, "test error", status)
			}))
			t.Cleanup(srv.Close)

			_, err := getVersionWithCommit(t.Context(), logrus.New(), "", srv.URL+"/repo.git",
				&resolver.Version{Tag: tc.tag})

			assert.ErrorIs(t, err, transport.ErrRepositoryNotFound, "expected the error of the second attempt")
			assert.Equal(t, requests.Load(), int32(2))
		})
	}
}
