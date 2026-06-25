// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the manifest auto-fixer.

package manifest

import (
	"testing"

	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

// mappingFrom decodes a YAML mapping document and returns its root mapping node.
func mappingFrom(t *testing.T, in string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	assert.NilError(t, yaml.Unmarshal([]byte(in), &doc))
	assert.Assert(t, len(doc.Content) == 1)
	return doc.Content[0]
}

func TestFindKey(t *testing.T) {
	m := mappingFrom(t, "a: 1\nb: 2\n")
	assert.Equal(t, 0, findKey(m, "a"))
	assert.Equal(t, 2, findKey(m, "b"))
	assert.Equal(t, -1, findKey(m, "missing"))
}

func TestRemoveKeyInPlace(t *testing.T) {
	m := mappingFrom(t, "a: 1\nb: 2\nc: 3\n")
	val := removeKey(m, "b")
	assert.Assert(t, val != nil)
	assert.Equal(t, "2", val.Value)
	// a and c remain, in order; b is gone.
	assert.Equal(t, 0, findKey(m, "a"))
	assert.Equal(t, -1, findKey(m, "b"))
	assert.Equal(t, 2, findKey(m, "c"))
}

func TestRemoveKeyMissing(t *testing.T) {
	m := mappingFrom(t, "a: 1\n")
	assert.Assert(t, removeKey(m, "nope") == nil)
	assert.Equal(t, 0, findKey(m, "a"))
}

func TestEnsureMappingExisting(t *testing.T) {
	m := mappingFrom(t, "schema:\n  type: string\n")
	s := ensureMapping(m, "schema")
	assert.Equal(t, yaml.MappingNode, s.Kind)
	assert.Equal(t, 0, findKey(s, "type"))
}

func TestEnsureMappingCreates(t *testing.T) {
	m := mappingFrom(t, "name: x\n")
	s := ensureMapping(m, "schema")
	assert.Equal(t, yaml.MappingNode, s.Kind)
	// schema appended last.
	assert.Equal(t, 2, findKey(m, "schema"))
}

func TestSetIfAbsent(t *testing.T) {
	m := mappingFrom(t, "a: 1\n")
	v := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rc"}
	assert.Assert(t, setIfAbsent(m, "channel", v))
	assert.Equal(t, 2, findKey(m, "channel"))
	// Second call is a no-op.
	assert.Assert(t, !setIfAbsent(m, "channel", v))
}
