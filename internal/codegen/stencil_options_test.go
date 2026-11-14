// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the Stencil options that
// replace file, git and box access.

package codegen

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/getoutreach/gobox/pkg/box"
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

// newMemFS returns an in-memory filesystem with one file.
func newMemFS(t *testing.T, path, contents string) billy.Filesystem {
	t.Helper()

	fs := memfs.New()
	assert.NilError(t, util.WriteFile(fs, path, []byte(contents), 0o644))
	return fs
}

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

			files, err := renderTemplate(t, tt.template, WithFS(newMemFS(t, tt.path, existingBlockFile)))
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

func TestTemplateFunctionsUseTheFilesystem(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     string
	}{
		{name: "stencil.ReadFile", template: `{{ stencil.ReadFile "existing.txt" }}`, want: existingBlockFile},
		{name: "stencil.Exists", template: `{{ stencil.Exists "existing.txt" }} {{ stencil.Exists "on-disk.txt" }}`, want: "true false"},
		{name: "stencil.ReadBlocks", template: `{{ index (stencil.ReadBlocks "existing.txt") "custom" }}`, want: "existing value"},
		{name: "stencil.ReadBlocks of a missing file", template: `{{ len (stencil.ReadBlocks "on-disk.txt") }}`, want: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// on-disk.txt is only in the working directory, so a function
			// that reads the real filesystem finds it.
			t.Chdir(t.TempDir())
			assert.NilError(t, os.WriteFile("on-disk.txt", []byte(existingBlockFile), 0o644))

			files, err := renderTemplate(t, tt.template, WithFS(newMemFS(t, "existing.txt", existingBlockFile)))
			assert.NilError(t, err)
			assert.Equal(t, files[0].String(), tt.want)
		})
	}
}

func TestFileStaticUsesTheFilesystem(t *testing.T) {
	t.Chdir(t.TempDir())

	files, err := renderTemplate(t, `{{ $_ := file.Static }}`, WithFS(newMemFS(t, "test-template", "existing")))
	assert.NilError(t, err)
	assert.Assert(t, files[0].Skipped, "expected a static file that exists in the filesystem to be skipped")

	files, err = renderTemplate(t, `{{ $_ := file.Static }}`, WithFS(memfs.New()))
	assert.NilError(t, err)
	assert.Assert(t, !files[0].Skipped, "expected a static file that does not exist to be rendered")
}

func TestFileRemoveAllUsesTheFilesystem(t *testing.T) {
	t.Chdir(t.TempDir())
	assert.NilError(t, os.MkdirAll("dir", 0o755))
	assert.NilError(t, os.WriteFile("dir/file.txt", []byte("on disk"), 0o644))

	fs := newMemFS(t, "dir/file.txt", "in memory")

	_, err := renderTemplate(t, `{{ file.RemoveAll "dir" }}`, WithFS(fs))
	assert.NilError(t, err)

	_, err = fs.Stat("dir")
	assert.ErrorIs(t, err, os.ErrNotExist, "expected the directory to be removed from the filesystem")
	_, err = os.Stat("dir/file.txt")
	assert.NilError(t, err, "expected the directory on disk to be kept")
}

func TestWithGitInfoAndWithBoxConfig(t *testing.T) {
	// The working directory is not a git repository, so reading git data
	// would leave the ref empty and fail the default branch lookup.
	t.Chdir(t.TempDir())

	files, err := renderTemplate(t,
		`{{ .Git.Ref }} {{ .Git.Commit }} {{ .Git.Dirty }} {{ .Git.DefaultBranch }} {{ .Runtime.Box.Org }}`,
		WithGitInfo("refs/heads/feature", "abc123", "trunk", true),
		WithBoxConfig(&box.Config{Org: "example"}),
	)
	assert.NilError(t, err)
	assert.Equal(t, files[0].String(), "refs/heads/feature abc123 true trunk example")
}

// errCommandFailed is the error of a failed command.
var errCommandFailed = errors.New("failed")

// commandRecorder records the commands that it is asked to run.
type commandRecorder struct {
	names, commands []string
	err             error
}

func (r *commandRecorder) Run(_ context.Context, name, command string) error {
	r.names = append(r.names, name)
	r.commands = append(r.commands, command)
	return r.err
}

func TestWithCommandRunner(t *testing.T) {
	fs := createFakeModuleFSWithManifest(t, "name: testing\npostRunCommand:\n  - name: hello\n    command: echo hello\n")
	newStencil := func(r CommandRunner) *Stencil {
		return NewStencil(&configuration.ServiceManifest{Name: "test"},
			[]*modules.Module{modules.NewWithFS(t.Context(), "testing", fs)}, logrus.New(), WithCommandRunner(r))
	}

	r := &commandRecorder{}
	assert.NilError(t, newStencil(r).PostRun(t.Context(), logrus.New()))
	assert.DeepEqual(t, r.names, []string{"hello"})
	assert.DeepEqual(t, r.commands, []string{"echo hello"})

	r = &commandRecorder{err: errCommandFailed}
	assert.ErrorContains(t, newStencil(r).PostRun(t.Context(), logrus.New()), "failed to run post run command")
}

func TestDefaultCommandRunner(t *testing.T) {
	assert.NilError(t, defaultCommandRunner{}.Run(t.Context(), "true", "true"))
	assert.ErrorContains(t, defaultCommandRunner{}.Run(t.Context(), "false", "exit 3"), "exit status 3")
}

func TestWithModuleHook(t *testing.T) {
	st := NewStencil(&configuration.ServiceManifest{Name: "test"},
		[]*modules.Module{newTestModule(t, `{{ range stencil.GetModuleHook "seeded" }}{{ . }} {{ end }}`)}, logrus.New(),
		WithModuleHook("testing", "seeded", []any{"a"}),
		WithModuleHook("testing", "seeded", []any{"b"}),
	)
	tpls, err := st.Render(t.Context(), logrus.New())
	assert.NilError(t, err)
	// The render sorts hook values by their hash.
	assert.Equal(t, tpls[0].Files[0].String(), "b a ")
	assert.DeepEqual(t, st.ModuleHooks(), map[string][]any{"testing/seeded": {"b", "a"}})
}

func TestExportedParseBlocks(t *testing.T) {
	blocks, err := ParseBlocks([]byte(existingBlockFile))
	assert.NilError(t, err)
	assert.DeepEqual(t, blocks, map[string]string{"custom": "existing value"})

	_, err = ParseBlocks([]byte("## <<Stencil::Block(dangling)>>\n"))
	assert.ErrorIs(t, err, ErrBlockParse)
}
