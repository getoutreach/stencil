// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for git

// Package git implements helpers for interacting with git
package git

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/getoutreach/stencil/internal/retry"
	gogit "github.com/go-git/go-git/v5"
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

	// ErrNoOriginRemote is returned when a repository has no "origin" remote, which
	// is needed to determine its default branch.
	ErrNoOriginRemote = errors.New(`the repository has no "origin" remote to determine the default branch from`)

	// headPattern is used to parse git output to determine the head branch.
	headPattern = regexp.MustCompile(`HEAD branch: ([[:alpha:]]+)`)
)

// GetDefaultBranch determines the default/HEAD branch for a given git
// repository.
func GetDefaultBranch(ctx context.Context, path string) (string, error) {
	// If the repository can't be opened here, let git itself report why.
	if repo, err := gogit.PlainOpen(path); err == nil {
		if _, err := repo.Remote("origin"); errors.Is(err, gogit.ErrRemoteNotFound) {
			return "", ErrNoOriginRemote
		}
	}

	cmd := exec.CommandContext(ctx, "git", "remote", "show", "origin")
	cmd.Dir = path
	env := os.Environ()
	env = append(env, "LC_ALL=C")
	cmd.Env = env
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to get head branch from remote origin")
	}

	matches := headPattern.FindStringSubmatch(string(out))
	if len(matches) != 2 {
		return "", ErrNoRemoteHeadBranch
	}

	return matches[1], nil
}

// GetDefaultBranchWithRetry is GetDefaultBranch, retried with backoff (3 attempts
// by default) because it asks the remote, which can fail transiently. It does not
// retry when asking again won't change the answer: there is no "origin" remote,
// or git ran fine but did not report a branch. opts are applied after the defaults.
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
