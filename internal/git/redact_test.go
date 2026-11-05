// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for removing credentials from URLs.

package git_test

import (
	"errors"
	"fmt"
	"testing"

	stgit "github.com/getoutreach/stencil/internal/git"
	"gotest.tools/v3/assert"
)

func TestRedactURL(t *testing.T) {
	for _, tc := range []struct {
		name, url, want string
	}{
		{"no user information", "https://github.com/org/repo", "https://github.com/org/repo"},
		{"token as the user name", "https://ghp_secret@github.com/org/repo", "https://github.com/org/repo"},
		{"user name and password", "https://x-access-token:ghp_secret@github.com/org/repo", "https://github.com/org/repo"},
		{"an @ in the path is kept", "https://github.com/org/repo@v1", "https://github.com/org/repo@v1"},
		{"an @ in the query is kept", "https://github.com/org/repo?a=b@c", "https://github.com/org/repo?a=b@c"},
		{"scp style address", "git@github.com:org/repo.git", "git@github.com:org/repo.git"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, stgit.RedactURL(tc.url), tc.want)
		})
	}
}

var errNotFound = errors.New("not found")

func TestRedactError(t *testing.T) {
	assert.NilError(t, stgit.RedactError(nil))

	err := stgit.RedactError(fmt.Errorf("failed to clone %q: %w", "https://ghp_secret@github.com/org/repo", errNotFound))

	assert.Equal(t, err.Error(), `failed to clone "https://github.com/org/repo": not found`)
	assert.ErrorIs(t, err, errNotFound, "expected the original error to be found through the redacted one")
}
