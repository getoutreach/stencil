// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for the snapshot tests.

package rendertest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getoutreach/stencil/pkg/rendertest"
	"gotest.tools/v3/assert"
)

// renderTwo renders two.tpl, which writes one.txt and dir/two.txt, and then
// changes to an empty working directory for the snapshots.
func renderTwo(t *testing.T) *rendertest.Result {
	t.Helper()

	result := suite.NewTemplate("two.tpl").Run(t)
	assert.NilError(t, result.Error())
	t.Chdir(t.TempDir())
	return result
}

// writeSnapshot writes a snapshot of the test named name.
func writeSnapshot(t *testing.T, name, path, contents string) {
	t.Helper()

	p := filepath.Join("testdata", "snapshots", name, path)
	assert.NilError(t, os.MkdirAll(filepath.Dir(p), 0o755))
	assert.NilError(t, os.WriteFile(p, []byte(contents), 0o644))
}
