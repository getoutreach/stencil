// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the stencil lint command.

package main

import (
	"bytes"
	"context"
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
		"lint failed: 0 error(s), 1 warning(s)")
	assert.NilError(t, failIfFindings(warnOnly, false))
	assert.Error(t, failIfFindings(errOnly, true),
		"lint failed: 1 error(s), 0 warning(s)")
	assert.Error(t, failIfFindings(errOnly, false),
		"lint failed: 1 error(s), 0 warning(s)")
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

func TestManifestRunnerMissingManifestSkipped(t *testing.T) {
	findings, err := manifestRunner(filepath.Join(t.TempDir(), "manifest.yaml"))(discardLogger())
	assert.NilError(t, err) // a missing manifest is skipped, not an error
	assert.Equal(t, 0, len(findings), "aggregate lint skips a missing manifest (module may be templates-only)")
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
	// --fix flag present on both the group and the subcommand.
	assert.Assert(t, flagPresent(cmd.Flags, "fix"))
	for _, sub := range cmd.Commands {
		if sub.Name == "module-manifest" {
			assert.Assert(t, flagPresent(sub.Flags, "fix"))
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

// runModuleManifest invokes the real `lint module-manifest` action via the
// command tree, with the given trailing args, the --fix flag, and an optional
// stdin reader / stdout writer. It returns the command's error (nil means exit
// 0). Effects are asserted via the file, stdout, and this error; the action
// builds its own logger, so logger text is not captured here.
//
// args[0] is the root command's own name ("lint"), per urfave/cli/v3's
// Command.Run convention (args[0] is consumed as the program/command name).
// The reader/writer are set on the module-manifest subcommand (the command that
// runs the action); urfave/cli/v3 defaults each command's Reader/Writer
// independently and does not inherit them from the parent.
func runModuleManifest(t *testing.T, args []string, fix bool,
	stdin io.Reader, stdout io.Writer) error {
	t.Helper()
	root := NewLintCommand()
	for _, sub := range root.Commands {
		if sub.Name == "module-manifest" {
			sub.Writer = stdout
			if stdin != nil {
				sub.Reader = stdin
			}
		}
	}

	fullArgs := []string{"lint", "module-manifest"}
	if fix {
		fullArgs = append(fullArgs, "--fix")
	}
	fullArgs = append(fullArgs, args...)

	return root.Run(t.Context(), fullArgs)
}

func TestRunLintModuleManifestFixInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	assert.NilError(t, os.WriteFile(path,
		[]byte("name: m\narguments:\n  x:\n    type: string\n"), 0o600))

	err := runModuleManifest(t, []string{path}, true, nil, io.Discard)
	assert.NilError(t, err) // the only finding was a fixable warning → exit 0

	out, readErr := os.ReadFile(path)
	assert.NilError(t, readErr)
	assert.Assert(t, strings.Contains(string(out), "schema:"),
		"file should be rewritten with schema, got:\n%s", string(out))
	assert.Assert(t, !strings.Contains(string(out), "\n    type:"),
		"deprecated type should be gone, got:\n%s", string(out))
}

func TestRunLintModuleManifestFixLeavesUnfixable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	// A fixable warning (type) AND an unfixable error (bad type token).
	assert.NilError(t, os.WriteFile(path,
		[]byte("name: m\ntype: bogus\narguments:\n  x:\n    type: string\n"), 0o600))

	err := runModuleManifest(t, []string{path}, true, nil, io.Discard)
	assert.Assert(t, err != nil, "remaining error must fail the run")

	out, _ := os.ReadFile(path)
	assert.Assert(t, strings.Contains(string(out), "schema:"),
		"the fixable warning should still have been applied")
}

func TestRunLintModuleManifestFixNoOpDoesNotRewrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	clean := []byte("name: m\n")
	assert.NilError(t, os.WriteFile(path, clean, 0o600))

	info1, _ := os.Stat(path)
	err := runModuleManifest(t, []string{path}, true, nil, io.Discard)
	assert.NilError(t, err)

	out, _ := os.ReadFile(path)
	assert.Equal(t, string(clean), string(out)) // unchanged bytes
	info2, _ := os.Stat(path)
	assert.Equal(t, info1.ModTime(), info2.ModTime()) // not rewritten
}

func TestRunLintModuleManifestFixStdin(t *testing.T) {
	in := strings.NewReader("name: m\narguments:\n  x:\n    type: string\n")
	var stdout bytes.Buffer
	err := runModuleManifest(t, []string{"-"}, true, in, &stdout)
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(stdout.String(), "schema:"),
		"fixed YAML must be written to stdout, got:\n%s", stdout.String())
}

func TestRunLintAggregateFixInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	assert.NilError(t, os.WriteFile(path,
		[]byte("name: m\nmodules:\n  - name: dep\n    prerelease: true\n"), 0o600))

	// stencil lint <dir> --fix. The aggregate action runs on the root `lint`
	// command itself, so args[0] is "lint" (its own name, per urfave/cli/v3's
	// Command.Run convention) and the Writer is set on the root.
	root := NewLintCommand()
	root.Writer = io.Discard
	err := root.Run(t.Context(),
		[]string{"lint", "--fix", dir})
	assert.NilError(t, err) // prerelease warning fixed → exit 0

	out, _ := os.ReadFile(path)
	assert.Assert(t, strings.Contains(string(out), "channel: rc"),
		"aggregate --fix should migrate prerelease, got:\n%s", string(out))
	assert.Assert(t, !strings.Contains(string(out), "prerelease:"))
}

// TestRunLintFixMissingManifest pins the --fix not-found handling. The explicit
// module-manifest --fix path reports the "manifest file not found" finding (via
// the shared resolveManifestPath) and fails, since the user asked for a
// manifest. The aggregate --fix path instead SKIPS a missing manifest and exits
// cleanly, because a module may be templates-only.
func TestRunLintFixMissingManifest(t *testing.T) {
	t.Run("module-manifest", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope.yaml")
		err := runModuleManifest(t, []string{missing}, true, nil, io.Discard)
		assert.Assert(t, err != nil, "missing manifest must fail")
		assert.Assert(t, strings.Contains(err.Error(), "1 error(s)"),
			"expected the not-found finding to fail the run, got: %v", err)
	})

	t.Run("aggregate dir without manifest", func(t *testing.T) {
		dir := t.TempDir() // no manifest.yaml inside
		root := NewLintCommand()
		root.Writer = io.Discard
		err := root.Run(t.Context(), []string{"lint", "--fix", dir})
		assert.NilError(t, err) // aggregate --fix skips a missing manifest
	})
}

func TestNewLintCommandHasTemplatesSubcommand(t *testing.T) {
	cmd := NewLintCommand()
	var hasTemplates bool
	for _, sub := range cmd.Commands {
		if sub.Name == "templates" {
			hasTemplates = true
			assert.Assert(t, flagPresent(sub.Flags, "warnings-as-errors"))
		}
	}
	assert.Assert(t, hasTemplates)
}

func TestTemplateRunnerFindsBadTemplate(t *testing.T) {
	dir := t.TempDir()
	tdir := filepath.Join(dir, "templates")
	assert.NilError(t, os.MkdirAll(tdir, 0o750))
	// good: has file.Block; bad: block without file.Block.
	good := "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block>>\n"
	bad := "## <<Stencil::Block(y)>>\nnope\n## <</Stencil::Block>>\n"
	assert.NilError(t, os.WriteFile(filepath.Join(tdir, "good.tpl"), []byte(good), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(tdir, "bad.tpl"), []byte(bad), 0o600))

	findings, err := templateRunner(tdir)(discardLogger())
	assert.NilError(t, err)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, lint.SeverityError, findings[0].Severity)
	assert.Assert(t, strings.Contains(findings[0].Path, "bad.tpl"))
}

func TestTemplateRunnerEmptyDirIsClean(t *testing.T) {
	findings, err := templateRunner(t.TempDir())(discardLogger())
	assert.NilError(t, err) // no .tpl files -> nothing to lint
	assert.Equal(t, 0, len(findings))
}

func TestRunTemplateFileLogsDebug(t *testing.T) {
	dir := t.TempDir()
	good := "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block>>\n"
	path := filepath.Join(dir, "a.tpl")
	assert.NilError(t, os.WriteFile(path, []byte(good), 0o600))

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	_, err := runTemplateFile(log, path)
	assert.NilError(t, err)

	out := buf.String()
	assert.Assert(t, strings.Contains(out, "linting template"),
		"expected debug log line, got: %s", out)
	assert.Assert(t, strings.Contains(out, path),
		"expected the template path in the log, got: %s", out)
	assert.Assert(t, strings.Contains(out, "level=debug"),
		"expected debug level, got: %s", out)
}

