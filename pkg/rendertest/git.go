// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the git fixtures.

package rendertest

// gitConfig holds the .Git values that templates see.
type gitConfig struct {
	// ref is the current ref, in the refs/<type>/<name> format
	ref string

	// commit is the current commit
	commit string

	// defaultBranch is the default branch of the repository
	defaultBranch string

	// dirty is whether the working tree has changes
	dirty bool
}

// GitOption sets a .Git value that templates see.
type GitOption func(*gitConfig)

// Ref sets .Git.Ref, in the refs/<type>/<name> format, for example
// "refs/heads/main".
func Ref(ref string) GitOption {
	return func(c *gitConfig) {
		c.ref = ref
	}
}

// Commit sets .Git.Commit.
func Commit(sha string) GitOption {
	return func(c *gitConfig) {
		c.commit = sha
	}
}

// DefaultBranch sets the value of .Git.DefaultBranch.
func DefaultBranch(branch string) GitOption {
	return func(c *gitConfig) {
		c.defaultBranch = branch
	}
}

// Dirty makes .Git.Dirty true.
func Dirty() GitOption {
	return func(c *gitConfig) {
		c.dirty = true
	}
}

// Clean makes .Git.Dirty false.
func Clean() GitOption {
	return func(c *gitConfig) {
		c.dirty = false
	}
}

// Release sets .Git.Ref to the tag, as "refs/tags/<tag>", and makes
// .Git.Dirty false.
func Release(tag string) GitOption {
	return func(c *gitConfig) {
		c.ref = "refs/tags/" + tag
		c.dirty = false
	}
}
