// Copyright 2025 Outreach Corporation. All Rights Reserved.

// Description: Unit tests for the git package.

package git_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	stgit "github.com/getoutreach/stencil/internal/git"
	"github.com/getoutreach/stencil/internal/retry"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
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

// createTwoBranchRepo is createTempGitRepo with a second branch, so that the
// default branch cannot be inferred without an "origin" remote.
func createTwoBranchRepo(t *testing.T) string {
	t.Helper()
	repoDir := createTempGitRepo(t, plumbing.Main)
	repo, err := git.PlainOpen(repoDir)
	assert.NilError(t, err)
	head, err := repo.Head()
	assert.NilError(t, err)
	assert.NilError(t, repo.Storer.SetReference(plumbing.NewHashReference(plumbing.NewBranchReferenceName("other"), head.Hash())))
	return repoDir
}

func TestGetDefaultBranchWithoutOriginRemote(t *testing.T) {
	t.Run("infers the only branch", func(t *testing.T) {
		repoDir := createTempGitRepo(t, plumbing.NewBranchReferenceName("develop"))
		defaultBranch, err := stgit.GetDefaultBranch(t.Context(), repoDir)
		assert.NilError(t, err)
		assert.Equal(t, defaultBranch, "develop")
	})

	t.Run("uses the branch HEAD names when there are no commits yet", func(t *testing.T) {
		repoDir := t.TempDir()
		_, err := git.PlainInitWithOptions(repoDir, &git.PlainInitOptions{
			InitOptions: git.InitOptions{DefaultBranch: plumbing.NewBranchReferenceName("develop")},
		})
		assert.NilError(t, err)

		defaultBranch, err := stgit.GetDefaultBranch(t.Context(), repoDir)
		assert.NilError(t, err)
		assert.Equal(t, defaultBranch, "develop")
	})

	t.Run("fails when HEAD is detached and there are no branches", func(t *testing.T) {
		repoDir := createTempGitRepo(t, plumbing.Main)
		repo, err := git.PlainOpen(repoDir)
		assert.NilError(t, err)
		head, err := repo.Head()
		assert.NilError(t, err)
		assert.NilError(t, repo.Storer.SetReference(plumbing.NewHashReference(plumbing.HEAD, head.Hash())))
		assert.NilError(t, repo.Storer.RemoveReference(plumbing.Main))

		_, err = stgit.GetDefaultBranch(t.Context(), repoDir)
		assert.ErrorIs(t, err, stgit.ErrNoOriginRemote)
		assert.ErrorContains(t, err, "detached")
	})

	t.Run("sees the branches of other worktrees", func(t *testing.T) {
		repoDir := createTempGitRepo(t, plumbing.Main)
		worktreeDir := filepath.Join(t.TempDir(), "worktree")
		gitInDir(t, repoDir, "worktree", "add", "-q", "-b", "other", worktreeDir)

		_, err := stgit.GetDefaultBranch(t.Context(), worktreeDir)
		assert.ErrorIs(t, err, stgit.ErrNoOriginRemote)
		assert.ErrorContains(t, err, "2 local branches")
	})

	t.Run("fails when there are several branches", func(t *testing.T) {
		_, err := stgit.GetDefaultBranch(t.Context(), createTwoBranchRepo(t))
		assert.ErrorIs(t, err, stgit.ErrNoOriginRemote)
		assert.ErrorContains(t, err, "2 local branches")
	})
}

func TestGetDefaultBranchWithRetry(t *testing.T) {
	fast := retry.WithBackoff(retry.Exponential(time.Millisecond, time.Millisecond, 1, 0))

	t.Run("retries when git fails", func(t *testing.T) {
		repoDir := createTempGitRepo(t, plumbing.Main)
		// An unreachable origin makes `git remote show origin` fail every time.
		repo, err := git.PlainOpen(repoDir)
		assert.NilError(t, err)
		_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{filepath.Join(t.TempDir(), "missing")}})
		assert.NilError(t, err)

		retries := 0
		_, err = stgit.GetDefaultBranchWithRetry(t.Context(), repoDir, fast,
			retry.WithOnRetry(func(context.Context, int, error, time.Duration) { retries++ }))
		var exitErr *exec.ExitError
		assert.Assert(t, errors.As(err, &exitErr), "expected git to exit with an error, got: %v", err)
		assert.Equal(t, retries, 2, "expected 3 attempts, so 2 retries")
	})

	t.Run("does not retry without an origin remote", func(t *testing.T) {
		retries := 0
		_, err := stgit.GetDefaultBranchWithRetry(t.Context(), createTwoBranchRepo(t), fast,
			retry.WithOnRetry(func(context.Context, int, error, time.Duration) { retries++ }))
		assert.ErrorIs(t, err, stgit.ErrNoOriginRemote)
		assert.Equal(t, retries, 0)
	})

	t.Run("returns the branch", func(t *testing.T) {
		repoDir := t.TempDir()
		_, err := git.PlainCloneContext(t.Context(), repoDir, false, &git.CloneOptions{
			URL: createTempGitRepo(t, plumbing.Master),
		})
		assert.NilError(t, err)

		db, err := stgit.GetDefaultBranchWithRetry(t.Context(), repoDir, fast)
		assert.NilError(t, err)
		assert.Equal(t, db, "master")
	})
}

func TestGetDefaultBranchWithNonLetterCharacters(t *testing.T) {
	for _, branch := range []string{"release-1", "1.x", "team/feature_2", "_main"} {
		t.Run(branch, func(t *testing.T) {
			repoDir := t.TempDir()
			_, err := git.PlainCloneContext(t.Context(), repoDir, false, &git.CloneOptions{
				URL: createTempGitRepo(t, plumbing.NewBranchReferenceName(branch)),
			})
			assert.NilError(t, err)

			defaultBranch, err := stgit.GetDefaultBranch(t.Context(), repoDir)
			assert.NilError(t, err)
			assert.Equal(t, defaultBranch, branch)
		})
	}
}

// gitInDir runs git with args in dir and fails the test on error.
func gitInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	assert.NilError(t, err, string(out))
}

func TestGetDefaultBranchUnknown(t *testing.T) {
	upstream := createTempGitRepo(t, plumbing.Main)
	// Point the remote's HEAD at a branch that does not exist.
	gitInDir(t, upstream, "symbolic-ref", "HEAD", "refs/heads/missing")

	repoDir := filepath.Join(t.TempDir(), "clone")
	gitInDir(t, filepath.Dir(repoDir), "clone", "-q", upstream, repoDir)

	_, err := stgit.GetDefaultBranch(t.Context(), repoDir)
	assert.ErrorIs(t, err, stgit.ErrNoRemoteHeadBranch)
}
