// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements safe, comment-preserving auto-fixes for a Stencil
// template repository manifest (manifest.yaml). The fixer edits the yaml.Node
// DOM in place so comments and key order survive, and only applies migrations
// that are unambiguous.

package manifest

import (
	"sort"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Applied records one fix the fixer made, for logging. Path mirrors the
// corresponding lint.Finding.Path (e.g. "arguments.x.type").
type Applied struct {
	Path    string
	Message string
}

// findKey returns the index in m.Content of the key node named key, or -1.
// m must be a MappingNode (Content is a flat [key, value, key, value, ...]).
func findKey(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// removeKey deletes the key/value pair named key from mapping m in place and
// returns the removed value node, or nil if key was absent. Surviving pairs
// keep their relative order.
func removeKey(m *yaml.Node, key string) *yaml.Node {
	i := findKey(m, key)
	if i < 0 {
		return nil
	}
	val := m.Content[i+1]
	m.Content = append(m.Content[:i], m.Content[i+2:]...)
	return val
}

// ensureMapping returns the existing child mapping stored under key in m, or
// appends a new empty MappingNode under key and returns it.
func ensureMapping(m *yaml.Node, key string) *yaml.Node {
	if i := findKey(m, key); i >= 0 {
		return m.Content[i+1]
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	v := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content, k, v)
	return v
}

// setIfAbsent appends key: val to mapping m only when key is absent. It returns
// true when it added the pair. The val node is used as-is (its Style and
// comments are preserved).
func setIfAbsent(m *yaml.Node, key string, val *yaml.Node) bool {
	if findKey(m, key) >= 0 {
		return false
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	m.Content = append(m.Content, k, val)
	return true
}

// scalarsEqual reports whether a and b are both scalar nodes with equal values.
func scalarsEqual(a, b *yaml.Node) bool {
	return a != nil && b != nil &&
		a.Kind == yaml.ScalarNode && b.Kind == yaml.ScalarNode &&
		a.Value == b.Value
}

// fixArgType migrates a deprecated argument `type: X` into `schema.type: X`.
// It is conservative: it only acts when type is a scalar, never overwrites an
// existing differing schema.type, and reuses the original value node (keeping
// its comment and style).
func fixArgType(argName string, arg *yaml.Node, applied *[]Applied) {
	i := findKey(arg, "type")
	if i < 0 {
		return
	}
	typeVal := arg.Content[i+1]
	if typeVal.Kind != yaml.ScalarNode {
		return // non-scalar: leave for the linter
	}
	schema := ensureMapping(arg, "schema")
	if si := findKey(schema, "type"); si >= 0 {
		existing := schema.Content[si+1]
		if !scalarsEqual(existing, typeVal) {
			return // ambiguous: differing schema.type wins, change nothing
		}
		// Redundant: schema.type already equals the deprecated type.
		removeKey(arg, "type")
		*applied = append(*applied, Applied{
			Path:    "arguments." + argName + ".type",
			Message: "removed redundant 'type' (already set in schema.type)",
		})
		return
	}
	removeKey(arg, "type")
	setIfAbsent(schema, "type", typeVal)
	*applied = append(*applied, Applied{
		Path:    "arguments." + argName + ".type",
		Message: "migrated 'type' into 'schema.type'",
	})
}

// fixArgValues migrates a deprecated argument `values: [...]` into
// `schema.enum: [...]`. Same conservative rules as fixArgType; the sequence
// node is reused verbatim so its flow/block style is preserved.
func fixArgValues(argName string, arg *yaml.Node, applied *[]Applied) {
	i := findKey(arg, "values")
	if i < 0 {
		return
	}
	valuesVal := arg.Content[i+1]
	if valuesVal.Kind != yaml.SequenceNode {
		return
	}
	schema := ensureMapping(arg, "schema")
	if findKey(schema, "enum") >= 0 {
		// schema.enum already present: drop the deprecated values, keep schema.
		removeKey(arg, "values")
		*applied = append(*applied, Applied{
			Path:    "arguments." + argName + ".values",
			Message: "removed redundant 'values' (schema.enum already set)",
		})
		return
	}
	removeKey(arg, "values")
	setIfAbsent(schema, "enum", valuesVal)
	*applied = append(*applied, Applied{
		Path:    "arguments." + argName + ".values",
		Message: "migrated 'values' into 'schema.enum'",
	})
}

// fixModulePrerelease migrates a deprecated module `prerelease` field. A true
// value becomes `channel: rc` (unless channel is already set, which is left
// untouched); a false value is simply removed as a redundant default.
func fixModulePrerelease(modPath string, mod *yaml.Node, applied *[]Applied) {
	i := findKey(mod, "prerelease")
	if i < 0 {
		return
	}
	val := mod.Content[i+1]
	if val.Kind != yaml.ScalarNode {
		return
	}
	switch val.Value {
	case "true":
		removeKey(mod, "prerelease")
		if setIfAbsent(mod, "channel",
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "rc"}) {
			*applied = append(*applied, Applied{
				Path:    modPath + ".prerelease",
				Message: "migrated 'prerelease: true' to 'channel: rc'",
			})
		} else {
			*applied = append(*applied, Applied{
				Path:    modPath + ".prerelease",
				Message: "removed deprecated 'prerelease' (channel already set)",
			})
		}
	case "false":
		removeKey(mod, "prerelease")
		*applied = append(*applied, Applied{
			Path:    modPath + ".prerelease",
			Message: "removed redundant 'prerelease: false'",
		})
	}
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
	fixModules(root, &applied)
	return applied
}

// fixArguments applies argument migrations in sorted key order, skipping
// arguments that set from: (whose other fields are ignored at render time).
func fixArguments(root *yaml.Node, applied *[]Applied) {
	ai := findKey(root, "arguments")
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
		arg := args.Content[findKey(args, name)+1]
		if arg.Kind != yaml.MappingNode {
			continue
		}
		if findKey(arg, "from") >= 0 {
			continue // from: arguments are skipped, matching checkArguments
		}
		fixArgType(name, arg, applied)
		fixArgValues(name, arg, applied)
	}
}

// fixModules applies module migrations in slice order.
func fixModules(root *yaml.Node, applied *[]Applied) {
	mi := findKey(root, "modules")
	if mi < 0 {
		return
	}
	modules := root.Content[mi+1]
	if modules.Kind != yaml.SequenceNode {
		return
	}
	for i, mod := range modules.Content {
		if mod.Kind != yaml.MappingNode {
			continue
		}
		fixModulePrerelease(moduleFixPath(mod, i), mod, applied)
	}
}

// moduleFixPath builds the finding path for module i, preferring its name,
// mirroring modulePath in the checker.
func moduleFixPath(mod *yaml.Node, i int) string {
	if ni := findKey(mod, "name"); ni >= 0 && mod.Content[ni+1].Value != "" {
		return "modules." + mod.Content[ni+1].Value
	}
	return "modules[" + strconv.Itoa(i) + "]"
}
