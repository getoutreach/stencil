// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements safe, comment-preserving auto-fixes for a Stencil
// template repository manifest (manifest.yaml). The fixer edits the yaml.Node
// DOM in place so comments and key order survive, and only applies migrations
// that are unambiguous.

package manifest

import (
	"bytes"
	"sort"

	"github.com/getoutreach/stencil/internal/lint/modulefix"
	"github.com/getoutreach/stencil/internal/lint/yamlfix"
	"go.yaml.in/yaml/v3"
)

// Applied records one fix the fixer made, for logging. Aliased to the shared
// modulefix.Applied so both linters and the command layer log fixes uniformly.
type Applied = modulefix.Applied

// scalarsEqual reports whether a and b are both scalar nodes with equal values.
func scalarsEqual(a, b *yaml.Node) bool {
	return a != nil && b != nil &&
		a.Kind == yaml.ScalarNode && b.Kind == yaml.ScalarNode &&
		a.Value == b.Value
}

// knownArgFields are the recognized fields of configuration.Argument. Any other
// key on an argument mapping is unknown to the struct: the lenient render-time
// decoder drops it, and the strict lint decoder rejects it. The legacy `type:`
// form placed JSON-Schema keywords (properties, items, …) here as siblings, so
// these are the keys the fixer consolidates into `schema` when migrating `type`.
var knownArgFields = map[string]bool{
	"description": true,
	"required":    true,
	"default":     true,
	"schema":      true,
	"deprecated":  true,
	"type":        true,
	"values":      true,
	"from":        true,
}

// consolidateSchemaSiblings moves every argument sibling key that is not a
// recognized Argument field into schema, reusing each node verbatim (preserving
// comments and style). An existing schema key of the same name is never
// overwritten (setIfAbsent); a colliding sibling is left in place for the
// linter. Names are collected before mutating, since removal shifts the slice.
func consolidateSchemaSiblings(arg, schema *yaml.Node, argName string, applied *[]Applied) {
	var siblings []string
	for j := 0; j+1 < len(arg.Content); j += 2 {
		key := arg.Content[j].Value
		if !knownArgFields[key] {
			siblings = append(siblings, key)
		}
	}
	for _, key := range siblings {
		srcKey := arg.Content[yamlfix.FindKey(arg, key)]
		val := yamlfix.RemoveKey(arg, key)
		if yamlfix.SetIfAbsent(schema, key, val) {
			yamlfix.CarryKeyComments(schema, key, srcKey)
			*applied = append(*applied, Applied{
				Path:    "arguments." + argName + "." + key,
				Message: "migrated schema keyword '" + key + "' into 'schema'",
			})
		}
	}
}

// fixArgType migrates a deprecated argument `type: X` into `schema.type: X`.
// It is conservative: it only acts when type is a scalar, never overwrites an
// existing differing schema.type, and reuses the original value node (keeping
// its comment and style).
func fixArgType(argName string, arg *yaml.Node, applied *[]Applied) {
	i := yamlfix.FindKey(arg, "type")
	if i < 0 {
		return
	}
	typeVal := arg.Content[i+1]
	if typeVal.Kind != yaml.ScalarNode {
		return // non-scalar: leave for the linter
	}
	schema := yamlfix.EnsureMapping(arg, "schema")
	if si := yamlfix.FindKey(schema, "type"); si >= 0 {
		existing := schema.Content[si+1]
		if !scalarsEqual(existing, typeVal) {
			return // ambiguous: differing schema.type wins, change nothing
		}
		// Redundant: schema.type already equals the deprecated type.
		yamlfix.RemoveKey(arg, "type")
		*applied = append(*applied, Applied{
			Path:    "arguments." + argName + ".type",
			Message: "removed redundant 'type' (already set in schema.type)",
		})
		return
	}
	srcKey := arg.Content[i] // capture before removeKey (i still valid here)
	yamlfix.RemoveKey(arg, "type")
	yamlfix.SetIfAbsent(schema, "type", typeVal)
	yamlfix.CarryKeyComments(schema, "type", srcKey)
	*applied = append(*applied, Applied{
		Path:    "arguments." + argName + ".type",
		Message: "migrated 'type' into 'schema.type'",
	})
	// Clean migration path only: relocate any stranded schema-keyword siblings
	// (e.g. the legacy `type: object` + bare `properties`) into the same schema.
	consolidateSchemaSiblings(arg, schema, argName, applied)
}

