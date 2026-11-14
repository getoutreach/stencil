// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the snapshot assertions, which compare
// rendered files with files in testdata/snapshots.

package rendertest

import (
	"bytes"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

// updateSnapshots is whether Snapshot writes snapshots instead of comparing
// them. Only an init function sets it, so reads need no lock.
var updateSnapshots = false //nolint:gochecknoglobals // Why: A build tag sets it in init.

// updateSnapshotsCommand is the command that updates snapshots.
const updateSnapshotsCommand = "mise run rendertest:update-snapshots"

// snapshotDir is the directory, relative to the test package, that holds
// the snapshots.
const snapshotDir = "testdata/snapshots"

// unsafePathChars are the characters of snapshot paths that some file
// systems do not allow in file names.
const unsafePathChars = `:*?"<>|\`

// safePath returns p with each character in unsafePathChars replaced with
// "_".
func safePath(p string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(unsafePathChars, r) {
			return '_'
		}
		return r
	}, p)
}

// Snapshot compares every rendered file with its snapshot, at
// testdata/snapshots/<test name>/<file path>. It reports each missing or
// different snapshot with Error, and then calls FailNow if there was one. A
// test that is built with the rendertest_update_snapshots build tag writes
// the snapshots instead.
func (r *Result) Snapshot(tb testing.TB) {
	tb.Helper()

	if r.err != nil {
		tb.Fatalf("cannot compare snapshots, the render failed: %v", r.err)
	}

	ok := true
	for _, f := range r.files {
		ok = f.snapshot(tb, tb.Errorf) && ok
	}
	if !ok {
		tb.FailNow()
	}
}

// Snapshot compares the file with its snapshot, at
// testdata/snapshots/<test name>/<file path>. It fails the test with Fatal if
// the snapshot is missing or different. A test that is built with the
// rendertest_update_snapshots build tag writes the snapshot instead.
func (fr *FileResult) Snapshot(tb testing.TB) {
	tb.Helper()
	fr.snapshot(tb, tb.Fatalf)
}

// snapshot compares the file with its snapshot, or updates the snapshot. It
// reports a problem with fail and returns false.
func (fr *FileResult) snapshot(tb testing.TB, fail func(format string, args ...any)) bool {
	tb.Helper()

	p := filepath.Join(snapshotDir, filepath.FromSlash(safePath(path.Join(tb.Name(), fr.path))))

	if updateSnapshots {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			fail("failed to create the snapshot directory for %s: %v", fr.path, err)
			return false
		}
		if err := os.WriteFile(p, fr.contents, 0o600); err != nil {
			fail("failed to write the snapshot of %s: %v", fr.path, err)
			return false
		}
		tb.Logf("snapshot updated: %s", filepath.ToSlash(p))
		return true
	}

	want, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		fail("no snapshot found for %s at %s\n\nTo create snapshots, run: %s",
			fr.path, filepath.ToSlash(p), updateSnapshotsCommand)
		return false
	} else if err != nil {
		fail("failed to read the snapshot of %s: %v", fr.path, err)
		return false
	}

	if !bytes.Equal(want, fr.contents) {
		fail("snapshot mismatch for %s\n\n%s\nTo update snapshots, run: %s",
			fr.path, unifiedDiff(filepath.ToSlash(p), "rendered output", string(want), fr.String()), updateSnapshotsCommand)
		return false
	}
	return true
}
