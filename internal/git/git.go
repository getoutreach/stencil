// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for git

// Package git implements helpers for interacting with git
package git

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/getoutreach/stencil/internal/retry"
	"github.com/pkg/errors"
)

// This block contains errors and regexes.
var (
	// ErrNoHeadBranch is returned when a repository's HEAD (aka default) branch cannot
	// be determine.
	ErrNoHeadBranch = errors.New("failed to find a head branch, does one exist?")

	// ErrNoRemoteHeadBranch is returned when a repository's remote  default/HEAD branch
	// cannot be determined.
	ErrNoRemoteHeadBranch = errors.New("failed to get head branch from remote origin")

	// ErrNoOriginRemote is returned when a repository has no "origin" remote and its
	// default branch cannot be inferred from its local branches.
	ErrNoOriginRemote = errors.New(`the repository has no "origin" remote to determine the default branch from`)

	// headPattern is used to parse git output to determine the head branch. It does not
	// match git's "(unknown)" placeholder.
	headPattern = regexp.MustCompile(`HEAD branch: ([^\s(]\S*)`)
)

// GetDefaultBranch determines the default/HEAD branch for a given git
// repository. It asks the "origin" remote. A repository without an "origin"
// remote, such as one that was just created, has no remote to ask, so its default
// branch is inferred from its local branches instead, see inferLocalDefaultBranch.
func GetDefaultBranch(ctx context.Context, path string) (string, error) {
	remotes, err := runGit(ctx, path, "remote")
	if err != nil {
		return "", errors.Wrap(err, "failed to list the remotes")
	}
	if !slices.Contains(strings.Fields(remotes), "origin") {
		return inferLocalDefaultBranch(ctx, path)
	}

	out, err := runGit(ctx, path, "remote", "show", "origin")
	if err != nil {
		return "", errors.Wrap(err, "failed to get head branch from remote origin")
	}

	matches := headPattern.FindStringSubmatch(out)
	if len(matches) != 2 {
		return "", ErrNoRemoteHeadBranch
	}

	return matches[1], nil
}

// runGit runs git with args in path and returns its standard output.
func runGit(ctx context.Context, path string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = path
	// Keep git's output untranslated, so that it can be parsed.
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	return string(out), err
}

// inferLocalDefaultBranch infers the default branch of a repository that has no
// remote to ask. With exactly one local branch, that branch is the default. With no
// branches yet, HEAD names the branch that the first commit will create. With
// several branches there is no way to tell, so it returns ErrNoOriginRemote.
func inferLocalDefaultBranch(ctx context.Context, path string) (string, error) {
	// Use git instead of reading the repository, so that the branches of linked
	// worktrees are included.
	out, err := runGit(ctx, path, "for-each-ref", "--format=%(refname)", "refs/heads")
	if err != nil {
		return "", errors.Wrap(err, "failed to list the local branches")
	}
	refs := strings.Fields(out)

	switch len(refs) {
	case 1:
		return strings.TrimPrefix(refs[0], "refs/heads/"), nil
	case 0:
		head, err := runGit(ctx, path, "symbolic-ref", "--short", "HEAD")
		if err != nil {
			return "", errors.Wrap(ErrNoOriginRemote, "there are no branches and HEAD is detached")
		}
		return strings.TrimSpace(head), nil
	default:
		return "", errors.Wrapf(ErrNoOriginRemote, "found %d local branches, add an \"origin\" remote to choose one", len(refs))
	}
}

// GetDefaultBranchWithRetry is GetDefaultBranch, retried with backoff (3 attempts
// by default) because it asks the remote, which can fail transiently. It does not
// retry when asking again won't change the answer: there is no "origin" remote and
// the branch can't be inferred, or git ran fine but did not report a branch.
// opts are applied after the defaults.
func GetDefaultBranchWithRetry(ctx context.Context, path string, opts ...retry.Option) (string, error) {
	opts = append([]retry.Option{
		retry.WithBackoff(retry.Exponential(time.Second, 4*time.Second, 2, 0.2)),
	}, opts...)

	return retry.DoValue(ctx, func(ctx context.Context) (string, error) {
		db, err := GetDefaultBranch(ctx, path)
		if errors.Is(err, ErrNoOriginRemote) || errors.Is(err, ErrNoRemoteHeadBranch) {
			return "", retry.Permanent(err)
		}
		return db, err
	}, opts...)
}
