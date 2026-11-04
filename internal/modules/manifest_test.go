// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file tests the caching behavior of Module.Manifest.

package modules_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
)

// countingFS counts how many times manifest.yaml is opened.
type countingFS struct {
	billy.Filesystem

	opens atomic.Int32
}

func (c *countingFS) Open(name string) (billy.File, error) {
	if name == "manifest.yaml" {
		c.opens.Add(1)
	}
	return c.Filesystem.Open(name)
}

func writeManifest(t *testing.T, fs billy.Filesystem) {
	t.Helper()
	assert.NilError(t, util.WriteFile(fs, "manifest.yaml", []byte(testManifest), 0o644))
}

const testManifest = `name: test
modules:
  - name: github.com/getoutreach/stencil-base
    version: main
postRunCommand:
  - name: hello
    command: echo hello
arguments:
  foo:
    schema:
      type: object
      properties:
        list:
          type: array
    default:
      key: [a, b]
    values: [x, y]
`

func TestManifestIsParsedOncePerModule(t *testing.T) {
	ctx := context.Background()
	fs := &countingFS{Filesystem: memfs.New()}
	writeManifest(t, fs)
	m := modules.NewWithFS(ctx, "test", fs)

	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() {
			mf, err := m.Manifest(ctx)
			assert.Check(t, err)
			assert.Check(t, mf.Name == "test")
		})
	}
	wg.Wait()

	assert.Equal(t, fs.opens.Load(), int32(1), "manifest.yaml should be opened once")
}

func TestManifestErrorsAreNotCached(t *testing.T) {
	ctx := context.Background()
	fs := memfs.New()
	m := modules.NewWithFS(ctx, "test", fs)

	_, err := m.Manifest(ctx)
	assert.Assert(t, err != nil, "expected an error when manifest.yaml is missing")

	writeManifest(t, fs)
	mf, err := m.Manifest(ctx)
	assert.NilError(t, err)
	assert.Equal(t, mf.Name, "test")
}

func TestManifestNameMismatchIsAnError(t *testing.T) {
	ctx := context.Background()
	fs := memfs.New()
	writeManifest(t, fs)
	m := modules.NewWithFS(ctx, "other", fs)

	_, err := m.Manifest(ctx)
	assert.ErrorIs(t, err, modules.ErrImportPathMismatch)
}

func TestManifestReturnsIndependentCopies(t *testing.T) {
	ctx := context.Background()
	fs := memfs.New()
	writeManifest(t, fs)
	m := modules.NewWithFS(ctx, "test", fs)

	first, err := m.Manifest(ctx)
	assert.NilError(t, err)

	// modify everything reachable from the returned manifest
	first.Name = "changed"
	first.Modules[0].Version = "changed"
	first.Modules = append(first.Modules, &configuration.TemplateRepository{Name: "extra"})
	first.PostRunCommand[0].Command = "changed"
	arg := first.Arguments["foo"]
	arg.Schema["type"] = "changed"
	arg.Schema["properties"].(map[string]any)["list"].(map[string]any)["type"] = "changed"
	arg.Default.(map[string]any)["key"].([]any)[0] = "changed"
	//nolint:staticcheck // Why: Values is deprecated but must still be copied.
	arg.Values[0] = "changed"
	first.Arguments["extra"] = configuration.Argument{}

	second, err := m.Manifest(ctx)
	assert.NilError(t, err)

	fresh := modules.NewWithFS(ctx, "test", fs)
	want, err := fresh.Manifest(ctx)
	assert.NilError(t, err)
	assert.DeepEqual(t, second, want, cmpopts.IgnoreUnexported(configuration.TemplateRepositoryTypes{}))
}

func TestManifestAndGetFSAreSafeToCallConcurrently(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	assert.NilError(t, os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(testManifest), 0o600))

	// A local module sets its filesystem on the first call to GetFS.
	m, err := modules.New(ctx, dir, &configuration.TemplateRepository{Name: "test"})
	assert.NilError(t, err)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			if i%2 == 0 {
				_, err := m.GetFS(ctx)
				assert.Check(t, err)
				return
			}
			_, err := m.Manifest(ctx)
			assert.Check(t, err)
		})
	}
	wg.Wait()
}

func TestFailedCloneIsRetried(t *testing.T) {
	ctx := context.Background()
	// Nothing listens on port 1, so the clone fails right away.
	m, err := modules.New(ctx, "git://127.0.0.1:1/getoutreach/missing.git",
		&configuration.TemplateRepository{Name: "github.com/getoutreach/missing", Version: "v1.0.0"})
	assert.NilError(t, err)
	t.Cleanup(func() { os.RemoveAll(m.FSCacheDir()) })

	_, err = m.Manifest(ctx)
	assert.Assert(t, err != nil, "expected the clone to fail")
	_, err = os.Stat(m.FSCacheDir())
	assert.Assert(t, os.IsNotExist(err), "expected the partial clone to be removed")

	// A second call tries the clone again instead of using what the first left behind.
	_, err = m.GetFS(ctx)
	assert.Assert(t, err != nil, "expected the clone to be tried again")
}

func TestArgument(t *testing.T) {
	ctx := context.Background()
	fs := memfs.New()
	writeManifest(t, fs)
	m := modules.NewWithFS(ctx, "test", fs)

	arg, ok, err := m.Argument(ctx, "foo")
	assert.NilError(t, err)
	assert.Assert(t, ok)
	assert.Equal(t, arg.Schema["type"], "object")

	// The result is a copy.
	arg.Schema["type"] = "changed"
	again, _, err := m.Argument(ctx, "foo")
	assert.NilError(t, err)
	assert.Equal(t, again.Schema["type"], "object")

	_, ok, err = m.Argument(ctx, "missing")
	assert.NilError(t, err)
	assert.Assert(t, !ok)

	_, _, err = modules.NewWithFS(ctx, "test", memfs.New()).Argument(ctx, "foo")
	assert.Assert(t, err != nil, "expected an error when manifest.yaml is missing")
}

func TestImportsModule(t *testing.T) {
	ctx := context.Background()
	fs := memfs.New()
	writeManifest(t, fs)
	m := modules.NewWithFS(ctx, "test", fs)

	imports, err := m.ImportsModule(ctx, "github.com/getoutreach/stencil-base")
	assert.NilError(t, err)
	assert.Assert(t, imports)

	imports, err = m.ImportsModule(ctx, "github.com/getoutreach/other")
	assert.NilError(t, err)
	assert.Assert(t, !imports)

	_, err = modules.NewWithFS(ctx, "test", memfs.New()).ImportsModule(ctx, "x")
	assert.Assert(t, err != nil, "expected an error when manifest.yaml is missing")
}
