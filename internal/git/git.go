// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for git

// Package git implements helpers for interacting with git
package git

import (
	"context"
	"os/exec"
	"regexp"

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

	// headPattern is used to parse git output to determine the head branch
	headPattern = regexp.MustCompile(`HEAD branch: ([[:alpha:]]+)`)
)

// GetDefaultBranch determines the default/HEAD branch for a given git
// repository.
func GetDefaultBranch(ctx context.Context, path string) (string, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to open directory as a repository")
	}

	_, err = r.Remote("origin")
	if err != nil {
		// loop through the local branchs
		candidates := []string{"main", "master"}
		for _, branch := range candidates {
			_, err := r.Reference(plumbing.NewBranchReferenceName(branch), true) //nolint:govet
			if err == nil {
				return branch, nil
			}
		}

		// we couldn't find one
		return "", ErrNoHeadBranch
	}

	// we found an origin reference, figure out the HEAD
	cmd := exec.CommandContext(ctx, "git", "remote", "show", "origin")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get head branch from remote origin")
	}

	matches := headPattern.FindStringSubmatch(string(out))
	if len(matches) != 2 {
		return "", ErrNoRemoteHeadBranch
	}

	return matches[1], nil
}
