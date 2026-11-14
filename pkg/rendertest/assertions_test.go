// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the assertions and the git
// fixtures.

package rendertest_test

import (
	"io/fs"
	"os"
	"testing"
	"text/template"

	"github.com/getoutreach/stencil/pkg/rendertest"
	"gotest.tools/v3/assert"
)

// assertErrors checks that fn reports a failure with Error, not Fatal.
func assertErrors(t *testing.T, fn func(tb *recordingTB)) {
	t.Helper()

	r := record(t, fn)
	assert.Assert(t, !r.fatal, "expected Error, got Fatal: %q", r.errors)
	assert.Equal(t, len(r.errors), 1, "expected one failure")
}

func TestResultAssertions(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("files.tpl").Run(t)

	assertPasses(t, func(tb *recordingTB) { result.HasNoError(tb) })
	assertPasses(t, func(tb *recordingTB) { assert.Equal(t, result.HasFile(tb, "kept.txt").Path(), "kept.txt") })
	assertFatal(t, func(tb *recordingTB) { result.HasFile(tb, "missing.txt") })
	assertPasses(t, func(tb *recordingTB) { result.HasNoFile(tb, "deleted.txt") })
	assertErrors(t, func(tb *recordingTB) { result.HasNoFile(tb, "kept.txt") })
	assertPasses(t, func(tb *recordingTB) { result.AssertDeleted(tb, "deleted.txt") })
	assertErrors(t, func(tb *recordingTB) { result.AssertDeleted(tb, "kept.txt") })
	assertPasses(t, func(tb *recordingTB) { result.AssertSkipped(tb, "skipped.txt") })
	assertErrors(t, func(tb *recordingTB) { result.AssertSkipped(tb, "kept.txt") })
	assertFatal(t, func(tb *recordingTB) { result.HasError(tb) })
}

func TestResultAssertionsOnARenderError(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("args.tpl").Run(t)

	assertErrors(t, func(tb *recordingTB) { result.HasNoError(tb) })
	assertFatal(t, func(tb *recordingTB) { result.HasFile(tb, "args") })
	assertPasses(t, func(tb *recordingTB) { assert.Assert(t, result.HasError(tb).Error() != nil) })
}

func TestErrorResult(t *testing.T) {
	t.Parallel()

	er := suite.NewTemplate("existing.tpl").Run(t).HasError(t)

	assertPasses(t, func(tb *recordingTB) { er.Contains(tb, "input.txt") })
	assertErrors(t, func(tb *recordingTB) { er.Contains(tb, "not in the error") })
	assertPasses(t, func(tb *recordingTB) { er.Is(tb, er.Error()) })
	assertFatal(t, func(tb *recordingTB) { er.Is(tb, os.ErrPermission) })

	var execErr template.ExecError
	assertPasses(t, func(tb *recordingTB) { er.As(tb, &execErr) })
	assert.Equal(t, execErr.Name, "github.com/example/module/existing.tpl")
	var pathErr *fs.PathError
	assertFatal(t, func(tb *recordingTB) { er.As(tb, &pathErr) })
}

func TestFileResultAssertions(t *testing.T) {
	t.Parallel()

	f := suite.NewTemplate("blocks.tpl").WithBlocks(map[string]string{"custom": "value"}).Run(t).HasFile(t, "blocks")

	assertPasses(t, func(tb *recordingTB) { f.Contains(tb, "value") })
	assertErrors(t, func(tb *recordingTB) { f.Contains(tb, "missing") })
	assertPasses(t, func(tb *recordingTB) { f.NotContains(tb, "missing") })
	assertErrors(t, func(tb *recordingTB) { f.NotContains(tb, "value") })
	assertPasses(t, func(tb *recordingTB) {
		f.ContentsEqual(tb, "## <<Stencil::Block(custom)>>\nvalue\n## <</Stencil::Block>>\n")
	})
	assertErrors(t, func(tb *recordingTB) { f.ContentsEqual(tb, "other") })
	assertPasses(t, func(tb *recordingTB) { f.ModeEqual(tb, f.Mode()) })
	assertErrors(t, func(tb *recordingTB) { f.ModeEqual(tb, 0o777) })
	assertPasses(t, func(tb *recordingTB) { assert.Equal(t, f.HasBlock(tb, "custom"), "value") })
	assertFatal(t, func(tb *recordingTB) { f.HasBlock(tb, "missing") })
}

func TestFileResultWriteTo(t *testing.T) {
	t.Parallel()

	f := suite.NewTemplate("files.tpl").Run(t).HasFile(t, "kept.txt")

	var p string
	t.Run("write", func(t *testing.T) {
		p = f.WriteTo(t)

		b, err := os.ReadFile(p)
		assert.NilError(t, err)
		assert.Equal(t, string(b), "kept")
		info, err := os.Stat(p)
		assert.NilError(t, err)
		assert.Equal(t, info.Mode().Perm(), os.FileMode(0o600))
	})

	_, err := os.Stat(p)
	assert.ErrorIs(t, err, os.ErrNotExist, "expected the file to be removed when the test ends")
}

func TestGitValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts []rendertest.GitOption
		want string
	}{
		{name: "defaults", want: "  main false"},
		{
			name: "all values",
			opts: []rendertest.GitOption{
				rendertest.Ref("refs/heads/feature"), rendertest.Commit("abc123"),
				rendertest.DefaultBranch("trunk"), rendertest.Dirty(),
			},
			want: "refs/heads/feature abc123 trunk true",
		},
		{name: "clean", opts: []rendertest.GitOption{rendertest.Dirty(), rendertest.Clean()}, want: "  main false"},
		{
			name: "release",
			opts: []rendertest.GitOption{rendertest.Dirty(), rendertest.Release("v1.2.3")},
			want: "refs/tags/v1.2.3  main false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := suite.NewTemplate("git.tpl").WithGitValues(tt.opts...).Run(t).HasFile(t, "git")
			f.ContentsEqual(t, tt.want+"\n")
		})
	}
}
