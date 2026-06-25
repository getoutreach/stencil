// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Resolves dotted lint finding paths (e.g. "arguments.foo.type")
// to the 1-based source line of the corresponding YAML key node, for line
// annotation of manifest lint findings.

package manifest

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// resolvePath returns the 1-based line of the key node identified by a dotted
// finding path within root, or 0 if root is nil or the path cannot be matched.
// root is a yaml.v3 DocumentNode; the top mapping is its first content child.
//
// Module paths are special-cased because module names are Go import paths that
// contain dots (e.g. github.com/getoutreach/stencil-base), which a naive split
// on "." would shred. All other paths are walked as dotted mapping keys.
//
// On any failure to match, resolvePath returns 0. It never panics.
func resolvePath(root *yaml.Node, path string) int {
	if root == nil || path == "" {
		return 0
	}
	top := deref(root)
	if top != nil && top.Kind == yaml.DocumentNode {
		if len(top.Content) == 0 {
			return 0
		}
		top = deref(top.Content[0])
	}
	if top == nil || top.Kind != yaml.MappingNode {
		return 0
	}

	// Module paths need bespoke parsing (names contain dots).
	if path == "modules" || strings.HasPrefix(path, "modules.") || strings.HasPrefix(path, "modules[") {
		return resolveModulePath(top, path)
	}

	// General dotted mapping walk.
	cur := top
	lastKeyLine := 0
	for _, seg := range strings.Split(path, ".") {
		if cur == nil || cur.Kind != yaml.MappingNode {
			return 0
		}
		keyNode, valNode := mappingChild(cur, seg)
		if keyNode == nil {
			return 0
		}
		lastKeyLine = keyNode.Line
		cur = deref(valNode)
	}
	return lastKeyLine
}

// resolveModulePath resolves "modules[N].field" and "modules.NAME.field" within
// the top mapping, returning the field key's line (or the matched item's name:
// line when no field segment is present). NAME may contain dots. Returns 0 on
// any miss.
func resolveModulePath(top *yaml.Node, path string) int {
	rest := strings.TrimPrefix(path, "modules")
	_, seqVal := mappingChild(top, "modules")
	seq := deref(seqVal)
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return 0
	}

	// Index form: modules[N].field
	if strings.HasPrefix(rest, "[") {
		closeIdx := strings.IndexByte(rest, ']')
		if closeIdx < 0 {
			return 0
		}
		idx, err := strconv.Atoi(rest[1:closeIdx])
		if err != nil || idx < 0 || idx >= len(seq.Content) {
			return 0
		}
		field := strings.TrimPrefix(rest[closeIdx+1:], ".")
		return fieldLineIn(deref(seq.Content[idx]), field)
	}

	// Name form: modules.NAME.field — the LAST dot separates the (dotted) NAME
	// from the trailing field.
	rest = strings.TrimPrefix(rest, ".")
	lastDot := strings.LastIndexByte(rest, '.')
	if lastDot < 0 {
		return 0
	}
	name, field := rest[:lastDot], rest[lastDot+1:]
	item, nameLine := sequenceItemByName(seq, name)
	if item == nil {
		return 0
	}
	if field == "" {
		return nameLine
	}
	return fieldLineIn(item, field)
}

// fieldLineIn returns the key line of field within mapping node m, or 0 when
// field is empty, m is not a mapping, or the key is absent.
func fieldLineIn(m *yaml.Node, field string) int {
	if field == "" {
		return 0
	}
	keyNode, _ := mappingChild(m, field)
	if keyNode == nil {
		return 0
	}
	return keyNode.Line
}

// deref follows an alias node to its anchored target, returning other nodes
// unchanged. A nil node returns nil.
func deref(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.AliasNode && n.Alias != nil {
		return n.Alias
	}
	return n
}

// mappingChild finds the key/value pair in a mapping node whose key scalar
// equals key. Returns (nil, nil) if not found or m is not a mapping.
func mappingChild(m *yaml.Node, key string) (keyNode, valNode *yaml.Node) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		if k.Value == key {
			return k, m.Content[i+1]
		}
	}
	return nil, nil
}

// sequenceItemByName scans a sequence of mapping nodes for the first item whose
// name: child value equals name. Returns the item node and the line of its
// name: key, or (nil, 0) if not found.
func sequenceItemByName(seq *yaml.Node, name string) (item *yaml.Node, nameLine int) {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil, 0
	}
	for _, raw := range seq.Content {
		it := deref(raw)
		keyNode, valNode := mappingChild(it, "name")
		if keyNode != nil && valNode != nil && valNode.Value == name {
			return it, keyNode.Line
		}
	}
	return nil, 0
}
