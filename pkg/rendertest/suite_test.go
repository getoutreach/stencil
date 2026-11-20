// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for loading a Suite. The paths are
// relative to testdata/templates, the working directory set by TestMain.

package rendertest

import (
	"path/filepath"
	"testing"

	"github.com/getoutreach/gobox/pkg/cfg"
	"gotest.tools/v3/assert"
)

// newTestSuite returns a Suite for the module under test, with the local
// directories of its dependencies.
func newTestSuite() *Suite {
	return NewSuite(nil,
		WithManifestPath("../module/manifest.yaml"),
		WithModuleReplacements(map[string]string{
			"github.com/example/dep":     "../dep",
			"github.com/example/another": "../another",
		}),
	)
}

func TestLoadResolvesReplacedDependencies(t *testing.T) {
	s := newTestSuite()
	assert.NilError(t, s.load(t.Context()))

	assert.Equal(t, s.name, "github.com/example/module")
	assert.Equal(t, len(s.deps), 2)
	assert.Equal(t, s.deps[0].Version, "local")

	fs, err := s.deps[0].GetFS(t.Context())
	assert.NilError(t, err)
	_, err = fs.Stat("templates")
	assert.NilError(t, err, "expected the dependency to have an empty templates directory")
}

func TestLoadSortsDependencies(t *testing.T) {
	// The suite relies on the engine to sort the modules. The engine once
	// returned them in a random order, so one load could be sorted by chance.
	for range 20 {
		s := newTestSuite()
		assert.NilError(t, s.load(t.Context()))

		names := make([]string, 0, len(s.deps))
		for _, d := range s.deps {
			names = append(names, d.Name)
		}
		assert.DeepEqual(t, names, []string{"github.com/example/another", "github.com/example/dep"})
	}
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

func TestLoadWithAllDependenciesReplacedDoesNotAskForAToken(t *testing.T) {
	tokenAsked := false
	s := newTestSuite()
	s.getToken = func() (cfg.SecretData, error) {
		tokenAsked = true
		return "", nil
	}
	assert.NilError(t, s.load(t.Context()))

	assert.Assert(t, !tokenAsked, "expected no GitHub token to be asked for")
}
