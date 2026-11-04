// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file tests the caching behavior of Module.Manifest.

package modules_test

import (
	"context"
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
