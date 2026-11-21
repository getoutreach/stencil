// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains FindTemplates, which finds the templates of
// a module.

package rendertest

import (
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// FindTemplates returns the paths of the .tpl files under dir, in sorted
// order and with forward slashes. It skips directories with a name that
// starts with a dot, such as .snapshots, and directories named testdata, where
// rendertest keeps its snapshots. A failure to read dir fails the test.
//
// Each path is dir joined with the path of the file, so "." gives paths such
// as "a.tpl" and "sub/b.tpl". To use the paths with WithTemplates, dir must be
// inside the working directory of the test, because an absolute dir gives
// absolute paths. The result has the template under test too, which is safe
// to pass to WithTemplates.
func FindTemplates(tb testing.TB, dir string) []string {
	tb.Helper()

	var found []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		switch {
		case d.IsDir() && p != dir && (strings.HasPrefix(name, ".") || name == "testdata"):
			return filepath.SkipDir
		case !d.IsDir() && filepath.Ext(p) == ".tpl":
			found = append(found, filepath.ToSlash(p))
		}
		return nil
	})
	if err != nil {
		tb.Fatalf("rendertest: failed to find the templates in %q: %v", dir, err)
	}

	slices.Sort(found)
	return found
}
