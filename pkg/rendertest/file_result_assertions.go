// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the assertions on a FileResult.

package rendertest

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Contains checks that the contents of the file contain substr.
func (fr *FileResult) Contains(tb testing.TB, substr string) {
	tb.Helper()

	if !bytes.Contains(fr.contents, []byte(substr)) {
		tb.Errorf("expected file %q to contain %q, got:\n%s", fr.path, substr, fr.contents)
	}
}

// NotContains checks that the contents of the file do not contain substr.
func (fr *FileResult) NotContains(tb testing.TB, substr string) {
	tb.Helper()

	if bytes.Contains(fr.contents, []byte(substr)) {
		tb.Errorf("expected file %q not to contain %q, got:\n%s", fr.path, substr, fr.contents)
	}
}

// ContentsEqual checks that the contents of the file are expected.
func (fr *FileResult) ContentsEqual(tb testing.TB, expected string) {
	tb.Helper()

	if diff := cmp.Diff(expected, fr.String()); diff != "" {
		tb.Errorf("unexpected contents of file %q (-want +got):\n%s", fr.path, diff)
	}
}

// ModeEqual checks that the mode of the file is expected.
func (fr *FileResult) ModeEqual(tb testing.TB, expected os.FileMode) {
	tb.Helper()

	if fr.mode != expected {
		tb.Errorf("expected file %q to have mode %v, got %v", fr.path, expected, fr.mode)
	}
}

// HasBlock returns the contents of the named block in the file. It fails the
// test with Fatal if the file has no such block.
func (fr *FileResult) HasBlock(tb testing.TB, name string) string {
	tb.Helper()

	v, ok := fr.blocks[name]
	if !ok {
		tb.Fatalf("expected file %q to have block %q", fr.path, name)
	}
	return v
}

// WriteTo writes the contents of the file to a temporary file, for tools
// that read files, and returns its path. The file has the same name and
// mode, and is removed when the test ends.
func (fr *FileResult) WriteTo(tb testing.TB) string {
	tb.Helper()

	p := filepath.Join(tb.TempDir(), filepath.Base(fr.path))
	if err := os.WriteFile(p, fr.contents, fr.mode.Perm()); err != nil {
		tb.Fatalf("failed to write file %q: %v", fr.path, err)
	}
	return p
}
