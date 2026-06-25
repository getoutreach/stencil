// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements safe, comment-preserving auto-fixes for a Stencil
// template repository manifest (manifest.yaml). The fixer edits the yaml.Node
// DOM in place so comments and key order survive, and only applies migrations
// that are unambiguous.

package manifest

import "gopkg.in/yaml.v3"

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
