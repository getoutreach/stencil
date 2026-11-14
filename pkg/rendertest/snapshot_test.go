// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for comparing snapshots. They are
// not built with the rendertest_update_snapshots build tag.

//go:build !rendertest_update_snapshots

package rendertest_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSnapshotsMatch(t *testing.T) {
	result := renderTwo(t)
	writeSnapshot(t, t.Name(), "one.txt", "one\n")
	writeSnapshot(t, t.Name(), "dir/two.txt", "two\n")

	assertPasses(t, func(tb *recordingTB) { result.Snapshot(tb) })
	assertPasses(t, func(tb *recordingTB) { result.File("dir/two.txt").Snapshot(tb) })
}

func TestSnapshotsMissing(t *testing.T) {
	result := renderTwo(t)

	r := record(t, func(tb *recordingTB) { result.Snapshot(tb) })
	assert.Assert(t, r.fatal, "expected FailNow after the errors")
	assert.Equal(t, len(r.errors), 2, "expected one error for each file")
	for _, e := range r.errors {
		assert.Assert(t, strings.HasPrefix(e, "no snapshot found for "), e)
		assert.Assert(t, strings.Contains(e, "mise run rendertest:update-snapshots"), e)
	}

	_, err := os.Stat("testdata")
	assert.ErrorIs(t, err, os.ErrNotExist, "expected no snapshots to be written")

	r = record(t, func(tb *recordingTB) { result.File("one.txt").Snapshot(tb) })
	assert.Assert(t, r.fatal, "expected Fatal")
	assert.Equal(t, len(r.errors), 1)
}

func TestSnapshotsMismatch(t *testing.T) {
	result := renderTwo(t)
	writeSnapshot(t, t.Name(), "one.txt", "one\n")
	writeSnapshot(t, t.Name(), "dir/two.txt", "old\n")

	r := record(t, func(tb *recordingTB) { result.Snapshot(tb) })
	assert.Assert(t, r.fatal, "expected FailNow after the errors")
	assert.DeepEqual(t, r.errors, []string{
		"snapshot mismatch for dir/two.txt\n\n" +
			"--- testdata/snapshots/TestSnapshotsMismatch/dir/two.txt\n" +
			"+++ rendered output\n" +
			"@@ -1 +1 @@\n" +
			"-old\n" +
			"+two\n" +
			"\nTo update snapshots, run: mise run rendertest:update-snapshots",
	})

	b, err := os.ReadFile(filepath.Join("testdata", "snapshots", t.Name(), "dir", "two.txt"))
	assert.NilError(t, err)
	assert.Equal(t, string(b), "old\n", "expected the snapshot not to be updated")
}

func TestSnapshotsOfSubtests(t *testing.T) {
	result := renderTwo(t)

	t.Run("name: with *unsafe* characters?", func(t *testing.T) {
		writeSnapshot(t, "TestSnapshotsOfSubtests/name__with__unsafe__characters_", "one.txt", "one\n")
		assertPasses(t, func(tb *recordingTB) { result.File("one.txt").Snapshot(tb) })
	})
}

func TestSnapshotOfAFailedRender(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("args.tpl").Run(t)
	assertFatal(t, func(tb *recordingTB) { result.Snapshot(tb) })
}
