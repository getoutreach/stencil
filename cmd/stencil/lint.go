// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: This file contains the stencil lint command, which statically
// validates a Stencil module without resolving dependencies.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/getoutreach/stencil/internal/lint"
	lintmanifest "github.com/getoutreach/stencil/internal/lint/manifest"
)

// NewLintCommand returns the `lint` command group: an aggregate `lint [dir]`
// plus a `lint module-manifest [path]` subcommand. Validation is local-only.
func NewLintCommand() *cli.Command {
	return &cli.Command{
		Name:        "lint",
		Usage:       "Validate a Stencil module without resolving dependencies",
		ArgsUsage:   "[dir]",
		Description: "Validate a Stencil module's manifest without resolving dependencies (template linting follows in DT-4828)",
		Flags:       []cli.Flag{warningsAsErrorsFlag(), fixFlag()},
		Commands:    []*cli.Command{newLintModuleManifestCommand()},
		Action:      runLintAggregate,
	}
}

// warningsAsErrorsFlag is declared on both the lint group and the manifest
// subcommand, since urfave/cli/v3 does not inherit parent flags.
func warningsAsErrorsFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  "warnings-as-errors",
		Usage: "treat warnings as errors (fail on any finding)",
		Value: true,
	}
}

// fixFlag is declared on both the lint group and the module-manifest
// subcommand, since urfave/cli/v3 does not inherit parent flags. It is opt-in.
func fixFlag() cli.Flag {
	return &cli.BoolFlag{
		Name: "fix",
		Usage: "automatically fix safe deprecations in place, re-encoding the " +
			"manifest at 2-space indent when a fix is applied (re-lints after fixing)",
		Value: false,
	}
}

// newLintLogger builds the command logger, honoring the global --debug flag.
func newLintLogger(c *cli.Command) *logrus.Logger {
	log := logrus.New()
	if c.Bool("debug") {
		log.SetLevel(logrus.DebugLevel)
	}
	return log
}

// runner lints one input and returns its findings. A non-nil error is an I/O
// failure (not a lint finding). DT-4828 adds a template runner alongside the
// manifest runner.
type runner func(log logrus.FieldLogger) ([]lint.Finding, error)

// newLintModuleManifestCommand builds the `lint module-manifest [path]` subcommand.
func newLintModuleManifestCommand() *cli.Command {
	return &cli.Command{
		Name:        "module-manifest",
		Usage:       "Validate a module's manifest.yaml without resolving dependencies",
		ArgsUsage:   "[path]",
		Description: "Validate a single template repository manifest (manifest.yaml; defaults to ./manifest.yaml). Use '-' to read from stdin.",
		Flags:       []cli.Flag{warningsAsErrorsFlag(), fixFlag()},
		Action:      runLintModuleManifest,
	}
}

// runLintAggregate is the `stencil lint [dir]` action.
func runLintAggregate(_ context.Context, c *cli.Command) error {
	if c.Args().Len() > 1 {
		return errors.New("expected at most one argument, a module directory")
	}
	if c.Args().Len() == 0 && stdinIsPipe() {
		return errors.New("stencil lint expects a module directory, not piped input; " +
			"use 'stencil lint module-manifest -' (or, once available, 'stencil lint templates -') to lint from stdin")
	}

	dir := "."
	if c.Args().Len() == 1 {
		dir = c.Args().First()
	}

	log := newLintLogger(c)

	if c.Bool("fix") {
		// Aggregate --fix covers the manifest only (templates: future DT-4828).
		fixPath, finding, err := resolveManifestPath(filepath.Join(dir, "manifest.yaml"))
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		if finding != nil {
			logFindings(log, []lint.Finding{*finding})
			return failIfFindings([]lint.Finding{*finding}, c.Bool("warnings-as-errors"))
		}
		raw, readErr := os.ReadFile(fixPath) //nolint:gosec // Why: user-provided lint target.
		if readErr != nil {
			return errors.Wrapf(readErr, "failed to read %q", fixPath)
		}
		return fixAndRelint(c, log, fixPath, raw, writeFixedFile(fixPath))
	}

	runners := []runner{
		manifestRunner(filepath.Join(dir, "manifest.yaml")),
		// DT-4828 appends a template runner here.
	}

	var all []lint.Finding
	for _, run := range runners {
		findings, err := run(log)
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		all = append(all, findings...)
	}
	logFindings(log, all)
	return failIfFindings(all, c.Bool("warnings-as-errors"))
}

