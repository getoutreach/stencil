// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for finding templates.

package rendertest_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/getoutreach/stencil/pkg/rendertest"
	"gotest.tools/v3/assert"
)

// writeFiles creates empty files at paths relative to dir, and the
// directories that they need.
func writeFiles(t *testing.T, dir string, paths ...string) {
	t.Helper()

	for _, p := range paths {
		full := filepath.Join(dir, p)
		assert.NilError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		assert.NilError(t, os.WriteFile(full, nil, 0o644))
	}
}

func TestFindTemplates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFiles(t, dir,
		"a.tpl",
		"_helpers.tpl",
		"notes.txt",
		"sub/c.tpl",
		"sub/deep/d.tpl",
		// Directories with a name that starts with a dot hold generated
		// files, such as old snapshots.
		".snapshots/e.tpl",
		"sub/.snapshots/f.tpl",
		// A rendertest snapshot directory is in testdata.
		"testdata/snapshots/TestA/g.tpl",
		"sub/testdata/h.tpl",
	)

	prefix := filepath.ToSlash(dir) + "/"
	assert.DeepEqual(t, rendertest.FindTemplates(t, dir), []string{
		prefix + "_helpers.tpl",
		prefix + "a.tpl",
		prefix + "sub/c.tpl",
		prefix + "sub/deep/d.tpl",
	})
}

func TestFindTemplatesInRootThatStartsWithADot(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), ".root")
	writeFiles(t, dir, "a.tpl")

	assert.DeepEqual(t, rendertest.FindTemplates(t, dir), []string{filepath.ToSlash(dir) + "/a.tpl"})
}

func TestFindTemplatesInWorkingDirectory(t *testing.T) {
	t.Parallel()

	// The paths of a relative dir are relative to the working directory, so
	// they work with WithTemplates.
	found := rendertest.FindTemplates(t, ".")
	assert.Assert(t, slices.Contains(found, "args.tpl"), "found: %q", found)
}

func TestFindTemplatesFailsWhenTheDirectoryDoesNotExist(t *testing.T) {
	t.Parallel()

	assertFatal(t, func(tb *recordingTB) {
		rendertest.FindTemplates(tb, filepath.Join(t.TempDir(), "missing"))
	})
}
