// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the assertions on a Result.

package rendertest

import (
	"slices"
	"testing"
)

// paths returns the paths of the rendered files.
func (r *Result) paths() []string {
	paths := make([]string, 0, len(r.files))
	for _, f := range r.files {
		paths = append(paths, f.path)
	}
	return paths
}

// HasFile returns the rendered file at path. It fails the test with Fatal if
// the render failed or there is no such file.
func (r *Result) HasFile(tb testing.TB, path string) *FileResult {
	tb.Helper()

	if r.err != nil {
		tb.Fatalf("expected file %q, but the render failed: %v", path, r.err)
	}
	f := r.File(path)
	if f == nil {
		tb.Fatalf("expected file %q, got files %q", path, r.paths())
	}
	return f
}

// reportRenderError fails the assertion that expected what, a printf format
// with args, if the render failed, and returns whether it did. A failed render
// has no files, so the assertions on files could not tell it from a template
// that does not write the file.
func (r *Result) reportRenderError(tb testing.TB, what string, args ...any) bool {
	tb.Helper()

	if r.err == nil {
		return false
	}
	tb.Errorf("expected "+what+", but the render failed: %v", append(args, r.err)...)
	return true
}

// HasNoFile checks that there is no rendered file at path. Deleted and
// skipped files are not rendered files. It fails if the render failed.
func (r *Result) HasNoFile(tb testing.TB, path string) {
	tb.Helper()

	if r.reportRenderError(tb, "no file %q", path) {
		return
	}
	if r.File(path) != nil {
		tb.Errorf("expected no file %q", path)
	}
}

// AssertDeleted checks that the template deleted the file at path. It fails if
// the render failed.
func (r *Result) AssertDeleted(tb testing.TB, path string) {
	tb.Helper()

	if r.reportRenderError(tb, "file %q to be deleted", path) {
		return
	}
	if !slices.Contains(r.deletedPaths, path) {
		tb.Errorf("expected file %q to be deleted, got deleted files %q", path, r.deletedPaths)
	}
}

// AssertSkipped checks that the template skipped the file at path. It fails if
// the render failed.
func (r *Result) AssertSkipped(tb testing.TB, path string) {
	tb.Helper()

	if r.reportRenderError(tb, "file %q to be skipped", path) {
		return
	}
	if !slices.Contains(r.skippedPaths, path) {
		tb.Errorf("expected file %q to be skipped, got skipped files %q", path, r.skippedPaths)
	}
}

// HasError returns the render error. It fails the test with Fatal if the
// render succeeded.
func (r *Result) HasError(tb testing.TB) *ErrorResult {
	tb.Helper()

	if r.err == nil {
		tb.Fatal("expected the render to fail, but it succeeded")
	}
	return &ErrorResult{err: r.err}
}

// HasNoError checks that the render succeeded.
func (r *Result) HasNoError(tb testing.TB) {
	tb.Helper()

	if r.err != nil {
		tb.Errorf("expected the render to succeed, got error: %v", r.err)
	}
}