// runLintModuleManifest is the `stencil lint module-manifest [path]` action.
func runLintModuleManifest(_ context.Context, c *cli.Command) error {
	if c.Args().Len() > 1 {
		return errors.New("expected at most one argument, a path to manifest.yaml")
	}

	log := newLintLogger(c)

	// stdin mode
	if c.Args().First() == "-" {
		if readerIsTTY(c.Reader) {
			return errors.New("'-' expects piped input, not an interactive terminal")
		}
		raw, err := io.ReadAll(c.Reader)
		if err != nil {
			return errors.Wrap(err, "failed to read stdin")
		}
		if c.Bool("fix") {
			// Fixed YAML goes to stdout; diagnostics go to the logger (stderr).
			return fixAndRelint(c, log, "<stdin>", raw, func(fixed []byte) error {
				_, werr := c.Writer.Write(fixed)
				return werr
			})
		}
		findings, err := runManifestReader(log, "<stdin>", bytes.NewReader(raw))
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		logFindings(log, findings)
		return failManifest(log, "<stdin>", findings, c.Bool("warnings-as-errors"))
	}

	path := "./manifest.yaml"
	if c.Args().Len() == 1 {
		path = c.Args().First()
	}

	if c.Bool("fix") {
		// Resolve a directory arg to its manifest.yaml, then fix in place.
		fixPath, finding, err := resolveManifestPath(path)
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		if finding != nil {
			logFindings(log, []lint.Finding{*finding})
			return failManifest(log, fixPath, []lint.Finding{*finding},
				c.Bool("warnings-as-errors"))
		}
		raw, readErr := os.ReadFile(fixPath) //nolint:gosec // Why: user-provided lint target.
		if readErr != nil {
			return errors.Wrapf(readErr, "failed to read %q", fixPath)
		}
		return fixAndRelint(c, log, fixPath, raw, writeFixedFile(fixPath))
	}

	r, closer, finding, err := resolveManifestReader(path)
	if err != nil {
		return errors.Wrap(err, "lint failed")
	}
	if finding != nil {
		logFindings(log, []lint.Finding{*finding})
		return failManifest(log, path, []lint.Finding{*finding}, c.Bool("warnings-as-errors"))
	}
	if closer != nil {
		defer closer.Close()
	}

	findings, err := runManifestReader(log, path, r)
	if err != nil {
		return errors.Wrap(err, "lint failed")
	}
	logFindings(log, findings)
	return failManifest(log, path, findings, c.Bool("warnings-as-errors"))
}

// manifestRunner returns a runner that lints the manifest at path. A missing
// file is reported as a finding (not an error) so the aggregate treats it
// uniformly with other findings.
func manifestRunner(path string) runner {
	return func(log logrus.FieldLogger) ([]lint.Finding, error) {
		r, closer, finding, err := resolveManifestReader(path)
		if err != nil {
			return nil, err
		}
		if finding != nil {
			return []lint.Finding{*finding}, nil
		}
		if closer != nil {
			defer closer.Close()
		}
		return runManifestReader(log, path, r)
	}
}

// resolveManifestPath resolves path to the manifest file to operate on. If path
// is a directory, it appends "manifest.yaml". A missing file yields a "manifest
// file not found" finding (not an error), mirroring resolveManifestReader so the
// --fix and non-fix paths agree on target location and absence reporting.
func resolveManifestPath(path string) (resolved string, finding *lint.Finding, err error) {
	info, statErr := os.Stat(path)
	if statErr == nil && info.IsDir() {
		path = filepath.Join(path, "manifest.yaml")
		_, statErr = os.Stat(path)
	}
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return path, &lint.Finding{
				Severity: lint.SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("manifest file not found: %s", path),
			}, nil
		}
		return path, nil, errors.Wrapf(statErr, "failed to stat %q", path)
	}
	return path, nil, nil
}

// resolveManifestReader resolves path to an io.Reader. If path is a directory,
// it appends "manifest.yaml". A missing file yields a "manifest file not found"
// finding (not an error). The returned io.Closer (when non-nil) must be closed
// by the caller. A non-nil error is an unexpected I/O failure.
func resolveManifestReader(path string) (io.Reader, io.Closer, *lint.Finding, error) {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		// A directory: lint its manifest.yaml (mirrors `lint [dir]`).
		path = filepath.Join(path, "manifest.yaml")
		_, err = os.Stat(path)
	}
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, &lint.Finding{
				Severity: lint.SeverityError,
				Path:     path,
				Message:  fmt.Sprintf("manifest file not found: %s", path),
			}, nil
		}
		return nil, nil, nil, errors.Wrapf(err, "failed to stat %q", path)
	}

	fh, err := os.Open(path) //nolint:gosec // Why: path is a user-provided lint target.
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to open %q", path)
	}
	return fh, fh, nil, nil
}

// runManifestReader loads + validates one manifest from r and returns its
// findings. It does NOT log the findings themselves; logging is the command's
// responsibility so findings are emitted exactly once. name is used in messages.
func runManifestReader(log logrus.FieldLogger, name string, r io.Reader) ([]lint.Finding, error) {
	res, err := lintmanifest.Load(r)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read manifest %q", name)
	}
	findings := lintmanifest.Validate(res)
	if res.MultiDoc {
		log.WithField("path", name).Warn("additional YAML documents after the first are ignored")
	}
	return findings, nil
}