func TestRunTemplateFileDebugSuppressedAtInfoLevel(t *testing.T) {
	dir := t.TempDir()
	good := "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block>>\n"
	path := filepath.Join(dir, "a.tpl")
	assert.NilError(t, os.WriteFile(path, []byte(good), 0o600))

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.InfoLevel)

	_, err := runTemplateFile(log, path)
	assert.NilError(t, err)

	out := buf.String()
	assert.Assert(t, !strings.Contains(out, "linting template"),
		"debug line must be suppressed at info level, got: %s", out)
}

func TestCollectTemplateFilesLogsDiscoveryCount(t *testing.T) {
	dir := t.TempDir()
	tdir := filepath.Join(dir, "templates")
	assert.NilError(t, os.MkdirAll(tdir, 0o750))
	good := "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block>>\n"
	assert.NilError(t, os.WriteFile(filepath.Join(tdir, "a.tpl"), []byte(good), 0o600))
	assert.NilError(t, os.WriteFile(filepath.Join(tdir, "b.tpl"), []byte(good), 0o600))

	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	files, err := collectTemplateFiles(log, tdir)
	assert.NilError(t, err)
	assert.Equal(t, len(files), 2)

	out := buf.String()
	assert.Assert(t, strings.Contains(out, "discovered 2 template(s)"),
		"expected discovery count log line, got: %s", out)
	assert.Assert(t, strings.Contains(out, "level=debug"),
		"expected debug level, got: %s", out)
}

func TestCollectTemplateFilesLogsMissingDir(t *testing.T) {
	var buf bytes.Buffer
	log := logrus.New()
	log.SetOutput(&buf)
	log.SetLevel(logrus.DebugLevel)

	files, err := collectTemplateFiles(log, filepath.Join(t.TempDir(), "nope"))
	assert.NilError(t, err)
	assert.Assert(t, files == nil, "expected nil files, got: %v", files)

	out := buf.String()
	assert.Assert(t, strings.Contains(out, "no templates directory"),
		"expected missing-dir log line, got: %s", out)
}

func TestRunLintTemplatesStdin(t *testing.T) {
	// A bad template (block without file.Block) piped via c.Reader with '-'.
	bad := "## <<Stencil::Block(y)>>\nnope\n## <</Stencil::Block>>\n"
	var out bytes.Buffer
	cmd := NewLintCommand()
	// Drive the templates subcommand directly with '-' and a piped reader.
	var sub *cli.Command
	for _, s := range cmd.Commands {
		if s.Name == "templates" {
			sub = s
		}
	}
	assert.Assert(t, sub != nil)
	sub.Reader = strings.NewReader(bad)
	sub.Writer = &out
	// warnings-as-errors default true; the rule-1 error must fail the run.
	err := sub.Run(context.Background(), []string{"templates", "-"})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "lint failed"))
}

// TestAggregateTemplatesOnlyModulePasses proves a templates-only module (valid
// templates, no manifest.yaml) lints cleanly through the real aggregate action.
func TestAggregateTemplatesOnlyModulePasses(t *testing.T) {
	dir := t.TempDir()
	tdir := filepath.Join(dir, "templates")
	assert.NilError(t, os.MkdirAll(tdir, 0o750))
	good := "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block>>\n"
	assert.NilError(t, os.WriteFile(filepath.Join(tdir, "a.tpl"), []byte(good), 0o600))

	root := NewLintCommand()
	root.Writer = io.Discard
	err := root.Run(context.Background(), []string{"lint", dir})
	assert.NilError(t, err) // no manifest is fine; valid templates pass → exit 0
}

// TestAggregateTemplatesOnlyModuleStillLintsTemplates proves templates are still
// validated even when the module has no manifest: an invalid template fails.
func TestAggregateTemplatesOnlyModuleStillLintsTemplates(t *testing.T) {
	dir := t.TempDir()
	tdir := filepath.Join(dir, "templates")
	assert.NilError(t, os.MkdirAll(tdir, 0o750))
	bad := "## <<Stencil::Block(x)>>\nno file block\n## <</Stencil::Block>>\n"
	assert.NilError(t, os.WriteFile(filepath.Join(tdir, "a.tpl"), []byte(bad), 0o600))

	root := NewLintCommand()
	root.Writer = io.Discard
	err := root.Run(context.Background(), []string{"lint", dir})
	assert.Assert(t, err != nil, "invalid template must fail even without a manifest")
	assert.Assert(t, strings.Contains(err.Error(), "lint failed"),
		"expected a lint failure, got: %v", err)
}
