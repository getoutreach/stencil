// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Guards against cupaloy snapshot filenames containing
// characters or names that are illegal in zip files (which breaks `go
// get`/`go mod download`, since module source is packaged as a zip) and on
// Windows/NTFS filesystems.

package snapshotlint_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode"

	"gotest.tools/v3/assert"
)

// allowedFileNamePunctuation are the ASCII punctuation characters, plus the
// space character, that golang.org/x/mod/module's (unexported) fileNameOK
// allows in a module zip file path element, on top of ASCII letters and
// digits. Every other ASCII character - including `" ' * / : ; < > ? \` |`
// and all ASCII control characters - is rejected. This must track
// fileNameOK in golang.org/x/mod/module/module.go exactly; it is not a
// "known bad characters" blocklist.
const allowedFileNamePunctuation = "!#$%&()+,-.=@[]^_{}~ "

// windowsReservedNames are the path element base names (the portion before
// the first dot, matched case-insensitively) that Windows reserves as
// device names, per golang.org/x/mod/module's badWindowsNames.
var windowsReservedNames = map[string]bool{
	"CON": true, "PRN": true, "AUX": true, "NUL": true,
	"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
	"COM6": true, "COM7": true, "COM8": true, "COM9": true,
	"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
	"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
}

// TestSnapshotFilenamesAreZipSafe walks every directory named ".snapshots"
// in the repository and fails if any file within it has a name that would
// be rejected by golang.org/x/mod/module.CheckFilePath. Go runs that check
// when `go get`/`go mod download` packages the module as a zip, so a
// violation here breaks installing the module - as happened with colons
// (#514), and, undetected by that fix, semicolons. cupaloy writes a
// snapshot's filename verbatim from the enclosing test's t.Name(), so any
// illegal character or reserved name in a subtest name produces a broken
// snapshot file.
//
// This checks the committed snapshot files directly, rather than parsing Go
// test sources for how the underlying subtest name was constructed
// (literal, table-driven struct field, fmt.Sprintf, etc.), since the
// filename on disk is the actual artifact that breaks `go get`.
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
			if reason := fileNameViolation(entry.Name()); reason != "" {
				findings = append(findings, fmt.Sprintf("%s: %s", filepath.Join(path, entry.Name()), reason))
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
// location (internal/snapshotlint/zip_safe_test.go is two directories below
// the root).
func findRepoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	assert.Assert(t, ok, "failed to determine the location of this test file")
	return filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
}

// fileNameViolation reports why name would be rejected as a module zip
// path element (mirroring golang.org/x/mod/module.CheckFilePath's
// checkElem), or an empty string if name is valid. name must be a single
// path element (no "/"); snapshot filenames never contain one.
func fileNameViolation(name string) string {
	if name == "" {
		return "empty file name"
	}
	if strings.Count(name, ".") == len(name) {
		return fmt.Sprintf("file name %q consists entirely of dots", name)
	}
	if strings.HasSuffix(name, ".") {
		return fmt.Sprintf("file name %q ends in a dot", name)
	}

	var bad []rune
	for _, r := range name {
		if !fileNameCharOK(r) {
			bad = append(bad, r)
		}
	}
	if len(bad) > 0 {
		return fmt.Sprintf("file name contains character(s) illegal in a module zip path element: %q", string(bad))
	}

	base := name
	if i := strings.IndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	if windowsReservedNames[strings.ToUpper(base)] {
		return fmt.Sprintf("file name %q is a Windows-reserved device name", base)
	}

	return ""
}

// fileNameCharOK reports whether r is allowed in a module zip file path
// element: an ASCII letter or digit, one of allowedFileNamePunctuation, or
// any Unicode letter outside the ASCII range. This mirrors
// golang.org/x/mod/module's unexported fileNameOK.
func fileNameCharOK(r rune) bool {
	if r < 0x80 {
		if '0' <= r && r <= '9' || 'A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' {
			return true
		}
		return strings.ContainsRune(allowedFileNamePunctuation, r)
	}
	return unicode.IsLetter(r)
}
