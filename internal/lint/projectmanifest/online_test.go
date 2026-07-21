// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for online project-manifest validation (argument index,
// O2-O4).

package projectmanifest

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/pkg/configuration"
)

// mod is a test helper building a ResolvedModule from an import path and a set
// of argument declarations (no filesystem/network — the manifest is in memory).
func mod(importPath string, args map[string]configuration.Argument, deps ...string) ResolvedModule {
	mf := &configuration.TemplateRepositoryManifest{Name: importPath, Arguments: args}
	for _, d := range deps {
		mf.Modules = append(mf.Modules, &configuration.TemplateRepository{Name: d})
	}
	return ResolvedModule{ImportPath: importPath, Manifest: mf}
}

func TestBuildArgIndexSimple(t *testing.T) {
	mods := []ResolvedModule{
		mod("github.com/x/a", map[string]configuration.Argument{
			"foo": {Schema: map[string]interface{}{"type": "string"}},
		}),
	}
	idx, findings := buildArgIndex(mods)
	assert.Equal(t, 0, len(findings))
	assert.Equal(t, 1, len(idx["foo"]))
	assert.Equal(t, "github.com/x/a", idx["foo"][0].importPath)
}

func TestBuildArgIndexFromRedirect(t *testing.T) {
	// module b's arg 'foo' has from: a; a declares foo; b lists a as a dep.
	mods := []ResolvedModule{
		mod("github.com/x/a", map[string]configuration.Argument{
			"foo": {Schema: map[string]interface{}{"type": "string"}},
		}),
		mod("github.com/x/b", map[string]configuration.Argument{
			"foo": {From: "github.com/x/a"},
		}, "github.com/x/a"),
	}
	idx, findings := buildArgIndex(mods)
	assert.Equal(t, 0, len(findings))
	// b's 'foo' resolves to a's declaration (schema present).
	var bDecl *declaration
	for i := range idx["foo"] {
		if idx["foo"][i].importPath == "github.com/x/b" {
			bDecl = &idx["foo"][i]
		}
	}
	assert.Assert(t, bDecl != nil)
	assert.Assert(t, len(bDecl.arg.Schema) > 0) // resolved to a's schema
}

func TestBuildArgIndexFromMissingDependency(t *testing.T) {
	// b's foo has from: a, but b does NOT list a as a dependency → O4 error.
	mods := []ResolvedModule{
		mod("github.com/x/a", map[string]configuration.Argument{
			"foo": {Schema: map[string]interface{}{"type": "string"}},
		}),
		mod("github.com/x/b", map[string]configuration.Argument{
			"foo": {From: "github.com/x/a"},
		}), // no deps
	}
	_, findings := buildArgIndex(mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, "arguments.foo", findings[0].Path)
	assert.Assert(t, findings[0].Message != "")
}

func TestBuildArgIndexFromReferencedModuleAbsent(t *testing.T) {
	// b's foo has from: c, b lists c as a dep, but c is not in the resolved set.
	mods := []ResolvedModule{
		mod("github.com/x/b", map[string]configuration.Argument{
			"foo": {From: "github.com/x/c"},
		}, "github.com/x/c"),
	}
	_, findings := buildArgIndex(mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, "arguments.foo", findings[0].Path)
}

func TestBuildArgIndexFromUnexposedArgument(t *testing.T) {
	// a is present + listed as dep, but a does not declare 'foo'.
	mods := []ResolvedModule{
		mod("github.com/x/a", map[string]configuration.Argument{}),
		mod("github.com/x/b", map[string]configuration.Argument{
			"foo": {From: "github.com/x/a"},
		}, "github.com/x/a"),
	}
	_, findings := buildArgIndex(mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, "arguments.foo", findings[0].Path)
}
