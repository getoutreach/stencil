// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for resolving dotted lint finding paths to YAML source lines.

package manifest

import (
	"testing"

	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

// parseNode is a test helper: decode YAML text into a *yaml.Node document root.
func parseNode(t *testing.T, in string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	assert.NilError(t, yaml.Unmarshal([]byte(in), &n))
	return &n
}

func TestResolvePath(t *testing.T) {
	// Line-numbered for reference (1-based):
	// 1 name: testing
	// 2 type: templates
	// 3 stencilVersion: ">=1.0.0"
	// 4 arguments:
	// 5   foo:
	// 6     type: string
	// 7     values: [a, b]
	// 8     schema:
	// 9       type: string
	// 10 modules:
	// 11   - name: github.com/getoutreach/stencil-base
	// 12     url: https://x
	// 13     prerelease: true
	const doc = `name: testing
type: templates
stencilVersion: ">=1.0.0"
arguments:
  foo:
    type: string
    values: [a, b]
    schema:
      type: string
modules:
  - name: github.com/getoutreach/stencil-base
    url: https://x
    prerelease: true
`
	root := parseNode(t, doc)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"top-level name", "name", 1},
		{"top-level type", "type", 2},
		{"top-level stencilVersion", "stencilVersion", 3},
		{"argument block key", "arguments.foo", 5},
		{"argument type field", "arguments.foo.type", 6},
		{"argument values field", "arguments.foo.values", 7},
		{"argument schema field", "arguments.foo.schema", 8},
		{"module url by name", "modules.github.com/getoutreach/stencil-base.url", 12},
		{"module prerelease by name", "modules.github.com/getoutreach/stencil-base.prerelease", 13},
		{"module url by index", "modules[0].url", 12},
		{"module prerelease by index", "modules[0].prerelease", 13},
		{"whole-document path misses", "manifest.yaml", 0},
		{"nonexistent argument misses", "arguments.nope.type", 0},
		{"out-of-range module index misses", "modules[9].url", 0},
		{"malformed index brace misses", "modules[0.url", 0},
		{"non-numeric index misses", "modules[x].url", 0},
		{"nonexistent module name misses", "modules.no-such-module.url", 0},
		{"empty path misses", "", 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, resolvePath(root, test.path))
		})
	}
}

func TestResolvePathNilRoot(t *testing.T) {
	assert.Equal(t, 0, resolvePath(nil, "name"))
}

func TestResolvePathFollowsAlias(t *testing.T) {
	// foo is an alias of the &def anchor; resolving through it must land on the
	// anchor definition's key line.
	// 1 defaults: &def
	// 2   type: string
	// 3   values: [a, b]
	// 4 arguments:
	// 5   foo: *def
	const doc = `defaults: &def
  type: string
  values: [a, b]
arguments:
  foo: *def
`
	root := parseNode(t, doc)
	// The aliased mapping's type key is defined on line 2.
	assert.Equal(t, 2, resolvePath(root, "arguments.foo.type"))
}

func TestResolvePathDottedArgNameIsKnownLimitation(t *testing.T) {
	// An argument whose name contains a dot mis-splits; documents the accepted
	// limitation. "weird.name" splits into "weird"/"name", which won't match.
	const doc = `arguments:
  weird.name:
    type: string
`
	root := parseNode(t, doc)
	assert.Equal(t, 0, resolvePath(root, "arguments.weird.name.type"))
}
