// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for updating snapshots. They are
// built only with the rendertest_update_snapshots build tag.

//go:build rendertest_update_snapshots

package rendertest_test

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestSnapshotsAreUpdated(t *testing.T) {
	result := renderTwo(t)
	// An outdated snapshot is replaced.
	writeSnapshot(t, t.Name(), "one.txt", "old\n")

	assertPasses(t, func(tb *recordingTB) { result.Snapshot(tb) })

	for path, want := range map[string]string{"one.txt": "one\n", "dir/two.txt": "two\n"} {
		b, err := os.ReadFile(filepath.Join("testdata", "snapshots", t.Name(), path))
		assert.NilError(t, err)
		assert.Equal(t, string(b), want)
	}
}
