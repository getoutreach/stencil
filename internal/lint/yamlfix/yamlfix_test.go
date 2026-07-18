// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the shared yaml.Node primitives.

package yamlfix_test

import (
	"testing"

	"go.yaml.in/yaml/v3"
	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/internal/lint/yamlfix"
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
	assert.Equal(t, 0, yamlfix.FindKey(m, "a"))
	assert.Equal(t, 2, yamlfix.FindKey(m, "b"))
	assert.Equal(t, -1, yamlfix.FindKey(m, "missing"))
}

func TestRemoveKeyInPlace(t *testing.T) {
	m := mappingFrom(t, "a: 1\nb: 2\nc: 3\n")
	val := yamlfix.RemoveKey(m, "b")
	assert.Assert(t, val != nil)
	assert.Equal(t, "2", val.Value)
	assert.Equal(t, -1, yamlfix.FindKey(m, "b"))
	assert.Equal(t, 0, yamlfix.FindKey(m, "a"))
	assert.Equal(t, 2, yamlfix.FindKey(m, "c"))
}

func TestRemoveKeyMissing(t *testing.T) {
	m := mappingFrom(t, "a: 1\n")
	assert.Assert(t, yamlfix.RemoveKey(m, "nope") == nil)
}

func TestEnsureMappingExisting(t *testing.T) {
	m := mappingFrom(t, "schema:\n  type: string\n")
	got := yamlfix.EnsureMapping(m, "schema")
	assert.Assert(t, got.Kind == yaml.MappingNode)
	assert.Assert(t, yamlfix.FindKey(got, "type") >= 0)
}

func TestEnsureMappingCreates(t *testing.T) {
	m := mappingFrom(t, "name: x\n")
	got := yamlfix.EnsureMapping(m, "schema")
	assert.Assert(t, got.Kind == yaml.MappingNode)
	assert.Equal(t, 0, len(got.Content))
	assert.Assert(t, yamlfix.FindKey(m, "schema") >= 0)
}

func TestSetIfAbsent(t *testing.T) {
	m := mappingFrom(t, "a: 1\n")
	added := yamlfix.SetIfAbsent(m, "b",
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "2"})
	assert.Equal(t, true, added)
	assert.Assert(t, yamlfix.FindKey(m, "b") >= 0)

	again := yamlfix.SetIfAbsent(m, "b",
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "3"})
	assert.Equal(t, false, again)
}

func TestDerefFollowsAlias(t *testing.T) {
	// An anchored mapping referenced by an alias resolves to its target.
	doc := mappingFrom(t, "a: &anchor {k: v}\nb: *anchor\n")
	// b's value is an alias node; deref returns the anchored mapping.
	bIdx := yamlfix.FindKey(doc, "b")
	alias := doc.Content[bIdx+1]
	assert.Equal(t, yaml.AliasNode, alias.Kind)
	target := yamlfix.Deref(alias)
	assert.Equal(t, yaml.MappingNode, target.Kind)
}

func TestDerefNonAliasUnchanged(t *testing.T) {
	n := &yaml.Node{Kind: yaml.ScalarNode, Value: "x"}
	assert.Equal(t, n, yamlfix.Deref(n))
	assert.Assert(t, yamlfix.Deref(nil) == nil)
}

func TestCarryKeyComments(t *testing.T) {
	m := mappingFrom(t, "dst: 1\n")
	src := &yaml.Node{Kind: yaml.ScalarNode, Value: "src",
		HeadComment: "# head", FootComment: "# foot"}
	yamlfix.CarryKeyComments(m, "dst", src)
	i := yamlfix.FindKey(m, "dst")
	assert.Equal(t, "# head", m.Content[i].HeadComment)
	assert.Equal(t, "# foot", m.Content[i].FootComment)
}
