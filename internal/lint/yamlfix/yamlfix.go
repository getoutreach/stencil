// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Shared, generic yaml.Node DOM primitives used by the Stencil
// manifest lint fixers and line resolvers. These are domain-agnostic
// mapping/alias helpers; manifest- or project-specific logic lives in its own
// package.

// Package yamlfix provides generic yaml.Node DOM primitives shared by the
// Stencil manifest lint fixers and line resolvers.
package yamlfix

import "go.yaml.in/yaml/v3"

// FindKey returns the index in m.Content of the key node named key, or -1.
// m must be a MappingNode (Content is a flat [key, value, key, value, ...]).
func FindKey(m *yaml.Node, key string) int {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// RemoveKey deletes the key/value pair named key from mapping m in place and
// returns the removed value node, or nil if key was absent. Surviving pairs
// keep their relative order.
func RemoveKey(m *yaml.Node, key string) *yaml.Node {
	i := FindKey(m, key)
	if i < 0 {
		return nil
	}
	val := m.Content[i+1]
	m.Content = append(m.Content[:i], m.Content[i+2:]...)
	return val
}

// EnsureMapping returns the existing child mapping stored under key in m, or
// appends a new empty MappingNode under key and returns it.
func EnsureMapping(m *yaml.Node, key string) *yaml.Node {
	if i := FindKey(m, key); i >= 0 {
		return m.Content[i+1]
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	v := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content, k, v)
	return v
}

// SetIfAbsent appends key: val to mapping m only when key is absent. It returns
// true when it added the pair. The val node is used as-is (its Style and
// comments are preserved).
func SetIfAbsent(m *yaml.Node, key string, val *yaml.Node) bool {
	if FindKey(m, key) >= 0 {
		return false
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	m.Content = append(m.Content, k, val)
	return true
}

// CarryKeyComments copies the head and foot comments from src (the original
// key node being migrated away) onto the key node named key in mapping m.
// yaml.v3 stores a mapping entry's head/foot comments on its KEY node, so when
// a migration synthesizes a new key it must carry these over or they are lost.
func CarryKeyComments(m *yaml.Node, key string, src *yaml.Node) {
	i := FindKey(m, key)
	if i < 0 {
		return
	}
	m.Content[i].HeadComment = src.HeadComment
	m.Content[i].FootComment = src.FootComment
}

// Deref follows an alias node to its anchored target, returning other nodes
// unchanged. A nil node returns nil.
func Deref(n *yaml.Node) *yaml.Node {
	if n != nil && n.Kind == yaml.AliasNode && n.Alias != nil {
		return n.Alias
	}
	return n
}
