// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for loading a Suite. The paths are
// relative to testdata/templates, the working directory set by TestMain.

package rendertest

import (
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
)

func TestLoadResolvesReplacedDependencies(t *testing.T) {
	s := NewSuite(nil,
		WithManifestPath("../module/manifest.yaml"),
		WithModuleReplacements(map[string]string{"github.com/example/dep": "../dep"}),
	)
	assert.NilError(t, s.load(t.Context()))

	assert.Equal(t, s.name, "github.com/example/module")
	assert.Equal(t, len(s.deps), 1)
	assert.Equal(t, s.deps[0].Name, "github.com/example/dep")
	assert.Equal(t, s.deps[0].Version, "local")

	fs, err := s.deps[0].GetFS(t.Context())
	assert.NilError(t, err)
	_, err = fs.Stat("templates")
	assert.NilError(t, err, "expected the dependency to have an empty templates directory")
}

func TestLoadWithoutModuleResolution(t *testing.T) {
	s := NewSuite(nil, WithManifestPath("../module/manifest.yaml"), WithoutModuleResolution())
	assert.NilError(t, s.load(t.Context()))
	assert.Equal(t, len(s.deps), 0)
}

func TestLoadFailsWithoutAManifest(t *testing.T) {
	s := NewSuite(nil, WithManifestPath("../missing.yaml"))
	assert.ErrorContains(t, s.load(t.Context()), "failed to read the module manifest")
	assert.Assert(t, !s.loaded)
}

func TestDefaultManifestPath(t *testing.T) {
	p, err := defaultManifestPath(t.Context())
	assert.NilError(t, err)

	// TestMain changes to testdata/templates, four directories under the
	// root of this Go module.
	want, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "manifest.yaml"))
	assert.NilError(t, err)
	assert.Equal(t, p, want)
}
