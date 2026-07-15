// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Defines the shared finding model used by the stencil lint
// subcommands.

// Package lint defines the shared finding model used by the stencil lint
// subcommands.
package lint

import "fmt"

// Severity is the severity of a lint Finding.
type Severity string

// Severity values.
const (
	// SeverityError marks a finding that is always a failure.
	SeverityError Severity = "error"
	// SeverityWarning marks a finding that fails only when warnings are
	// treated as errors.
	SeverityWarning Severity = "warning"
	// SeverityInfo marks an informational finding that never fails a lint run.
	SeverityInfo Severity = "info"
)

// Finding is a single problem discovered while linting.
type Finding struct {
	// Severity is error or warning.
	Severity Severity
	// Path locates the problem. The manifest linter stores a dotted document
	// location (e.g. "arguments.foo.schema"); the templates linter stores the
	// template file path. In both cases Line, when non-zero, is the 1-based
	// source line.
	Path string
	// Message is a human-readable description of the problem.
	Message string
	// Line is the 1-based source line of the YAML key this finding references,
	// or 0 when no line is known (whole-document findings, or an unresolved path).
	Line int
}

// Findings accumulates Finding values during a lint run. The zero value is
// ready to use.
type Findings struct {
	items []Finding
}

// Errorf appends a SeverityError finding at path with a formatted message.
func (f *Findings) Errorf(path, format string, a ...any) {
	f.items = append(f.items, Finding{
		Severity: SeverityError,
		Path:     path,
		Message:  fmt.Sprintf(format, a...),
	})
}

// Warnf appends a SeverityWarning finding at path with a formatted message.
func (f *Findings) Warnf(path, format string, a ...any) {
	f.items = append(f.items, Finding{
		Severity: SeverityWarning,
		Path:     path,
		Message:  fmt.Sprintf(format, a...),
	})
}

// Infof appends a SeverityInfo finding at path with a formatted message.
func (f *Findings) Infof(path, format string, a ...any) {
	f.items = append(f.items, Finding{
		Severity: SeverityInfo,
		Path:     path,
		Message:  fmt.Sprintf(format, a...),
	})
}

// Add appends a pre-built Finding.
func (f *Findings) Add(find Finding) {
	f.items = append(f.items, find)
}

// Items returns the accumulated findings in insertion order.
func (f *Findings) Items() []Finding {
	return f.items
}

// Counts returns the number of error and warning findings.
func (f *Findings) Counts() (errors, warnings int) {
	return Counts(f.items)
}

// Counts tallies the error and warning findings in a slice.
func Counts(findings []Finding) (errors, warnings int) {
	for i := range findings {
		switch findings[i].Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		case SeverityInfo:
			// Info findings are intentionally not counted: they never fail a
			// lint run, so they must not affect the error/warning tally.
		}
	}
	return errors, warnings
}
