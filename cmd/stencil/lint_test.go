// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the stencil lint command.

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/internal/lint"
)

func discardLogger() *logrus.Logger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func TestRunManifestReaderValid(t *testing.T) {
	findings, err := runManifestReader(discardLogger(), "<test>", strings.NewReader("name: testing\n"))
	assert.NilError(t, err)
	assert.Equal(t, 0, len(findings))
}

func TestRunManifestReaderInvalid(t *testing.T) {
	// An unknown top-level key is a strict-decode error → at least one finding.
	findings, err := runManifestReader(discardLogger(), "<test>", strings.NewReader("name: testing\nnme: oops\n"))
	assert.NilError(t, err)
	assert.Assert(t, len(findings) > 0)
}

func TestFailIfFindingsPolicy(t *testing.T) {
	warnOnly := []lint.Finding{{Severity: lint.SeverityWarning, Path: "x", Message: "w"}}
	errOnly := []lint.Finding{{Severity: lint.SeverityError, Path: "y", Message: "e"}}

	assert.NilError(t, failIfFindings(nil, true))
	assert.NilError(t, failIfFindings(nil, false))
	assert.Error(t, failIfFindings(warnOnly, true),
		"manifest validation failed: 0 error(s), 1 warning(s)")
	assert.NilError(t, failIfFindings(warnOnly, false))
	assert.Error(t, failIfFindings(errOnly, true),
		"manifest validation failed: 1 error(s), 0 warning(s)")
	assert.Error(t, failIfFindings(errOnly, false),
		"manifest validation failed: 1 error(s), 0 warning(s)")
}

func TestResolveManifestReaderMissing(t *testing.T) {
	r, closer, finding, err := resolveManifestReader(filepath.Join(t.TempDir(), "nope.yaml"))
	assert.NilError(t, err)
	assert.Assert(t, r == nil)
	assert.Assert(t, closer == nil)
	assert.Assert(t, finding != nil)
	assert.Equal(t, lint.SeverityError, finding.Severity)
	assert.Assert(t, strings.Contains(finding.Message, "manifest file not found"))
}

func TestResolveManifestReaderDirAppendsManifest(t *testing.T) {
	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte("name: testing\n"), 0o600))
	r, closer, finding, err := resolveManifestReader(dir)
	assert.NilError(t, err)
	assert.Assert(t, finding == nil)
	assert.Assert(t, r != nil)
	if closer != nil {
		defer closer.Close()
	}
	b, _ := io.ReadAll(r)
	assert.Assert(t, strings.Contains(string(b), "name: testing"))
}

func TestManifestRunnerValid(t *testing.T) {
	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte("name: testing\n"), 0o600))
	findings, err := manifestRunner(filepath.Join(dir, "manifest.yaml"))(discardLogger())
	assert.NilError(t, err)
	assert.Equal(t, 0, len(findings))
}

func TestManifestRunnerMissingFileIsFinding(t *testing.T) {
	findings, err := manifestRunner(filepath.Join(t.TempDir(), "manifest.yaml"))(discardLogger())
	assert.NilError(t, err) // missing file is a finding, not an error
	assert.Assert(t, len(findings) == 1)
	assert.Equal(t, lint.SeverityError, findings[0].Severity)
}

func TestNewLintCommandShape(t *testing.T) {
	cmd := NewLintCommand()
	assert.Equal(t, "lint", cmd.Name)
	// the module-manifest subcommand exists
	var hasManifest bool
	for _, sub := range cmd.Commands {
		if sub.Name == "module-manifest" {
			hasManifest = true
		}
	}
	assert.Assert(t, hasManifest)
	// warnings-as-errors flag present on both the group and the subcommand
	assert.Assert(t, flagPresent(cmd.Flags, "warnings-as-errors"))
	for _, sub := range cmd.Commands {
		if sub.Name == "module-manifest" {
			assert.Assert(t, flagPresent(sub.Flags, "warnings-as-errors"))
		}
	}
}

func flagPresent(flags []cli.Flag, name string) bool {
	for _, fl := range flags {
		for _, n := range fl.Names() {
			if n == name {
				return true
			}
		}
	}
	return false
}