// fixArgValues migrates a deprecated argument `values: [...]` into
// `schema.enum: [...]`. Same conservative rules as fixArgType; the sequence
// node is reused verbatim so its flow/block style is preserved.
func fixArgValues(argName string, arg *yaml.Node, applied *[]Applied) {
	i := yamlfix.FindKey(arg, "values")
	if i < 0 {
		return
	}
	valuesVal := arg.Content[i+1]
	if valuesVal.Kind != yaml.SequenceNode {
		return
	}
	schema := yamlfix.EnsureMapping(arg, "schema")
	if yamlfix.FindKey(schema, "enum") >= 0 {
		// schema.enum already present: drop the deprecated values, keep schema.
		yamlfix.RemoveKey(arg, "values")
		*applied = append(*applied, Applied{
			Path:    "arguments." + argName + ".values",
			Message: "removed redundant 'values' (schema.enum already set)",
		})
		return
	}
	srcKey := arg.Content[i] // capture before removeKey
	yamlfix.RemoveKey(arg, "values")
	yamlfix.SetIfAbsent(schema, "enum", valuesVal)
	yamlfix.CarryKeyComments(schema, "enum", srcKey)
	*applied = append(*applied, Applied{
		Path:    "arguments." + argName + ".values",
		Message: "migrated 'values' into 'schema.enum'",
	})
}

// Fix applies the safe deprecation migrations to the manifest document node in
// place and returns the list of changes made. doc is the *yaml.Node from
// yaml.Unmarshal (a DocumentNode wrapping a MappingNode). It never returns an
// error: anything it cannot safely fix is left untouched. Arguments are
// processed in sorted key order and modules in slice order, matching the
// checker, so the result is deterministic.
func Fix(doc *yaml.Node) []Applied {
	if doc == nil || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}

	var applied []Applied
	fixArguments(root, &applied)
	applied = append(applied, modulefix.FixModules(root)...)
	return applied
}

// fixArguments applies argument migrations in sorted key order, skipping
// arguments that set from: (whose other fields are ignored at render time).
func fixArguments(root *yaml.Node, applied *[]Applied) {
	ai := yamlfix.FindKey(root, "arguments")
	if ai < 0 {
		return
	}
	args := root.Content[ai+1]
	if args.Kind != yaml.MappingNode {
		return
	}

	// Collect argument names in sorted order (keys are at even indices).
	names := make([]string, 0, len(args.Content)/2)
	for i := 0; i+1 < len(args.Content); i += 2 {
		names = append(names, args.Content[i].Value)
	}
	sort.Strings(names)

	for _, name := range names {
		arg := yamlfix.Deref(args.Content[yamlfix.FindKey(args, name)+1])
		if arg.Kind != yaml.MappingNode {
			continue
		}
		if yamlfix.FindKey(arg, "from") >= 0 {
			continue // from: arguments are skipped, matching checkArguments
		}
		fixArgType(name, arg, applied)
		fixArgValues(name, arg, applied)
	}
}

// FixBytes decodes raw as a YAML node, applies the safe deprecation migrations,
// and re-encodes the result with the canonical 2-space indent. ok is false only
// when raw cannot be decoded as a YAML node, in which case the caller should
// skip fixing and run the normal lint (which reports the decode error). When ok
// is true and applied is empty, the document had nothing to fix.
func FixBytes(raw []byte) (fixed []byte, applied []Applied, ok bool) {
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, false
	}
	if len(doc.Content) == 0 {
		// Empty document: nothing to fix, but not an error.
		return raw, nil, true
	}
	applied = Fix(&doc)
	if len(applied) == 0 {
		// Nothing changed: return the original bytes verbatim so a no-op
		// --fix never reformats the file (yaml.v3 does not preserve the
		// original textual representation).
		return raw, nil, true
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, nil, false
	}
	if err := enc.Close(); err != nil {
		return nil, nil, false
	}
	return buf.Bytes(), applied, true
}
