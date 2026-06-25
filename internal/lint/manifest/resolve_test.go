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
	// 10   aws.IRSA:
	// 11     type: boolean
	// 12   terraform.datadog.monitoring.generateSLOs:
	// 13     type: boolean
	// 14     default: true
	// 15 modules:
	// 16   - name: github.com/getoutreach/stencil-base
	// 17     url: https://x
	// 18     prerelease: true
	const doc = `name: testing
type: templates
stencilVersion: ">=1.0.0"
arguments:
  foo:
    type: string
    values: [a, b]
    schema:
      type: string
  aws.IRSA:
    type: boolean
  terraform.datadog.monitoring.generateSLOs:
    type: boolean
    default: true
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
		{"dotted argument name, type field", "arguments.aws.IRSA.type", 11},
		{"dotted argument name, bare", "arguments.aws.IRSA", 10},
		{"deeply dotted argument name, type field", "arguments.terraform.datadog.monitoring.generateSLOs.type", 13},
		{"deeply dotted argument name, bare", "arguments.terraform.datadog.monitoring.generateSLOs", 12},
		{"module url by name", "modules.github.com/getoutreach/stencil-base.url", 17},
		{"module prerelease by name", "modules.github.com/getoutreach/stencil-base.prerelease", 18},
		{"module url by index", "modules[0].url", 17},
		{"module prerelease by index", "modules[0].prerelease", 18},
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

func TestResolvePathDottedArgName(t *testing.T) {
	// Argument names commonly contain dots for namespacing (e.g. "aws.IRSA",
	// "dependencies.optional"). The resolver must treat the segment between the
	// "arguments." prefix and a known trailing field as a single flat key.
	// 1 arguments:
	// 2   weird.name:
	// 3     type: string
	const doc = `arguments:
  weird.name:
    type: string
`
	root := parseNode(t, doc)
	// Field form resolves to the field key line; bare form resolves to the
	// argument's own key line.
	assert.Equal(t, 3, resolvePath(root, "arguments.weird.name.type"))
	assert.Equal(t, 2, resolvePath(root, "arguments.weird.name"))
}
