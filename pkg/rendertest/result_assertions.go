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

// HasNoFile checks that there is no rendered file at path. Deleted and
// skipped files are not rendered files.
func (r *Result) HasNoFile(tb testing.TB, path string) {
	tb.Helper()

	if r.File(path) != nil {
		tb.Errorf("expected no file %q", path)
	}
}

// AssertDeleted checks that the template deleted the file at path.
func (r *Result) AssertDeleted(tb testing.TB, path string) {
	tb.Helper()

	if !slices.Contains(r.deletedPaths, path) {
		tb.Errorf("expected file %q to be deleted, got deleted files %q", path, r.deletedPaths)
	}
}

// AssertSkipped checks that the template skipped the file at path.
func (r *Result) AssertSkipped(tb testing.TB, path string) {
	tb.Helper()

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
