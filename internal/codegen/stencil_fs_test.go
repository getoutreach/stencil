// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the filesystem that templates
// read existing files from.

package codegen

import (
	"os"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

// blockTemplate renders the "custom" block of the file that it writes.
const blockTemplate = `## <<Stencil::Block(custom)>>
{{ file.Block "custom" }}
## <</Stencil::Block>>
`

// existingBlockFile is an existing file with a value in the "custom" block.
const existingBlockFile = `## <<Stencil::Block(custom)>>
existing value
## <</Stencil::Block>>
`

// renderTemplate renders the module from newTestModule with the given
// template contents and options, and returns the rendered files.
func renderTemplate(t *testing.T, contents string, opts ...StencilOption) ([]*File, error) {
	t.Helper()

	st := NewStencil(&configuration.ServiceManifest{Name: "test"},
		[]*modules.Module{newTestModule(t, contents)}, logrus.New(), opts...)
	tpls, err := st.Render(t.Context(), logrus.New())
	if err != nil {
		return nil, err
	}

	assert.Equal(t, len(tpls), 1, "expected one rendered template")
	return tpls[0].Files, nil
}

func TestWithFSReadsBlocksFromTheFilesystem(t *testing.T) {
	tests := []struct {
		name     string
		template string
		path     string
		want     string // defaults to existingBlockFile
	}{
		{name: "default file", template: blockTemplate, path: "test-template"},
		{name: "file.SetPath", template: `{{ $_ := file.SetPath "set-path.txt" }}` + blockTemplate, path: "set-path.txt"},
		{
			name:     "file.Create",
			template: `{{ file.Create "created.txt" 0o644 now }}{{ file.Block "custom" | file.SetContents }}`,
			path:     "created.txt",
			want:     "existing value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The working directory has no files, so blocks can come only
			// from the in-memory filesystem.
			dir := t.TempDir()
			t.Chdir(dir)

			fs := memfs.New()
			assert.NilError(t, util.WriteFile(fs, tt.path, []byte(existingBlockFile), 0o644))

			files, err := renderTemplate(t, tt.template, WithFS(fs))
			assert.NilError(t, err)
			assert.Equal(t, len(files), 1)
			assert.Equal(t, files[0].Name(), tt.path)
			want := tt.want
			if want == "" {
				want = existingBlockFile
			}
			assert.Equal(t, files[0].String(), want)

			entries, err := os.ReadDir(dir)
			assert.NilError(t, err)
			assert.Equal(t, len(entries), 0, "expected no files on disk")
		})
	}
}

func TestDefaultFSIsTheWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	assert.NilError(t, os.WriteFile("test-template", []byte(existingBlockFile), 0o644))

	files, err := renderTemplate(t, blockTemplate)
	assert.NilError(t, err)
	assert.Equal(t, files[0].String(), existingBlockFile)
}

func TestFilePathCannotLeaveTheWorkingDirectory(t *testing.T) {
	t.Chdir(t.TempDir())

	_, err := renderTemplate(t, `{{ $_ := file.SetPath "../outside.txt" }}`)
	assert.ErrorIs(t, err, billy.ErrCrossedBoundary)
}
