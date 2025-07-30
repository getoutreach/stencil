// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for git

// Package git implements helpers for interacting with git
package git

import (
	"context"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/pkg/errors"
)

// This block contains errors and regexes
var (
	// ErrNoHeadBranch is returned when a repository's HEAD (aka default) branch cannot
	// be determine
	ErrNoHeadBranch = errors.New("failed to find a head branch, does one exist?")

	// ErrNoRemoteHeadBranch is returned when a repository's remote  default/HEAD branch
	// cannot be determined.
	ErrNoRemoteHeadBranch = errors.New("failed to get head branch from remote origin")
)

// GetDefaultBranch determines the default/HEAD branch for a given git
// repository.
func GetDefaultBranch(ctx context.Context, path string) (string, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to open git repository")
	}
	origin, err := repo.Remote("origin")
	if err != nil {
		return "", errors.Wrap(err, "failed to get remote origin")
	}
	refs, err := origin.ListContext(ctx, &git.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to list remote refs for origin")
	}
	var headRef *plumbing.Reference
	for _, ref := range refs {
		if ref.Type() == plumbing.SymbolicReference && ref.Name() == plumbing.HEAD {
			headRef = ref
			break
		}
	}
	if headRef == nil {
		return "", ErrNoRemoteHeadBranch
	}
	return headRef.Target().Short(), nil
}
