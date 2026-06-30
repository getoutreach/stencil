// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the manifest auto-fixer.

package manifest

import (
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
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

// TestFixBytesNoOpPreservesNonCanonicalFormatting pins Fix 1: when no
// deprecation is fixed, FixBytes returns the original bytes verbatim rather
// than re-encoding them. The input here is clean (nothing to fix) but uses
// 4-space indentation, which the yaml.v3 encoder would rewrite to 2-space if
// the no-op path re-encoded. Byte-identical output proves the no-op short
// circuit; without it this would reformat.
func TestFixBytesNoOpPreservesNonCanonicalFormatting(t *testing.T) {
	in := []byte("name: m\n" +
		"arguments:\n" +
		"    greeting:\n" +
		"        schema:\n" +
		"            type: string\n")
	fixed, applied, ok := FixBytes(in)
	assert.Assert(t, ok)
	assert.Equal(t, 0, len(applied))
	assert.Equal(t, string(in), string(fixed),
		"a no-op fix must not reformat the manifest, got:\n%s", string(fixed))
}

func TestFixPreservesHeadAndFootComments(t *testing.T) {
	in := "name: m\n" +
		"arguments:\n" +
		"  x:\n" +
		"    # head on values\n" +
		"    values: [a, b] # line on values\n" +
		"    # foot on values\n"
	out, _ := fixString(t, in)
	assert.Assert(t, strings.Contains(out, "# head on values"),
		"head comment must survive migration, got:\n%s", out)
	assert.Assert(t, strings.Contains(out, "# line on values"),
		"line comment must survive migration, got:\n%s", out)
	assert.Assert(t, strings.Contains(out, "# foot on values"),
		"foot comment must survive migration, got:\n%s", out)
}

func TestFixPrereleaseCarriesHeadComment(t *testing.T) {
	in := "name: m\n" +
		"modules:\n" +
		"  - name: dep\n" +
		"    # use rc channel\n" +
		"    prerelease: true\n"
	out, _ := fixString(t, in)
	assert.Assert(t, strings.Contains(out, "# use rc channel"),
		"head comment must move to channel, got:\n%s", out)
	assert.Assert(t, strings.Contains(out, "channel: rc"))
}

// fixRelintErrors runs the post-fix strict lint over the fixed output and
// returns the error-severity findings that remain. It mirrors what the CLI does
// after writing a fixed manifest.
func fixRelintErrors(t *testing.T, in string) []lint.Finding {
	t.Helper()
	out, _ := fixString(t, in)
	res, readErr := Load(strings.NewReader(out))
	assert.NilError(t, readErr)
	var errs []lint.Finding
	for _, f := range Validate(res) {
		if f.Severity == lint.SeverityError {
			errs = append(errs, f)
		}
	}
	return errs
}

func TestFixConsolidatesUnknownSiblingIntoSchema(t *testing.T) {
	// The legacy object form: a deprecated `type: object` next to bare schema
	// keywords (`properties`) placed as siblings of the argument. The fixer must
	// move BOTH into `schema`, producing a strict-valid object schema.
	in := "name: m\n" +
		"arguments:\n" +
		"  irsa:\n" +
		"    type: object\n" +
		"    properties:\n" +
		"      msk:\n" +
		"        type: boolean\n" +
		"    description: IRSA access.\n"
	out, applied := fixString(t, in)

	// type and properties both live under schema now; description stays an arg field.
	assert.Assert(t, strings.Contains(out, "schema:"), "got:\n%s", out)
	doc := mappingFrom(t, out)
	arg := doc.Content[findKey(doc, "arguments")+1].Content[1]
	assert.Assert(t, findKey(arg, "type") == -1, "deprecated type must leave the arg, got:\n%s", out)
	assert.Assert(t, findKey(arg, "properties") == -1, "stranded properties must leave the arg, got:\n%s", out)
	assert.Assert(t, findKey(arg, "description") >= 0, "description must remain an arg field, got:\n%s", out)
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Assert(t, findKey(schema, "type") >= 0, "schema.type expected, got:\n%s", out)
	assert.Assert(t, findKey(schema, "properties") >= 0, "schema.properties expected, got:\n%s", out)
	assert.Assert(t, len(applied) >= 1)

	// The whole point: the fixed manifest must strictly decode (no unknown-field error).
	assert.Equal(t, 0, len(fixRelintErrors(t, in)),
		"fixed manifest must have no remaining strict-decode errors")
}

func TestFixConsolidatesArraySibling(t *testing.T) {
	in := "name: m\n" +
		"arguments:\n" +
		"  tags:\n" +
		"    type: array\n" +
		"    items:\n" +
		"      type: string\n"
	out, _ := fixString(t, in)
	doc := mappingFrom(t, out)
	arg := doc.Content[findKey(doc, "arguments")+1].Content[1]
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Assert(t, findKey(schema, "items") >= 0, "items must move into schema, got:\n%s", out)
	assert.Assert(t, findKey(arg, "items") == -1, "items must leave the arg, got:\n%s", out)
	assert.Equal(t, 0, len(fixRelintErrors(t, in)))
}

func TestFixSiblingConsolidationKeepsKnownArgFields(t *testing.T) {
	// description, required, default must NOT be swept into schema.
	in := "name: m\n" +
		"arguments:\n" +
		"  x:\n" +
		"    type: object\n" +
		"    properties:\n" +
		"      a:\n" +
		"        type: string\n" +
		"    required: false\n" +
		"    default: {}\n" +
		"    description: keep me\n"
	out, _ := fixString(t, in)
	doc := mappingFrom(t, out)
	arg := doc.Content[findKey(doc, "arguments")+1].Content[1]
	for _, k := range []string{"required", "default", "description"} {
		assert.Assert(t, findKey(arg, k) >= 0, "%s must remain an arg field, got:\n%s", k, out)
	}
	schema := arg.Content[findKey(arg, "schema")+1]
	for _, k := range []string{"required", "default", "description"} {
		assert.Assert(t, findKey(schema, k) == -1, "%s must NOT be in schema, got:\n%s", k, out)
	}
	assert.Equal(t, 0, len(fixRelintErrors(t, in)))
}

func TestFixDoesNotConsolidateSiblingsWithoutDeprecatedType(t *testing.T) {
	// No deprecated `type` trigger: a stranded sibling is left for a human.
	in := "name: m\n" +
		"arguments:\n" +
		"  x:\n" +
		"    schema:\n" +
		"      type: object\n" +
		"    properties:\n" +
		"      a:\n" +
		"        type: string\n"
	out, applied := fixString(t, in)
	doc := mappingFrom(t, out)
	arg := doc.Content[findKey(doc, "arguments")+1].Content[1]
	assert.Assert(t, findKey(arg, "properties") >= 0,
		"properties must be left untouched when there is no deprecated type, got:\n%s", out)
	assert.Equal(t, 0, len(applied))
}

func TestFixDoesNotConsolidateSiblingsOnRedundantTypePath(t *testing.T) {
	// Deprecated `type` equals an existing `schema.type` (redundant path): the
	// fixer drops the redundant `type` but, by design, does NOT consolidate a
	// stranded `properties` sibling on this path. The orphan is left for a human
	// and the strict re-lint still reports it as an error.
	in := "name: m\n" +
		"arguments:\n" +
		"  x:\n" +
		"    type: object\n" +
		"    schema:\n" +
		"      type: object\n" +
		"    properties:\n" +
		"      a:\n" +
		"        type: string\n"
	out, applied := fixString(t, in)
	doc := mappingFrom(t, out)
	arg := doc.Content[findKey(doc, "arguments")+1].Content[1]

	// The redundant deprecated `type` is removed...
	assert.Assert(t, findKey(arg, "type") == -1, "redundant deprecated type should be removed, got:\n%s", out)
	// ...but `properties` is intentionally left orphaned (not swept into schema).
	assert.Assert(t, findKey(arg, "properties") >= 0,
		"properties must stay orphaned on the redundant path, got:\n%s", out)
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Assert(t, findKey(schema, "properties") == -1,
		"properties must NOT be moved into schema on the redundant path, got:\n%s", out)
	// Exactly one change: the redundant-type removal. No sibling migration entries.
	assert.Equal(t, 1, len(applied))
	assert.Equal(t, "arguments.x.type", applied[0].Path)

	// The orphaned sibling is still an unfixable error after fixing.
	assert.Assert(t, len(fixRelintErrors(t, in)) >= 1,
		"orphaned properties must remain a strict-decode error")
}

func TestFixDoesNotConsolidateSiblingsOnDiffersTypePath(t *testing.T) {
	// Deprecated `type` disagrees with an existing `schema.type` (differs path):
	// the fixer makes NO change at all — neither the type nor the orphaned
	// `properties` is touched — because the conflict is ambiguous.
	in := "name: m\n" +
		"arguments:\n" +
		"  x:\n" +
		"    type: object\n" +
		"    schema:\n" +
		"      type: string\n" +
		"    properties:\n" +
		"      a:\n" +
		"        type: string\n"
	out, applied := fixString(t, in)
	doc := mappingFrom(t, out)
	arg := doc.Content[findKey(doc, "arguments")+1].Content[1]

	// Nothing moved: deprecated type, schema.type, and properties all stay put.
	assert.Assert(t, findKey(arg, "type") >= 0, "deprecated type must stay on the differs path, got:\n%s", out)
	assert.Assert(t, findKey(arg, "properties") >= 0, "properties must stay orphaned on the differs path, got:\n%s", out)
	schema := arg.Content[findKey(arg, "schema")+1]
	assert.Equal(t, "string", schema.Content[findKey(schema, "type")+1].Value)
	assert.Assert(t, findKey(schema, "properties") == -1)
	assert.Equal(t, 0, len(applied))
}

// TestKnownArgFieldsMatchesArgument asserts knownArgFields lists exactly the
// yaml tags on configuration.Argument, so a new field cannot be silently swept
// into schema by consolidateSchemaSiblings.
func TestKnownArgFieldsMatchesArgument(t *testing.T) {
	want := map[string]bool{}
	rt := reflect.TypeOf(configuration.Argument{})
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		want[name] = true
	}
	assert.DeepEqual(t, want, knownArgFields)
}

