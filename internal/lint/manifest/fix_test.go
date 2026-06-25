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

// argNode returns the value mapping for the single argument in
// "arguments:\n  <name>:\n    ...". Helper for the per-argument fix tests.
func argNode(t *testing.T, name, body string) *yaml.Node {
	t.Helper()
	m := mappingFrom(t, "arguments:\n  "+name+":\n"+body)
	args := m.Content[findKey(m, "arguments")+1]
	return args.Content[1] // value of the first (only) argument
}

func TestFixArgTypeMovesIntoSchema(t *testing.T) {
	arg := argNode(t, "x", "    type: string\n")
	var applied []Applied
	fixArgType("x", arg, &applied)

	assert.Assert(t, findKey(arg, "type") == -1) // removed
	schema := arg.Content[findKey(arg, "schema")+1]
	ti := findKey(schema, "type")
	assert.Assert(t, ti >= 0)
	assert.Equal(t, "string", schema.Content[ti+1].Value)
	assert.Equal(t, 1, len(applied))
	assert.Equal(t, "arguments.x.type", applied[0].Path)
}

func TestFixArgTypeRedundantWhenSchemaTypeEqual(t *testing.T) {
	arg := argNode(t, "x", "    type: string\n    schema:\n      type: string\n")
	var applied []Applied
	fixArgType("x", arg, &applied)

	// Deprecated type dropped; existing schema.type kept.
	assert.Assert(t, findKey(arg, "type") == -1)
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Equal(t, "string", schema.Content[findKey(schema, "type")+1].Value)
	assert.Equal(t, 1, len(applied)) // still logged as a change (removed redundant)
}

func TestFixArgTypeNoChangeWhenSchemaTypeDiffers(t *testing.T) {
	arg := argNode(t, "x", "    type: string\n    schema:\n      type: integer\n")
	var applied []Applied
	fixArgType("x", arg, &applied)

	// Ambiguous: leave both, change nothing.
	assert.Assert(t, findKey(arg, "type") >= 0)
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Equal(t, "integer", schema.Content[findKey(schema, "type")+1].Value)
	assert.Equal(t, 0, len(applied))
}

func TestFixArgValuesMovesIntoSchemaEnum(t *testing.T) {
	arg := argNode(t, "x", "    values: [a, b]\n")
	var applied []Applied
	fixArgValues("x", arg, &applied)

	assert.Assert(t, findKey(arg, "values") == -1)
	schema := arg.Content[findKey(arg, "schema")+1]
	enum := schema.Content[findKey(schema, "enum")+1]
	assert.Equal(t, yaml.SequenceNode, enum.Kind)
	assert.Equal(t, 2, len(enum.Content))
	assert.Equal(t, 1, len(applied))
}

func TestFixModulePrereleaseTrueAddsChannel(t *testing.T) {
	mod := mappingFrom(t, "name: m\nprerelease: true\n")
	var applied []Applied
	fixModulePrerelease("modules.m", mod, &applied)

	assert.Assert(t, findKey(mod, "prerelease") == -1)
	assert.Equal(t, "rc", mod.Content[findKey(mod, "channel")+1].Value)
	assert.Equal(t, 1, len(applied))
}

func TestFixModulePrereleaseKeepsExistingChannel(t *testing.T) {
	mod := mappingFrom(t, "name: m\nchannel: stable\nprerelease: true\n")
	var applied []Applied
	fixModulePrerelease("modules.m", mod, &applied)

	assert.Assert(t, findKey(mod, "prerelease") == -1)
	assert.Equal(t, "stable", mod.Content[findKey(mod, "channel")+1].Value) // not overwritten
	assert.Equal(t, 1, len(applied))
}

func TestFixModulePrereleaseFalseDropped(t *testing.T) {
	mod := mappingFrom(t, "name: m\nprerelease: false\n")
	var applied []Applied
	fixModulePrerelease("modules.m", mod, &applied)

	assert.Assert(t, findKey(mod, "prerelease") == -1)
	assert.Assert(t, findKey(mod, "channel") == -1) // false is just removed
	assert.Equal(t, 1, len(applied))
}
