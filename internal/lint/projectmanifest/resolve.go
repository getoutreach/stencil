// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Resolves dotted project-manifest lint finding paths to 1-based
// source lines for annotation.

package projectmanifest

import (
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/getoutreach/stencil/internal/lint/yamlfix"
)

// opaqueLeafPrefixes are the top-level keys whose entries are opaque flat leaf
// keys: their sub-keys may contain dots (e.g. import paths) but their values are
// scalars, so a matching finding path has no trailing field to walk into.
var opaqueLeafPrefixes = []string{"arguments.", "versions.", "replacements."}

// resolvePath returns the 1-based line of the key identified by a dotted finding
// path within root, or 0 if unresolvable. root is a yaml.v3 DocumentNode.
//
// modules.<name>.<field> and modules[i].<field> get bespoke parsing because
// module names are Go import paths containing dots. arguments/versions/
// replacements keys are opaque flat leaf keys (their values are scalars, so
// there is no trailing field). Everything else is a plain dotted mapping walk.
func resolvePath(root *yaml.Node, path string) int {
	if root == nil || path == "" {
		return 0
	}
	top := yamlfix.Deref(root)
	if top != nil && top.Kind == yaml.DocumentNode {
		if len(top.Content) == 0 {
			return 0
		}
		top = yamlfix.Deref(top.Content[0])
	}
	if top == nil || top.Kind != yaml.MappingNode {
		return 0
	}

	if path == "modules" || strings.HasPrefix(path, "modules.") || strings.HasPrefix(path, "modules[") {
		return resolveModulePath(top, path)
	}
	// arguments.<key>, versions.<key>, replacements.<key>: opaque flat leaf key.
	for _, p := range opaqueLeafPrefixes {
		if after, ok := strings.CutPrefix(path, p); ok {
			return resolveLeafKey(top, after, strings.TrimSuffix(p, "."))
		}
	}

	// General dotted mapping walk (covers "name").
	cur := top
	lastKeyLine := 0
	for seg := range strings.SplitSeq(path, ".") {
		if cur == nil || cur.Kind != yaml.MappingNode {
			return 0
		}
		keyNode, valNode := mappingChild(cur, seg)
		if keyNode == nil {
			return 0
		}
		lastKeyLine = keyNode.Line
		cur = yamlfix.Deref(valNode)
	}
	return lastKeyLine
}

// resolveLeafKey resolves "<container>.<key>" where key is a single flat map key
// whose value is opaque (e.g. versions.golang). Returns the key node's line.
func resolveLeafKey(top *yaml.Node, key, container string) int {
	_, containerVal := mappingChild(top, container)
	m := yamlfix.Deref(containerVal)
	if m == nil || m.Kind != yaml.MappingNode {
		return 0
	}
	keyNode, _ := mappingChild(m, key)
	if keyNode == nil {
		return 0
	}
	return keyNode.Line
}

// resolveModulePath resolves modules[i].field and modules.NAME.field (NAME may
// contain dots; the last dot separates NAME from field).
func resolveModulePath(top *yaml.Node, path string) int {
	rest := strings.TrimPrefix(path, "modules")
	_, seqVal := mappingChild(top, "modules")
	seq := yamlfix.Deref(seqVal)
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return 0
	}
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
		return fieldLineIn(yamlfix.Deref(seq.Content[idx]), field)
	}
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

// fieldLineIn returns the 1-based line of key field within mapping m, or 0 if
// m is not a mapping, field is empty, or the key is absent.
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

func mappingChild(m *yaml.Node, key string) (keyNode, valNode *yaml.Node) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, nil
	}
	if i := yamlfix.FindKey(m, key); i >= 0 {
		return m.Content[i], m.Content[i+1]
	}
	return nil, nil
}

// sequenceItemByName finds the mapping item in seq whose "name" scalar equals
// name, returning the item node and its name key's 1-based line, or nil/0 if no
// item matches.
func sequenceItemByName(seq *yaml.Node, name string) (item *yaml.Node, nameLine int) {
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil, 0
	}
	for _, raw := range seq.Content {
		it := yamlfix.Deref(raw)
		keyNode, valNode := mappingChild(it, "name")
		if keyNode != nil && valNode != nil && valNode.Value == name {
			return it, keyNode.Line
		}
	}
	return nil, 0
}
