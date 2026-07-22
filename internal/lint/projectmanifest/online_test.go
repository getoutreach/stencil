// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for online project-manifest validation (argument index,
// O2-O4).

package projectmanifest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"gotest.tools/v3/assert"

	lint "github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// renderOnlineFindings formats findings deterministically for snapshotting,
// mirroring projectmanifest_test.go's renderFindings (duplicated here because
// online_test.go is the internal package, not the _test package).
func renderOnlineFindings(findings []lint.Finding) string {
	if len(findings) == 0 {
		return "(no findings)\n"
	}
	sevWidth, locWidth := 0, 0
	locs := make([]string, len(findings))
	for i := range findings {
		locs[i] = fmt.Sprintf("%s:%d", findings[i].Path, findings[i].Line)
		if l := len(string(findings[i].Severity)); l > sevWidth {
			sevWidth = l
		}
		if l := len(locs[i]); l > locWidth {
			locWidth = l
		}
	}
	var b strings.Builder
	for i := range findings {
		fmt.Fprintf(&b, "%-*s  %-*s  %s\n",
			sevWidth, findings[i].Severity, locWidth, locs[i], findings[i].Message)
	}
	return b.String()
}

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

// TestBuildArgIndexOwnerNotAliased guards the pointer-aliasing footgun: each
// declaration's owner must be its own ResolvedModule, so a from: dependency
// check reads the correct Modules slice. Module b (arg foo from a, lists a) is
// fine; module c (arg bar from a, does NOT list a) must produce exactly one O4
// finding attributed to bar. If owners were aliased to the last module, both
// would read the same Modules and the O4 result would be wrong.
func TestBuildArgIndexOwnerNotAliased(t *testing.T) {
	mods := []ResolvedModule{
		mod("github.com/x/a", map[string]configuration.Argument{
			"foo": {Schema: map[string]interface{}{"type": "string"}},
			"bar": {Schema: map[string]interface{}{"type": "string"}},
		}),
		mod("github.com/x/b", map[string]configuration.Argument{
			"foo": {From: "github.com/x/a"},
		}, "github.com/x/a"), // lists a
		mod("github.com/x/c", map[string]configuration.Argument{
			"bar": {From: "github.com/x/a"},
		}), // does NOT list a
	}
	_, findings := buildArgIndex(mods)
	// Only c's 'bar' from: should fail (missing dependency listing).
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, "arguments.bar", findings[0].Path)
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

func TestValidateOnlineRunsOfflineFirst(t *testing.T) {
	// An offline finding (invalid name, F2) AND an online finding (O3) both appear,
	// offline first.
	res, _ := Load(strings.NewReader("name: Bad Name\n"))
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Required: true},
	})}
	findings := ValidateOnline(res, mods)
	assert.Assert(t, len(findings) >= 2)
	assert.Equal(t, "name", findings[0].Path) // offline F2 precedes online O3
}

func TestValidateOnlineDeterministicOrder(t *testing.T) {
	// Two modules both require distinct args; output is stable across runs.
	res, _ := Load(strings.NewReader("name: s\n"))
	mods := []ResolvedModule{
		mod("github.com/x/b", map[string]configuration.Argument{"beta": {Required: true}}),
		mod("github.com/x/a", map[string]configuration.Argument{"alpha": {Required: true}}),
	}
	f1 := ValidateOnline(res, mods)
	f2 := ValidateOnline(res, mods)
	assert.DeepEqual(t, f1, f2)
	// sorted by Path: arguments.alpha before arguments.beta
	assert.Equal(t, "arguments.alpha", f1[0].Path)
	assert.Equal(t, "arguments.beta", f1[1].Path)
}

// TestValidateOnlineSnapshot exercises a mix of offline + O3 + O4 findings whose
// messages are all stencil-owned, so the golden is stable. O2 (jsonschema
// library-worded) violations are asserted by severity+path in
// TestValidateOnlineO2SeverityPath, not snapshotted — the manifest package's
// TestValidateExternalErrors is the precedent.
func TestValidateOnlineSnapshot(t *testing.T) {
	// Offline F2 (invalid name) + O3 (required arg missing) + O4 (from: missing dep).
	res, _ := Load(strings.NewReader("name: Bad Name\n"))
	mods := []ResolvedModule{
		mod("github.com/x/a", map[string]configuration.Argument{
			"needed": {Required: true},
		}),
		mod("github.com/x/b", map[string]configuration.Argument{
			"borrowed": {From: "github.com/x/c"}, // no dep listed → O4
		}),
	}
	cupaloy.SnapshotT(t, renderOnlineFindings(ValidateOnline(res, mods)))
}

// TestValidateOnlineO2SeverityPath asserts the O2 (schema-violation) finding by
// severity+path only; its message is jsonschema-library text and must not be
// snapshotted.
func TestValidateOnlineO2SeverityPath(t *testing.T) {
	res, _ := Load(strings.NewReader("name: s\narguments:\n  foo: 123\n"))
	mods := []ResolvedModule{mod("github.com/x/a", map[string]configuration.Argument{
		"foo": {Schema: map[string]interface{}{"type": "string"}},
	})}
	findings := ValidateOnline(res, mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, lint.SeverityError, findings[0].Severity)
	assert.Equal(t, "arguments.foo", findings[0].Path)
}

func TestCheckReplacementsUnmatchedKeyO5(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", nil)}
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Replacements: map[string]string{"github.com/x/nope": "file:///wherever"},
	}}
	findings := checkReplacements(res, mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, lint.SeverityWarning, findings[0].Severity)
	assert.Equal(t, "replacements.github.com/x/nope", findings[0].Path)
}

func TestCheckReplacementsMatchedLocalMissingO8(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", nil)}
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Replacements: map[string]string{"github.com/x/a": "file:///does/not/exist/xyz"},
	}}
	findings := checkReplacements(res, mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, lint.SeverityError, findings[0].Severity) // O8
	assert.Equal(t, "replacements.github.com/x/a", findings[0].Path)
}

func TestCheckReplacementsMatchedLocalExistsNoFinding(t *testing.T) {
	dir := t.TempDir()
	mods := []ResolvedModule{mod("github.com/x/a", nil)}
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Replacements: map[string]string{"github.com/x/a": "file://" + dir},
	}}
	assert.Equal(t, 0, len(checkReplacements(res, mods)))
}

func TestCheckReplacementsMatchedRemoteNotChecked(t *testing.T) {
	mods := []ResolvedModule{mod("github.com/x/a", nil)}
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Replacements: map[string]string{"github.com/x/a": "https://github.com/x/a"},
	}}
	// matched key + remote value: neither O5 (matched) nor O8 (not local) → nothing.
	assert.Equal(t, 0, len(checkReplacements(res, mods)))
}

func TestCheckReplacementsBareLocalPath(t *testing.T) {
	// a bare path (no scheme) is local; missing → O8.
	mods := []ResolvedModule{mod("github.com/x/a", nil)}
	res := &LoadResult{Manifest: &configuration.ServiceManifest{
		Replacements: map[string]string{"github.com/x/a": "./nope-does-not-exist-xyz"},
	}}
	findings := checkReplacements(res, mods)
	assert.Equal(t, 1, len(findings))
	assert.Equal(t, lint.SeverityError, findings[0].Severity)
}
