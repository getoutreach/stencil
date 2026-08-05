// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the refines() subschema check.

package projectmanifest

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestRefines(t *testing.T) {
	cases := []struct {
		name   string
		child  map[string]any
		parent map[string]any
		want   bool
	}{
		{"identical", map[string]any{"type": "string"}, map[string]any{"type": "string"}, true},
		{"enum added where parent has none",
			map[string]any{"type": "string", "enum": []any{"a", "b"}},
			map[string]any{"type": "string"}, true},
		{"enum subset",
			map[string]any{"enum": []any{"a"}},
			map[string]any{"enum": []any{"a", "b"}}, true},
		{"enum not a subset",
			map[string]any{"enum": []any{"z"}},
			map[string]any{"enum": []any{"a", "b"}}, false},
		{"type widened",
			map[string]any{"type": "number"},
			map[string]any{"type": "string"}, false},
		{"add object properties",
			map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "boolean"}}},
			map[string]any{"type": "object"}, true},
		{"drop a required key",
			map[string]any{"type": "object"},
			map[string]any{"type": "object", "required": []any{"x"}}, false},
		{"additionalProperties true->false is narrowing",
			map[string]any{"type": "object", "additionalProperties": false},
			map[string]any{"type": "object", "additionalProperties": true}, true},
		{"additionalProperties false->true is loosening",
			map[string]any{"type": "object", "additionalProperties": true},
			map[string]any{"type": "object", "additionalProperties": false}, false},
		{"tighten numeric bound",
			map[string]any{"type": "integer", "minimum": float64(5)},
			map[string]any{"type": "integer", "minimum": float64(1)}, true},
		{"loosen numeric bound",
			map[string]any{"type": "integer", "minimum": float64(0)},
			map[string]any{"type": "integer", "minimum": float64(1)}, false},
		{"add pattern where parent had none",
			map[string]any{"type": "string", "pattern": "^a"},
			map[string]any{"type": "string"}, true},
		{"differing pattern",
			map[string]any{"type": "string", "pattern": "^a"},
			map[string]any{"type": "string", "pattern": "^b"}, false},
		{"unknown keyword differs (equality fallback)",
			map[string]any{"type": "string", "wibble": float64(1)},
			map[string]any{"type": "string", "wibble": float64(2)}, false},
		{"anyOf branch narrowing (commands shape)",
			map[string]any{"anyOf": []any{
				map[string]any{"type": "string"},
				map[string]any{"type": "object", "properties": map[string]any{
					"delibird": map[string]any{"type": "boolean"},
				}},
			}},
			map[string]any{"anyOf": []any{
				map[string]any{"type": "string"},
				map[string]any{"type": "object"},
			}}, true},
		{"closed parent, child adds a property (loosening)",
			map[string]any{"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"x": map[string]any{"type": "string"},
					"y": map[string]any{"type": "string"},
				}},
			map[string]any{"type": "object", "additionalProperties": false,
				"properties": map[string]any{"x": map[string]any{"type": "string"}}}, false},
		{"closed parent, child same properties (valid)",
			map[string]any{"type": "object", "additionalProperties": false,
				"properties": map[string]any{"x": map[string]any{"type": "string"}}},
			map[string]any{"type": "object", "additionalProperties": false,
				"properties": map[string]any{"x": map[string]any{"type": "string"}}}, true},
		{"anyOf parent with sibling type, child drops it (loosening)",
			map[string]any{"anyOf": []any{map[string]any{"minimum": float64(0)}}},
			map[string]any{"type": "integer", "anyOf": []any{map[string]any{"minimum": float64(0)}}}, false},
		{"parent additionalProperties schema, child drops it (loosening)",
			map[string]any{"type": "object"},
			map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}, false},
		{"parent additionalProperties schema, child forbids all (narrowing)",
			map[string]any{"type": "object", "additionalProperties": false},
			map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}}, true},
		{"parent additionalProperties schema, child adds a conflicting named property (loosening)",
			map[string]any{"type": "object",
				"properties": map[string]any{
					"x": map[string]any{"type": "string"},
					"y": map[string]any{"type": "integer"},
				},
				"additionalProperties": map[string]any{"type": "string"}},
			map[string]any{"type": "object",
				"properties":           map[string]any{"x": map[string]any{"type": "string"}},
				"additionalProperties": map[string]any{"type": "string"}}, false},
		{"parent additionalProperties schema, child adds a conforming named property (narrowing)",
			map[string]any{"type": "object",
				"properties": map[string]any{
					"x": map[string]any{"type": "string"},
					"y": map[string]any{"type": "string", "minLength": float64(1)},
				},
				"additionalProperties": map[string]any{"type": "string"}},
			map[string]any{"type": "object",
				"properties":           map[string]any{"x": map[string]any{"type": "string"}},
				"additionalProperties": map[string]any{"type": "string"}}, true},
		{"parent oneOf may overlap (cannot verify)",
			map[string]any{"const": "a"},
			map[string]any{"oneOf": []any{
				map[string]any{"type": "string"},
				map[string]any{"enum": []any{"a"}},
			}}, false},
		{"child anyOf boolean branch accepts anything (cannot verify)",
			map[string]any{"anyOf": []any{map[string]any{"type": "string"}, true}},
			map[string]any{"type": "string"}, false},
		{"child anyOf is not a list (cannot verify)",
			map[string]any{"anyOf": "nonsense"},
			map[string]any{"type": "string"}, false},
		{"parent items is a bool (cannot verify)",
			map[string]any{"type": "array"},
			map[string]any{"type": "array", "items": true}, false},
		{"parent property is a bool (cannot verify)",
			map[string]any{"type": "object", "properties": map[string]any{"x": map[string]any{"type": "string"}}},
			map[string]any{"type": "object", "properties": map[string]any{"x": true}}, false},
		{"child oneOf narrows an anyOf parent (valid)",
			map[string]any{"oneOf": []any{map[string]any{"type": "string"}}},
			map[string]any{"anyOf": []any{
				map[string]any{"type": "string"},
				map[string]any{"type": "integer"},
			}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, reason := refines(tc.child, tc.parent)
			assert.Equal(t, tc.want, got, "reason: %s", reason)
		})
	}
}

func TestRefinesReasonNamesKeyword(t *testing.T) {
	ok, reason := refines(map[string]any{"type": "number"}, map[string]any{"type": "string"})
	assert.Assert(t, !ok)
	assert.Assert(t, contains(reason, "type")) // contains() is defined in online_test.go
}

func TestRefinesServiceActivities(t *testing.T) {
	parent := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	child := map[string]any{"type": "array", "items": map[string]any{
		"type": "string", "enum": []any{"http", "grpc", "temporal", "python"},
	}}
	ok, reason := refines(child, parent)
	assert.Assert(t, ok, "reason: %s", reason)
}
