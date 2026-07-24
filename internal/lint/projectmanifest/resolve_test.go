// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests resolving dotted project-manifest lint finding paths to
// 1-based YAML source lines, covering the opaque leaf-key branches
// (arguments/versions/replacements) and the module bracket-with-field branch.

package projectmanifest

import (
	"testing"

	"go.yaml.in/yaml/v3"
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
	// 1 name: my-service
	// 2 versions:
	// 3   golang: "1.22"
	// 4 arguments:
	// 5   foo: bar
	// 6 replacements:
	// 7   github.com/x/y: file://../y
	// 8 modules:
	// 9   - name: github.com/getoutreach/stencil-base
	// 10    version: v1.2.3
	const doc = `name: my-service
versions:
  golang: "1.22"
arguments:
  foo: bar
replacements:
  github.com/x/y: file://../y
modules:
  - name: github.com/getoutreach/stencil-base
    version: v1.2.3
`
	root := parseNode(t, doc)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"general dotted walk (name)", "name", 1},
		{"versions opaque leaf", "versions.golang", 3},
		{"arguments opaque leaf", "arguments.foo", 5},
		// The replacement key itself contains dots and slashes; it is matched
		// whole as a single opaque leaf key (not prefix-split).
		{"replacements opaque leaf with dots/slashes", "replacements.github.com/x/y", 7},
		{"module version by name (last-dot split)", "modules.github.com/getoutreach/stencil-base.version", 10},
		{"module name by bracket index", "modules[0].name", 9},
		{"unresolvable path misses", "nope.does.not.exist", 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, resolvePath(root, test.path))
		})
	}
}
