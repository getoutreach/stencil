// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the manifest auto-fixer.

package manifest

import (
	"strings"
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

func encode(t *testing.T, doc *yaml.Node) string {
	t.Helper()
	var sb strings.Builder
	enc := yaml.NewEncoder(&sb)
	enc.SetIndent(2)
	assert.NilError(t, enc.Encode(doc))
	assert.NilError(t, enc.Close())
	return sb.String()
}

// fixString decodes in, runs Fix, and returns the re-encoded YAML plus applied.
func fixString(t *testing.T, in string) (string, []Applied) {
	t.Helper()
	var doc yaml.Node
	assert.NilError(t, yaml.Unmarshal([]byte(in), &doc))
	applied := Fix(&doc)
	return encode(t, &doc), applied
}

func TestFix(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantOut   string
		wantCount int
	}{
		{
			name:      "arg type only",
			in:        "name: m\narguments:\n  x:\n    type: string\n",
			wantOut:   "name: m\narguments:\n  x:\n    schema:\n      type: string\n",
			wantCount: 1,
		},
		{
			name:      "arg values only",
			in:        "name: m\narguments:\n  x:\n    values: [a, b]\n",
			wantOut:   "name: m\narguments:\n  x:\n    schema:\n      enum: [a, b]\n",
			wantCount: 1,
		},
		{
			name: "arg type and values together",
			in:   "name: m\narguments:\n  x:\n    type: string\n    values: [a, b]\n",
			wantOut: "name: m\narguments:\n  x:\n    schema:\n      type: string\n" +
				"      enum: [a, b]\n",
			wantCount: 2,
		},
		{
			name:      "module prerelease true",
			in:        "name: m\nmodules:\n  - name: dep\n    prerelease: true\n",
			wantOut:   "name: m\nmodules:\n  - name: dep\n    channel: rc\n",
			wantCount: 1,
		},
		{
			name:      "module prerelease false removed",
			in:        "name: m\nmodules:\n  - name: dep\n    prerelease: false\n",
			wantOut:   "name: m\nmodules:\n  - name: dep\n",
			wantCount: 1,
		},
		{
			name:      "from arg untouched",
			in:        "name: m\narguments:\n  x:\n    from: dep\n    type: string\n",
			wantOut:   "name: m\narguments:\n  x:\n    from: dep\n    type: string\n",
			wantCount: 0,
		},
		{
			name:      "nothing to fix is byte identical",
			in:        "name: m\narguments:\n  x:\n    schema:\n      type: string\n",
			wantOut:   "name: m\narguments:\n  x:\n    schema:\n      type: string\n",
			wantCount: 0,
		},
		{
			name:      "schema.type differs leaves both",
			in:        "name: m\narguments:\n  x:\n    type: string\n    schema:\n      type: integer\n",
			wantOut:   "name: m\narguments:\n  x:\n    type: string\n    schema:\n      type: integer\n",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, applied := fixString(t, tt.in)
			assert.Equal(t, tt.wantOut, out)
			assert.Equal(t, tt.wantCount, len(applied))
		})
	}
}

func TestFixPreservesComment(t *testing.T) {
	in := "name: m\narguments:\n  x:\n    type: string # keep me\n"
	out, _ := fixString(t, in)
	assert.Assert(t, strings.Contains(out, "# keep me"),
		"comment must survive the move, got:\n%s", out)
}

func TestFixPreservesKeyOrder(t *testing.T) {
	// Deliberately non-alphabetical key order at two levels.
	in := "type: templates\n" +
		"arguments:\n  a:\n    required: true\n    description: hi\n    type: string\n" +
		"name: m\n"
	out, _ := fixString(t, in)

	// Top-level order: type, arguments, name (unchanged; encoder does not sort).
	tIdx := strings.Index(out, "\ntype:")
	if strings.HasPrefix(out, "type:") {
		tIdx = 0
	}
	argIdx := strings.Index(out, "arguments:")
	nameIdx := strings.Index(out, "\nname:")
	assert.Assert(t, tIdx < argIdx && argIdx < nameIdx,
		"top-level key order must be preserved, got:\n%s", out)

	// Within argument a: required, description precede the migrated schema.
	reqIdx := strings.Index(out, "required:")
	descIdx := strings.Index(out, "description:")
	schemaIdx := strings.Index(out, "schema:")
	assert.Assert(t, reqIdx < descIdx && descIdx < schemaIdx,
		"argument key order must be preserved, got:\n%s", out)
}

// TestFixArgValuesRedundantEnumDropped verifies that when both the deprecated
// values: and schema.enum are present, fixArgValues drops values and keeps enum.
func TestFixArgValuesRedundantEnumDropped(t *testing.T) {
	arg := argNode(t, "x", "    values: [a]\n    schema:\n      enum: [a]\n")
	var applied []Applied
	fixArgValues("x", arg, &applied)

	assert.Assert(t, findKey(arg, "values") == -1) // deprecated values removed
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Assert(t, findKey(schema, "enum") >= 0) // schema.enum kept
	assert.Equal(t, 1, len(applied))
}

// TestFixConservativeSkips verifies that the conservative-skip paths produce no
// change and no Applied entries when the deprecated field is not a fixable shape.
func TestFixConservativeSkips(t *testing.T) {
	t.Run("non-scalar type left alone", func(t *testing.T) {
		arg := argNode(t, "x", "    type:\n      nested: true\n")
		var applied []Applied
		fixArgType("x", arg, &applied)

		assert.Assert(t, findKey(arg, "type") >= 0) // left in place
		assert.Equal(t, 0, len(applied))
	})

	t.Run("sequence prerelease left alone", func(t *testing.T) {
		mod := mappingFrom(t, "name: m\nprerelease: [x]\n")
		var applied []Applied
		fixModulePrerelease("modules.m", mod, &applied)

		assert.Assert(t, findKey(mod, "prerelease") >= 0) // left in place
		assert.Equal(t, 0, len(applied))
	})

	t.Run("non-bool scalar prerelease left alone", func(t *testing.T) {
		mod := mappingFrom(t, "name: m\nprerelease: maybe\n")
		var applied []Applied
		fixModulePrerelease("modules.m", mod, &applied)

		assert.Assert(t, findKey(mod, "prerelease") >= 0) // left in place
		assert.Equal(t, 0, len(applied))
	})
}

func TestFixBytes(t *testing.T) {
	in := []byte("name: m\narguments:\n  x:\n    type: string\n")
	fixed, applied, ok := FixBytes(in)
	assert.Assert(t, ok)
	assert.Equal(t, 1, len(applied))
	assert.Assert(t, strings.Contains(string(fixed), "schema:"))
	assert.Assert(t, strings.Contains(string(fixed), "type: string"))
}

func TestFixBytesMalformed(t *testing.T) {
	// A tab-indented mapping is invalid YAML; FixBytes reports ok=false.
	_, _, ok := FixBytes([]byte("name: m\n\tbad: true\n"))
	assert.Assert(t, !ok)
}

func TestFixBytesNoChange(t *testing.T) {
	in := []byte("name: m\n")
	fixed, applied, ok := FixBytes(in)
	assert.Assert(t, ok)
	assert.Equal(t, 0, len(applied))
	assert.Equal(t, "name: m\n", string(fixed))
}
