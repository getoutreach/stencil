// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the shared module-deprecation fixer.

package modulefix_test

import (
	"testing"

	"go.yaml.in/yaml/v3"
	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/internal/lint/modulefix"
	"github.com/getoutreach/stencil/internal/lint/yamlfix"
)

func mappingFrom(t *testing.T, in string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	assert.NilError(t, yaml.Unmarshal([]byte(in), &doc))
	assert.Assert(t, len(doc.Content) == 1)
	return doc.Content[0]
}

func TestFixModulesPrereleaseTrueAddsChannel(t *testing.T) {
	root := mappingFrom(t, "modules:\n  - name: dep\n    prerelease: true\n")
	applied := modulefix.FixModules(root)
	assert.Equal(t, 1, len(applied))
	mods := root.Content[yamlfix.FindKey(root, "modules")+1]
	mod := mods.Content[0]
	assert.Equal(t, -1, yamlfix.FindKey(mod, "prerelease"))
	assert.Equal(t, "rc", mod.Content[yamlfix.FindKey(mod, "channel")+1].Value)
}

func TestFixModulesPrereleaseKeepsExistingChannel(t *testing.T) {
	root := mappingFrom(t, "modules:\n  - name: dep\n    channel: stable\n    prerelease: true\n")
	applied := modulefix.FixModules(root)
	assert.Equal(t, 1, len(applied))
	mods := root.Content[yamlfix.FindKey(root, "modules")+1]
	mod := mods.Content[0]
	assert.Equal(t, "stable", mod.Content[yamlfix.FindKey(mod, "channel")+1].Value)
	assert.Equal(t, -1, yamlfix.FindKey(mod, "prerelease"))
}

func TestFixModulesPrereleaseFalseDropped(t *testing.T) {
	root := mappingFrom(t, "modules:\n  - name: dep\n    prerelease: false\n")
	applied := modulefix.FixModules(root)
	assert.Equal(t, 1, len(applied))
	mods := root.Content[yamlfix.FindKey(root, "modules")+1]
	mod := mods.Content[0]
	assert.Equal(t, -1, yamlfix.FindKey(mod, "prerelease"))
	assert.Equal(t, -1, yamlfix.FindKey(mod, "channel"))
}

func TestFixModulesNoModulesNoOp(t *testing.T) {
	root := mappingFrom(t, "name: x\n")
	assert.Equal(t, 0, len(modulefix.FixModules(root)))
}

func TestFixModulesPathByIndexWhenNoName(t *testing.T) {
	root := mappingFrom(t, "modules:\n  - prerelease: true\n")
	applied := modulefix.FixModules(root)
	assert.Equal(t, 1, len(applied))
	assert.Equal(t, "modules[0].prerelease", applied[0].Path)
}

func TestModulePath(t *testing.T) {
	assert.Equal(t, "modules.github.com/x/y", modulefix.ModulePath("github.com/x/y", 0))
	assert.Equal(t, "modules[2]", modulefix.ModulePath("", 2))
}

// TestFixModulesConservativeSkips verifies the fixer leaves prerelease untouched
// when it is not a recognized boolean scalar, so an unexpected shape is reported
// by the linter rather than silently rewritten.
func TestFixModulesConservativeSkips(t *testing.T) {
	t.Run("sequence prerelease left alone", func(t *testing.T) {
		root := mappingFrom(t, "modules:\n  - name: m\n    prerelease: [x]\n")
		applied := modulefix.FixModules(root)
		mods := root.Content[yamlfix.FindKey(root, "modules")+1]
		mod := mods.Content[0]
		assert.Assert(t, yamlfix.FindKey(mod, "prerelease") >= 0) // left in place
		assert.Equal(t, 0, len(applied))
	})

	t.Run("non-bool scalar prerelease left alone", func(t *testing.T) {
		root := mappingFrom(t, "modules:\n  - name: m\n    prerelease: maybe\n")
		applied := modulefix.FixModules(root)
		mods := root.Content[yamlfix.FindKey(root, "modules")+1]
		mod := mods.Content[0]
		assert.Assert(t, yamlfix.FindKey(mod, "prerelease") >= 0) // left in place
		assert.Equal(t, 0, len(applied))
	})
}
