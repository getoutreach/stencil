// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Guards against cupaloy snapshot filenames containing
// characters that are illegal in zip files (which breaks `go get`/`go mod
// download`, since module source is packaged as a zip) and on Windows/NTFS
// filesystems.

package snapshotlint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// illegalFilenameChars are characters reserved on Windows/NTFS and unsafe in
// zip archives. cupaloy writes a snapshot's filename verbatim from the
// enclosing test's t.Name(), so any of these appearing in a subtest name
// produces a snapshot file that can't be packaged into a module zip or
// extracted on Windows.
const illegalFilenameChars = `<>:"|?*`

// TestSnapshotFilenamesAreZipSafe walks every directory named ".snapshots"
// in the repository and fails if any file within it has a name containing a
// character illegal in zip files or on Windows/NTFS. This checks the
// committed snapshot files directly, rather than parsing Go test sources for
// how the underlying subtest name was constructed (literal, table-driven
// struct field, fmt.Sprintf, etc.), since the filename on disk is the actual
// artifact that breaks `go get`.
func TestSnapshotFilenamesAreZipSafe(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var findings []string
	walkErr := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		name := d.Name()
		if path != repoRoot && strings.HasPrefix(name, ".") && name != ".snapshots" {
			return filepath.SkipDir
		}
		if name != ".snapshots" {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", path, err)
		}
		for _, entry := range entries {
			if bad := illegalChars(entry.Name()); bad != "" {
				findings = append(findings, fmt.Sprintf("%s: filename contains illegal character(s) %q",
					filepath.Join(path, entry.Name()), bad))
			}
		}
		return filepath.SkipDir
	})
	assert.NilError(t, walkErr)
	assert.Assert(t, len(findings) == 0,
		"found snapshot filenames illegal in zip files (breaking `go get`) and on Windows/NTFS:\n%s",
		strings.Join(findings, "\n"))
}

// findRepoRoot returns the repository root, derived from this file's own
// location (internal/snapshotlint/colon_test.go is two directories below
// the root).
func findRepoRoot(t *testing.T) string {
	_, thisFile, _, ok := runtime.Caller(0)
	assert.Assert(t, ok, "failed to determine the location of this test file")
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// illegalChars returns the distinct illegal characters found in name, or an
// empty string if name contains none.
func illegalChars(name string) string {
	var found []rune
	for _, r := range name {
		if r < 0x20 || strings.ContainsRune(illegalFilenameChars, r) {
			found = append(found, r)
		}
	}
	return string(found)
}
