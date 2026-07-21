// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for online project-manifest validation (argument index,
// O2-O4).

package projectmanifest

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	lint "github.com/getoutreach/stencil/internal/lint"
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

func TestCheckArgumentsO2Pass(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Schema: map[string]interface{}{"type": "string"}},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Arguments: map[string]interface{}{"foo": "hello"},
	}}
	findings := checkArguments(res, idx)
	assert.Equal(t, 0, len(findings))
}

func TestCheckArgumentsO2Violation(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Schema: map[string]interface{}{"type": "string"}},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Arguments: map[string]interface{}{"foo": 123}, // number, want string
	}}
	findings := checkArguments(res, idx)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, lint.SeverityError, findings[0].Severity)
	assert.Equal(t, "arguments.foo", findings[0].Path)
}

func TestCheckArgumentsO3RequiredMissing(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Required: true},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{Arguments: nil}}
	findings := checkArguments(res, idx)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, "arguments.foo", findings[0].Path)
	assert.Assert(t, contains(findings[0].Message, "required"))
}

func TestCheckArgumentsO3RequiredSatisfiedByValue(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Required: true},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Arguments: map[string]interface{}{"foo": "x"},
	}}
	assert.Equal(t, 0, len(checkArguments(res, idx)))
}

func TestCheckArgumentsO3RequiredSatisfiedByDefault(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Required: true, Default: "d"},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{Arguments: nil}}
	assert.Equal(t, 0, len(checkArguments(res, idx)))
}

func TestCheckArgumentsExplicitNullIsNotProvided(t *testing.T) {
	// foo: null must count as NOT provided → required arg still missing (O3).
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Required: true},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Arguments: map[string]interface{}{"foo": nil},
	}}
	findings := checkArguments(res, idx)
	assert.Equal(t, 1, len(findings)) // O3 fires
}

func TestCheckArgumentsOptionalOmittedNoFindings(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Schema: map[string]interface{}{"type": "string"}}, // optional
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{Arguments: nil}}
	assert.Equal(t, 0, len(checkArguments(res, idx)))
}

func TestCheckArgumentsHermeticExternalRef(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Schema: map[string]interface{}{"$ref": "https://example.com/s.json"}},
	})}
	idx, _ := buildArgIndex(mods)
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Arguments: map[string]interface{}{"foo": "x"},
	}}
	findings := checkArguments(res, idx)
	assert.Equal(t, 1, len(findings)) // external $ref rejected → O2 error, no fetch
	assert.Equal(t, "arguments.foo", findings[0].Path)
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