// TestFixArgumentsOrderAndFromSkip pins the fixer's iteration contract to the
// checker's: arguments are visited in sorted key order, and any argument with a
// from: reference is skipped (its other fields are ignored at render time).
func TestFixArgumentsOrderAndFromSkip(t *testing.T) {
	// b and a both have a fixable `type`; c is a from: ref with a `type` that
	// must be left untouched. Applied paths reflect visitation order.
	in := "name: m\narguments:\n" +
		"  b:\n    type: string\n" +
		"  a:\n    type: string\n" +
		"  c:\n    from: other\n    type: string\n"
	root := mappingFrom(t, in)
	var applied []Applied
	fixArguments(root, &applied)

	var paths []string
	for _, a := range applied {
		paths = append(paths, a.Path)
	}
	assert.DeepEqual(t, []string{
		"arguments.a.type",
		"arguments.b.type",
	}, paths)
}

// TestFixDerefsAliasedModule pins Fix 9: a module list entry that is a YAML
// alias to an anchored mapping is dereferenced before the mapping-kind guard,
// so its deprecated `prerelease` is migrated. Without the deref, fixModules
// skips the alias node (Kind == AliasNode, not MappingNode) and the deprecation
// is left in place.
func TestFixDerefsAliasedModule(t *testing.T) {
	in := "name: m\n" +
		"definitions:\n" +
		"  dep: &dep\n" +
		"    name: github.com/getoutreach/dep\n" +
		"    prerelease: true\n" +
		"modules:\n" +
		"  - *dep\n"
	out, applied := fixString(t, in)
	assert.Assert(t, len(applied) >= 1, "aliased module should be fixed, got %d applied", len(applied))
	assert.Assert(t, strings.Contains(out, "channel: rc"),
		"aliased module's prerelease should migrate to channel: rc, got:\n%s", out)
	assert.Assert(t, !strings.Contains(out, "prerelease:"),
		"prerelease should be removed from the aliased module, got:\n%s", out)
}
