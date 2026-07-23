// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: This file contains the stencil lint command, which statically
// validates a Stencil module without resolving dependencies.
//
// Every file path here is a CLI argument the user typed, or derived from one.
// Each path-accepting function runs it through filepath.Clean before use, but
// that's normalization, not a security boundary: there's no privilege
// boundary for a path to cross in a local CLI tool operating on paths the
// invoking user already has filesystem access to.

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"

	"github.com/getoutreach/stencil/internal/lint"
	lintmanifest "github.com/getoutreach/stencil/internal/lint/manifest"
	linttemplates "github.com/getoutreach/stencil/internal/lint/templates"
)

// templatesDir is the conventional subdirectory holding a module's *.tpl files.
const templatesDir = "templates"

// stdinName is the placeholder name/path used for stdin ('-') input across
// both the manifest and templates lint/fix paths, so a finding, log line, or
// Applied entry sourced from stdin always reads the same way.
const stdinName = "<stdin>"

// NewLintCommand returns the `lint` command group: an aggregate `lint [dir]`
// plus `lint module-manifest [path]` and `lint templates [files...]`
// subcommands. Validation is local-only.
func NewLintCommand() *cli.Command {
	return &cli.Command{
		Name:        "lint",
		Usage:       "Validate a Stencil module without resolving dependencies",
		ArgsUsage:   "[dir]",
		Description: "Validate a Stencil module's manifest and templates without resolving dependencies",
		Flags:       []cli.Flag{warningsAsErrorsFlag(), fixFlag()},
		Commands:    []*cli.Command{newLintModuleManifestCommand(), newLintTemplatesCommand()},
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

// fixFlag is declared on the lint group and both the module-manifest and
// templates subcommands, since urfave/cli/v3 does not inherit parent flags.
// It is opt-in.
func fixFlag() cli.Flag {
	return &cli.BoolFlag{
		Name: "fix",
		Usage: "automatically fix safe deprecations in place (a manifest is " +
			"re-encoded at 2-space indent when fixed; re-lints after fixing)",
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
// failure (not a lint finding).
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

// newLintTemplatesCommand builds the `lint templates [files...]` subcommand.
func newLintTemplatesCommand() *cli.Command {
	return &cli.Command{
		Name:        "templates",
		Usage:       "Validate Stencil templates' block correctness without rendering",
		ArgsUsage:   "[files...]",
		Description: "Validate template files (defaults to ./templates/**/*.tpl). Use '-' to read a single template from stdin.",
		Flags:       []cli.Flag{warningsAsErrorsFlag(), fixFlag()},
		Action:      runLintTemplates,
	}
}

// runLintTemplates is the `stencil lint templates [files...]` action.
func runLintTemplates(_ context.Context, c *cli.Command) error {
	log := newLintLogger(c)
	fix := c.Bool("fix")

	// stdin mode: a single '-' reads one template from stdin.
	if c.Args().Len() == 1 && c.Args().First() == "-" {
		if readerIsTTY(c.Reader) {
			return errors.New("'-' expects piped input, not an interactive terminal")
		}
		if fix {
			raw, err := io.ReadAll(c.Reader)
			if err != nil {
				return errors.Wrap(err, "failed to read stdin")
			}
			// Fixed template goes to stdout; diagnostics go to the logger
			// (stderr), mirroring `lint module-manifest - --fix`.
			fixed, applied := linttemplates.FixBytes(stdinName, raw)
			if _, werr := c.Writer.Write(fixed); werr != nil {
				return errors.Wrap(werr, "failed to write fixed template")
			}
			logAppliedTemplates(log, applied)
			findings, err := linttemplates.LintReader(stdinName, bytes.NewReader(fixed))
			if err != nil {
				return errors.Wrap(err, "lint failed")
			}
			logFindings(log, findings)
			return failTemplates(log, findings, c.Bool("warnings-as-errors"))
		}
		log.WithField("path", stdinName).Debug("linting template")
		findings, err := linttemplates.LintReader(stdinName, c.Reader)
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		logFindings(log, findings)
		return failTemplates(log, findings, c.Bool("warnings-as-errors"))
	}

	// Resolve the set of template files to lint.
	var files []string
	if c.Args().Len() == 0 {
		fs, err := collectTemplateFiles(log, templatesDir)
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		files = fs
	} else {
		for _, arg := range c.Args().Slice() {
			info, statErr := os.Stat(arg)
			switch {
			case statErr == nil && info.IsDir():
				fs, err := collectTemplateFiles(log, arg)
				if err != nil {
					return errors.Wrap(err, "lint failed")
				}
				files = append(files, fs...)
			default:
				// Explicit non-directory args are linted as-is (a missing/non-.tpl
				// path yields a not-found/parse finding), unlike directory walks
				// which include only .tpl files. This asymmetry is intentional: an
				// explicitly named file is an explicit request to lint it.
				files = append(files, arg)
			}
		}
	}

	// collectTemplateFiles already returns sorted results, but explicit file
	// args can be interleaved with dir-collected files above; sort the merged
	// slice for deterministic output.
	sort.Strings(files)

	var all []lint.Finding
	var err error
	if fix {
		all, err = fixTemplateFiles(log, files)
	} else {
		all, err = lintTemplateFiles(log, files)
	}
	if err != nil {
		return errors.Wrap(err, "lint failed")
	}
	logFindings(log, all)
	return failTemplates(log, all, c.Bool("warnings-as-errors"))
}

// lintTemplateFiles lints each path in files and returns the concatenated
// findings. A non-nil error is an I/O failure (not a lint finding).
func lintTemplateFiles(log logrus.FieldLogger, files []string) ([]lint.Finding, error) {
	var all []lint.Finding
	for _, path := range files {
		findings, err := runTemplateFile(log, path)
		if err != nil {
			return nil, err
		}
		all = append(all, findings...)
	}
	return all, nil
}

// fixTemplateFiles fixes each path in files (see fixTemplateFile) and returns
// the concatenated findings. A non-nil error is an I/O failure.
func fixTemplateFiles(log logrus.FieldLogger, files []string) ([]lint.Finding, error) {
	var all []lint.Finding
	for _, path := range files {
		findings, err := fixTemplateFile(log, path)
		if err != nil {
			return nil, err
		}
		all = append(all, findings...)
	}
	return all, nil
}

// fixTemplateFile migrates legacy block syntax in the template at path to v2
// syntax, writes it back in place only when bytes changed, and returns the
// findings from re-linting the fixed content. A missing/unreadable file is
// reported as a finding (not an error), mirroring runTemplateFile.
func fixTemplateFile(log logrus.FieldLogger, path string) ([]lint.Finding, error) {
	path = filepath.Clean(path)
	log.WithField("path", path).Debug("fixing template")
	raw, err := os.ReadFile(path)
	if err != nil {
		return []lint.Finding{templateOpenErrorFinding(path, err)}, nil
	}

	fixed, applied := linttemplates.FixBytes(path, raw)
	if writeErr := writeFixedFile(path, raw)(fixed); writeErr != nil {
		return nil, errors.Wrapf(writeErr, "failed to write fixed template %q", path)
	}
	logAppliedTemplates(log, applied)

	return linttemplates.LintReader(path, bytes.NewReader(fixed))
}

// templateOpenErrorFinding builds the "not found"/"failed to open" finding
// for a template path that couldn't be read, shared by runTemplateFile and
// fixTemplateFile so their wording for the same class of failure can never
// diverge.
func templateOpenErrorFinding(path string, err error) lint.Finding {
	if os.IsNotExist(err) {
		return lint.Finding{
			Severity: lint.SeverityError,
			Path:     path,
			Message:  "template file not found: " + path,
		}
	}
	return lint.Finding{
		Severity: lint.SeverityError,
		Path:     path,
		Message:  fmt.Sprintf("failed to open template %s: %v", path, err),
	}
}

// logAppliedTemplates logs one info line per template-syntax fix, including
// the source line (linttemplates.Applied carries one; manifest.Applied does
// not).
func logAppliedTemplates(log logrus.FieldLogger, applied []linttemplates.Applied) {
	for _, a := range applied {
		logFixed(log, a.Path, a.Message, logrus.Fields{"line": a.Line})
	}
}

// templateRunner returns a runner that lints every *.tpl under dir. A missing
// or empty templates directory yields zero findings (nothing to lint).
func templateRunner(dir string) runner {
	return func(log logrus.FieldLogger) ([]lint.Finding, error) {
		files, err := collectTemplateFiles(log, dir)
		if err != nil {
			return nil, err
		}
		return lintTemplateFiles(log, files)
	}
}

// collectTemplateFiles returns every *.tpl file under dir (recursive), sorted
// for deterministic output. A missing dir yields an empty slice (no error).
func collectTemplateFiles(log logrus.FieldLogger, dir string) ([]string, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		log.WithField("dir", dir).Debug("no templates directory; nothing to lint")
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat %q", dir)
	}
	if !info.IsDir() {
		// A non-directory here (e.g. a .tpl path, or a stat race where the entry
		// changed type) is nothing to walk; the directory walk includes only .tpl
		// files, so a non-directory input yields no findings. This guards the
		// defensive/race case, not an invariant that callers can never reach it.
		log.WithField("dir", dir).Debug("template path is not a directory; nothing to lint")
		return nil, nil
	}
	var files []string
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if !d.IsDir() && strings.HasSuffix(path, ".tpl") {
			files = append(files, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, errors.Wrapf(walkErr, "failed to walk %q", dir)
	}
	sort.Strings(files)
	log.WithField("dir", dir).Debugf("discovered %d template(s)", len(files))
	return files, nil
}

// runTemplateFile lints a single template path. A missing file is reported as
// an error finding (not an error), mirroring resolveManifestReader.
func runTemplateFile(log logrus.FieldLogger, path string) ([]lint.Finding, error) {
	path = filepath.Clean(path)
	log.WithField("path", path).Debug("linting template")
	fh, err := os.Open(path)
	if err != nil {
		return []lint.Finding{templateOpenErrorFinding(path, err)}, nil
	}
	defer fh.Close()
	return linttemplates.LintReader(path, fh)
}

// failTemplates applies the warnings-as-errors policy and logs the templates
// success line. Thin wrapper over failLint.
func failTemplates(log logrus.FieldLogger, findings []lint.Finding, warningsAsErrors bool) error {
	return failLint(log, "templates are", findings, warningsAsErrors)
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

	// Checked before the --fix branch too, so a non-directory path gets the
	// same friendly finding either way, instead of --fix falling through to a
	// raw "failed to stat ...manifest.yaml: not a directory" error from
	// resolveManifestPath.
	if info, statErr := os.Stat(dir); statErr == nil && !info.IsDir() {
		finding := lint.Finding{
			Severity: lint.SeverityError,
			Path:     dir,
			Message:  fmt.Sprintf("%q is not a directory; stencil lint expects a module directory", dir),
		}
		logFindings(log, []lint.Finding{finding})
		return failIfFindings([]lint.Finding{finding}, c.Bool("warnings-as-errors"))
	}

	if c.Bool("fix") {
		var all []lint.Finding

		fixPath, finding, err := resolveManifestPath(filepath.Join(dir, "manifest.yaml"))
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		if finding == nil {
			raw, readErr := os.ReadFile(fixPath) // path was cleaned inside resolveManifestPath.
			if readErr != nil {
				return errors.Wrapf(readErr, "failed to read %q", fixPath)
			}
			findings, fixErr := fixManifestBytes(log, fixPath, raw, writeFixedFile(fixPath, raw))
			if fixErr != nil {
				return errors.Wrap(fixErr, "lint failed")
			}
			all = append(all, findings...)
		}
		// A missing manifest is skipped, not fixed: aggregate lint treats it as
		// "nothing to lint" (a module may be templates-only), matching the
		// non-fix aggregate path and manifestRunner.

		files, err := collectTemplateFiles(log, filepath.Join(dir, templatesDir))
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		findings, err := fixTemplateFiles(log, files)
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		all = append(all, findings...)

		logFindings(log, all)
		return failLint(log, fmt.Sprintf("module %q is", dir), all, c.Bool("warnings-as-errors"))
	}

	runners := []runner{
		manifestRunner(filepath.Join(dir, "manifest.yaml")),
		templateRunner(filepath.Join(dir, templatesDir)),
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
			return fixAndRelint(c, log, stdinName, raw, func(fixed []byte) error {
				_, werr := c.Writer.Write(fixed)
				return werr
			})
		}
		findings, err := runManifestReader(log, stdinName, bytes.NewReader(raw))
		if err != nil {
			return errors.Wrap(err, "lint failed")
		}
		logFindings(log, findings)
		return failManifest(log, stdinName, findings, c.Bool("warnings-as-errors"))
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
		raw, readErr := os.ReadFile(fixPath) // path was cleaned inside resolveManifestPath.
		if readErr != nil {
			return errors.Wrapf(readErr, "failed to read %q", fixPath)
		}
		return fixAndRelint(c, log, fixPath, raw, writeFixedFile(fixPath, raw))
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
			// Aggregate lint: a missing manifest is not an error (a module may be
			// templates-only). resolveManifestReader only returns a finding for the
			// not-found case, so skip it here. `stencil lint module-manifest` still
			// reports the missing manifest because it calls resolveManifestReader
			// directly, not this runner.
			return nil, nil
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
	path = filepath.Clean(path)
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
				Message:  "manifest file not found: " + path,
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
	path = filepath.Clean(path)
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
				Message:  "manifest file not found: " + path,
			}, nil
		}
		return nil, nil, nil, errors.Wrapf(err, "failed to stat %q", path)
	}

	fh, err := os.Open(path)
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

// failManifest applies the warnings-as-errors policy and logs the manifest
// success line. name is the manifest identifier. Thin wrapper over failLint.
func failManifest(log logrus.FieldLogger, name string, findings []lint.Finding, warningsAsErrors bool) error {
	return failLint(log, fmt.Sprintf("manifest %q is", name), findings, warningsAsErrors)
}

// logApplied logs one info line per fix the fixer applied.
func logApplied(log logrus.FieldLogger, applied []lintmanifest.Applied) {
	for _, a := range applied {
		logFixed(log, a.Path, a.Message, nil)
	}
}

// logFixed logs one info line for a single applied fix, in the shared
// "fixed: <message>" format every fixer uses. fields attaches any additional
// structured context beyond "path" (e.g. templates' source line); pass nil
// for none.
func logFixed(log logrus.FieldLogger, path, message string, fields logrus.Fields) {
	entry := log.WithField("path", path)
	if len(fields) > 0 {
		entry = entry.WithFields(fields)
	}
	entry.Infof("fixed: %s", message)
}

// writeFixedFile returns a writer that overwrites path with the fixed bytes,
// but only when they differ from original -- the bytes the caller already
// read from path -- so a no-op fix doesn't dirty the file, bump its mtime, or
// re-read path from disk. The existing file mode is preserved.
func writeFixedFile(path string, original []byte) func([]byte) error {
	path = filepath.Clean(path)
	return func(fixed []byte) error {
		if bytes.Equal(original, fixed) {
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
	findings, err := fixManifestBytes(log, name, raw, writeFixed)
	if err != nil {
		return errors.Wrap(err, "lint failed")
	}
	logFindings(log, findings)
	return failManifest(log, name, findings, c.Bool("warnings-as-errors"))
}

// fixManifestBytes applies safe deprecation fixes to raw, writes the fixed
// bytes via writeFixed (in-place for a file, or to stdout for '-'), logs each
// applied fix, and returns the findings from re-linting the fixed content
// (unlogged and policy-undecided, so callers can aggregate them with other
// targets' findings before logging/deciding once). If raw cannot be parsed as
// YAML, fixing is skipped and the original bytes are linted instead, so the
// decode error surfaces as a normal finding.
func fixManifestBytes(log logrus.FieldLogger, name string, raw []byte,
	writeFixed func([]byte) error) ([]lint.Finding, error) {
	fixed, applied, ok := lintmanifest.FixBytes(raw)
	if !ok {
		// Unparseable: fall back to a normal lint of the original bytes.
		return runManifestReader(log, name, bytes.NewReader(raw))
	}

	if err := writeFixed(fixed); err != nil {
		return nil, errors.Wrap(err, "failed to write fixed manifest")
	}
	logApplied(log, applied)

	return runManifestReader(log, name, bytes.NewReader(fixed))
}

// failLint applies the warnings-as-errors policy and, on success, logs a
// "<subject> valid" line ("<subject> valid (N warning(s))" when there are
// warnings). subject is a phrase with no trailing punctuation, e.g.
// `fmt.Sprintf("manifest %q is", name)` or "templates are". Shared by the
// per-target fail* wrappers.
func failLint(log logrus.FieldLogger, subject string, findings []lint.Finding, warningsAsErrors bool) error {
	if err := failIfFindings(findings, warningsAsErrors); err != nil {
		return err
	}
	if _, warnings := lint.Counts(findings); warnings > 0 {
		log.Infof("%s valid (%d warning(s))", subject, warnings)
	} else {
		log.Infof("%s valid", subject)
	}
	return nil
}

// failIfFindings returns a summary error when the findings fail the policy:
// with warningsAsErrors, any finding fails; otherwise only error findings fail.
func failIfFindings(findings []lint.Finding, warningsAsErrors bool) error {
	errs, warns := lint.Counts(findings)
	fail := errs > 0 || (warningsAsErrors && warns > 0)
	if !fail {
		return nil
	}
	return fmt.Errorf("lint failed: %d error(s), %d warning(s)", errs, warns)
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