// logFindings logs each finding via logrus (stderr). Errors log at error level,
// warnings at warn level, and info findings at info level. When a finding has a
// resolved source line (Line > 0), it is attached as a "line" field.
func logFindings(log logrus.FieldLogger, findings []lint.Finding) {
	for _, f := range findings {
		entry := log.WithField("path", f.Path)
		if f.Line > 0 {
			entry = entry.WithField("line", f.Line)
		}
		switch f.Severity {
		case lint.SeverityWarning:
			entry.Warn(f.Message)
		case lint.SeverityError:
			entry.Error(f.Message)
		case lint.SeverityInfo:
			entry.Info(f.Message)
		default:
			// Defensive: surface any unexpected severity as an error so it is
			// never silently dropped.
			entry.Error(f.Message)
		}
	}
}

// failManifest applies the warnings-as-errors policy and logs the success line
// when the manifest passes. name is the manifest identifier for the success log.
func failManifest(log logrus.FieldLogger, name string, findings []lint.Finding, warningsAsErrors bool) error {
	if err := failIfFindings(findings, warningsAsErrors); err != nil {
		return err
	}
	if _, warnings := lint.Counts(findings); warnings > 0 {
		log.Infof("manifest %q is valid (%d warning(s))", name, warnings)
	} else {
		log.Infof("manifest %q is valid", name)
	}
	return nil
}

// logApplied logs one info line per fix the fixer applied.
func logApplied(log logrus.FieldLogger, applied []lintmanifest.Applied) {
	for _, a := range applied {
		log.WithField("path", a.Path).Infof("fixed: %s", a.Message)
	}
}

// writeFixedFile returns a writer that overwrites path with the fixed bytes,
// but only when they differ from the current contents (so a no-op fix does not
// dirty the file or bump its mtime). The existing file mode is preserved.
func writeFixedFile(path string) func([]byte) error {
	return func(fixed []byte) error {
		current, err := os.ReadFile(path) //nolint:gosec // Why: user-provided lint target.
		if err == nil && bytes.Equal(current, fixed) {
			return nil // no change: leave the file untouched
		}
		mode := os.FileMode(0o644)
		if info, statErr := os.Stat(path); statErr == nil {
			mode = info.Mode().Perm()
		}
		return os.WriteFile(path, fixed, mode)
	}
}

// fixAndRelint applies safe deprecation fixes to raw, writes the fixed bytes via
// writeFixed (in-place for a file, or to stdout for '-'), logs each applied fix,
// then re-lints the fixed content and applies the warnings-as-errors policy to
// whatever remains. If raw cannot be parsed as YAML, fixing is skipped and the
// normal lint runs so the decode error is reported.
func fixAndRelint(c *cli.Command, log logrus.FieldLogger, name string, raw []byte,
	writeFixed func([]byte) error) error {
	fixed, applied, ok := lintmanifest.FixBytes(raw)
	if !ok {
		// Unparseable: fall back to a normal lint of the original bytes.
		findings, err := runManifestReader(log, name, bytes.NewReader(raw))
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		logFindings(log, findings)
		return failManifest(log, name, findings, c.Bool("warnings-as-errors"))
	}

	if err := writeFixed(fixed); err != nil {
		return errors.Wrap(err, "failed to write fixed manifest")
	}
	logApplied(log, applied)

	findings, err := runManifestReader(log, name, bytes.NewReader(fixed))
	if err != nil {
		return errors.Wrap(err, "lint failed")
	}
	logFindings(log, findings)
	return failManifest(log, name, findings, c.Bool("warnings-as-errors"))
}

// failIfFindings returns a summary error when the findings fail the policy:
// with warningsAsErrors, any finding fails; otherwise only error findings fail.
func failIfFindings(findings []lint.Finding, warningsAsErrors bool) error {
	errs, warns := lint.Counts(findings)
	fail := errs > 0 || (warningsAsErrors && warns > 0)
	if !fail {
		return nil
	}
	return fmt.Errorf("manifest validation failed: %d error(s), %d warning(s)", errs, warns)
}

// stdinIsPipe reports whether stdin is a pipe or a regular-file redirect (i.e.
// data is being piped in), as opposed to a terminal or an empty CI device.
func stdinIsPipe() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeNamedPipe != 0 {
		return true
	}
	return info.Mode().IsRegular() && info.Size() > 0
}

// readerIsTTY reports whether r is an interactive terminal. Only an *os.File
// backed by a character device (e.g. the real os.Stdin at a TTY) qualifies; any
// other reader (a pipe, a file redirect, or a reader injected by a test or
// programmatic caller) is treated as non-interactive. This keeps the real CLI
// behavior identical (c.Reader defaults to os.Stdin) while letting in-process
// callers feed piped input through c.Reader.
func readerIsTTY(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
