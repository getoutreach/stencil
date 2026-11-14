// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for using modules from concurrent
// renders and resolutions.

package modules_test

import (
	"bytes"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gotest.tools/v3/assert"
)

func TestCloneHasItsOwnTemplateSet(t *testing.T) {
	m, err := modulestest.NewModuleFromTemplates(&configuration.TemplateRepositoryManifest{Name: "testing"})
	assert.NilError(t, err)

	_, err = m.GetTemplate().New("original").Parse("original")
	assert.NilError(t, err)
	// text/template's Clone never fails, also after a template is executed.
	assert.NilError(t, m.GetTemplate().ExecuteTemplate(&bytes.Buffer{}, "original", nil))

	c := m.Clone()
	assert.Equal(t, c.Name, m.Name)
	assert.Equal(t, c.URI, m.URI)
	assert.Equal(t, c.Version, m.Version)
	assert.Assert(t, c.GetTemplate().Lookup("original") != nil, "expected the clone to have the original templates")

	_, err = c.GetTemplate().New("clone").Parse("clone")
	assert.NilError(t, err)
	assert.Assert(t, m.GetTemplate().Lookup("clone") == nil, "expected the clone not to change the original")
}

func TestCloneSharesTheFilesystemAndManifest(t *testing.T) {
	ctx := t.Context()
	fs := memfs.New()
	writeManifest(t, fs)

	m := modules.NewWithFS(ctx, "test", fs)
	c := m.Clone()

	cfs, err := c.GetFS(ctx)
	assert.NilError(t, err)
	assert.Equal(t, cfs, fs, "expected the clone to share the filesystem")

	_, err = c.Manifest(ctx)
	assert.NilError(t, err)

	// The original reads the manifest that the clone cached.
	assert.NilError(t, fs.Remove("manifest.yaml"))
	mf, err := m.Manifest(ctx)
	assert.NilError(t, err, "expected the original to use the manifest cached by the clone")
	assert.Equal(t, mf.Name, "test")
}

// newReplacedModule returns an in-memory module whose manifest imports deps.
func newReplacedModule(t *testing.T, name string, deps ...string) *modules.Module {
	t.Helper()

	mf := &configuration.TemplateRepositoryManifest{Name: name}
	for _, dep := range deps {
		mf.Modules = append(mf.Modules, &configuration.TemplateRepository{Name: dep})
	}

	m, err := modulestest.NewModuleFromTemplates(mf)
	assert.NilError(t, err)
	return m
}

// TestReplacedModuleInADiamondIsResolvedOnce resolves a diamond of replaced
// modules: top imports a and b, which both import shared. Under -race, it
// also checks that the resolve loop and the workers do not race on whether a
// module is to be resolved again.
func TestReplacedModuleInADiamondIsResolvedOnce(t *testing.T) {
	for range 20 {
		log, hook := test.NewNullLogger()
		log.SetLevel(logrus.DebugLevel)

		mods, err := modules.GetModulesForService(t.Context(), &modules.ModuleResolveOptions{
			Module: newReplacedModule(t, "top", "a", "b"),
			Replacements: map[string]*modules.Module{
				"a":      newReplacedModule(t, "a", "shared"),
				"b":      newReplacedModule(t, "b", "shared"),
				"shared": newReplacedModule(t, "shared"),
			},
			Log: log,
		})
		assert.NilError(t, err)

		names := map[string]bool{}
		for _, m := range mods {
			names[m.Name] = true
		}
		assert.DeepEqual(t, names, map[string]bool{"top": true, "a": true, "b": true, "shared": true})

		resolved := 0
		for _, e := range hook.AllEntries() {
			if e.Message == "resolved module" && e.Data["module"] == "shared" {
				resolved++
			}
		}
		assert.Equal(t, resolved, 1, "expected the shared module to be resolved once")
	}
}
