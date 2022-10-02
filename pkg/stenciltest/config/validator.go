// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains validator logic/struct

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/pkg/stenciltest/errors"
)

// Validator is a validator to be ran on a template matching
// specific criteria.
type Validator struct {
	// Command is the command to run to validate the template.
	// This command should exit with a non-zero exit code if the
	// template is invalid. The command will be ran with the path to
	// the template as the first argument, and from the root of the
	// repository.
	//
	// Example:
	//   Command: "goimports -w"
	Command string `yaml:"command"`

	// Extensions is a list of file extensions that this validator
	// should run on. If this is empty, this will be treated as "*".
	//
	// Note: Globs are supported.
	Extensions []string `yaml:"extensions"`

	// Path is a glob pattern that the template path must match.
	// If this is empty, this will be treated as "*".
	//
	// Note: This supports doublestar globs.
	Path string `yaml:"path"`

	// Func is a function to use to validate the template,
	// this is only usable when using the Go API. Cannot be set
	// when Command is set.
	Func ValidatorFunc `yaml:"-"`
}

// NewGoValidator creates a new validator that uses a Go function
// that runs on all files. This is a helper function for the Go API.
func NewGoValidator(f ValidatorFunc) Validator {
	return Validator{Func: f}
}

// ValidatorFunc is a function that validates a template.
type ValidatorFunc func(f *codegen.File) error

// Validate validates the template at the given path.
func (v Validator) Validate(t *testing.T, f *codegen.File) error {
	if v.Path == "" {
		v.Path = "**"
	}
	if len(v.Extensions) == 0 {
		v.Extensions = []string{"*"}
	}

	// Check if the path matches the glob
	if matched, err := doublestar.Match(v.Path, f.Name()); !matched || err != nil {
		return nil
	}

	// Check if the extension matches the extensions list
	matchedExtensions := false
	for _, ext := range v.Extensions {
		if matched, err := filepath.Match(ext, filepath.Ext(f.Name())); matched && err == nil {
			matchedExtensions = true
			break
		}
	}
	if !matchedExtensions {
		return nil
	}

	// If a function is set, use that instead of a command
	if v.Func != nil {
		return v.Func(f)
	}

	repoDir, err := GetRepositoryDirectory()
	if err != nil {
		return errors.Wrap(err, "failed to get repository directory")
	}

	// Create a temporary file to write the rendered template to
	// for the validator to run on.
	tempFile, err := os.CreateTemp(t.TempDir(), filepath.Base(f.Name()))
	if err != nil {
		return errors.Wrap(err, "failed to create temp file")
	}
	if _, err := tempFile.Write(f.Bytes()); err != nil {
		return errors.Wrap(err, "failed to write to temp file")
	}
	tempFile.Close()                 //nolint:errcheck // Why: Best effort
	defer os.Remove(tempFile.Name()) // Delete it in case t.TempDir() is not cleaned up

	// Run the validator against the file
	//nolint:gosec // Why: tests
	cmd := exec.Command("sh", "-c", v.Command+" "+tempFile.Name())
	cmd.Dir = repoDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
