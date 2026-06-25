// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the stencil lint command.

package main

import (
	"bytes"
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

func TestFailIfFindingsInfoOnlyPasses(t *testing.T) {
	infoOnly := []lint.Finding{{Severity: lint.SeverityInfo, Path: "arguments.x", Message: "i"}}
	// Info never fails, regardless of warnings-as-errors.
	assert.NilError(t, failIfFindings(infoOnly, true))
	assert.NilError(t, failIfFindings(infoOnly, false))
}

func TestFailManifestInfoOnlyLogsBareValidLine(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	// An info-only finding set must pass and log the bare "is valid" success
	// line: info findings are not counted as warnings, so the "(N warning(s))"
	// form must not appear.
	infoOnly := []lint.Finding{{Severity: lint.SeverityInfo, Path: "arguments.x", Message: "deprecated msg"}}
	err := failManifest(log, "manifest.yaml", infoOnly, true)
	assert.NilError(t, err)

	out := buf.String()
	assert.Assert(t, strings.Contains(out, "is valid"), "expected success line, got: %s", out)
	assert.Assert(t, !strings.Contains(out, "warning(s)"),
		"info-only findings must not log the warning-count form, got: %s", out)
}

func TestLogFindingsRoutesInfoToInfoLevel(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	logFindings(log, []lint.Finding{
		{Severity: lint.SeverityInfo, Path: "arguments.x", Message: "deprecated msg"},
	})

	out := buf.String()
	// logrus default text formatter writes level=info for Info(); it would be
	// level=error if the info case were missing (the defensive default).
	assert.Assert(t, strings.Contains(out, "level=info"),
		"expected info-level log, got: %s", out)
	assert.Assert(t, strings.Contains(out, "deprecated msg"))
}

func TestLogFindingsIncludesLineWhenSet(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	logFindings(log, []lint.Finding{
		{Severity: lint.SeverityWarning, Path: "arguments.x.type", Message: "dep", Line: 4},
		{Severity: lint.SeverityError, Path: "manifest.yaml", Message: "empty"}, // Line 0
	})

	out := buf.String()
	// The finding with a line emits a line field...
	assert.Assert(t, strings.Contains(out, "line=4"),
		"expected line field for the lined finding, got: %s", out)
	// ...and the zero-line finding does not.
	assert.Assert(t, !strings.Contains(out, "line=0"),
		"zero-line findings must not emit a line field, got: %s", out)
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
