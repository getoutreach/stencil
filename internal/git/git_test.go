// Copyright 2025 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for the git package.

package git_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	stgit "github.com/getoutreach/stencil/internal/git"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"gotest.tools/v3/assert"
)

func createTempGitRepo(t *testing.T, defaultBranch plumbing.ReferenceName) string {
	t.Helper()
	repoDir := t.TempDir()
	repo, err := git.PlainInitWithOptions(repoDir, &git.PlainInitOptions{
		InitOptions: git.InitOptions{DefaultBranch: defaultBranch},
		Bare:        false,
	})
	assert.NilError(t, err)
	wt, err := repo.Worktree()
	assert.NilError(t, err)
	assert.NilError(t, os.WriteFile(filepath.Join(repoDir, "test.txt"), []byte("hello world"), 0o644))
	_, err = wt.Add("test.txt")
	assert.NilError(t, err)
	_, err = wt.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test Author",
			Email: "test@example.com",
		},
	})
	assert.NilError(t, err)
	return repoDir
}

// Makes sure that GetDefaultBranch works correctly even when the system language is not set to English.
func TestGetDefaultBranchDifferentOSLanguage(t *testing.T) {
	ctx := context.Background()

	repoDir := t.TempDir()
	_, err := git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
		URL: createTempGitRepo(t, plumbing.Main),
	})
	assert.NilError(t, err)

	t.Setenv("LC_ALL", "fr_FR.UTF-8")
	defaultBranch, err := stgit.GetDefaultBranch(ctx, repoDir)
	assert.NilError(t, err)
	assert.Equal(t, defaultBranch, "main", "Expected default branch to be 'main'")
}

func TestGetDefaultBranchMaster(t *testing.T) {
	ctx := context.Background()

	repoDir := t.TempDir()
	_, err := git.PlainCloneContext(ctx, repoDir, false, &git.CloneOptions{
		URL: createTempGitRepo(t, plumbing.Master),
	})
	assert.NilError(t, err)

	defaultBranch, err := stgit.GetDefaultBranch(ctx, repoDir)
	assert.NilError(t, err)
	assert.Equal(t, defaultBranch, "master", "Expected default branch to be 'master'")
}
