// Copyright 2024 Outreach Corporation. All Rights Reserved.

// Description: Manifest to JSON Schema library.

// Package jsonschema creates JSON schemas.
package jsonschema

import "encoding/json"

// Schema is a representation of a JSON schema.
type Schema struct {
	Version     string     `json:"$schema"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Type        Types      `json:"type"`
	Properties  Properties `json:"properties"`
}

// Types is one or more JSON schema types.
type Types struct {
	types []string
}

func NewTypes(types ...string) Types {
	return Types{types}
}

func (t Types) MarshalJSON() ([]byte, error) {
	if len(t.types) == 1 {
		return json.Marshal(t.types[0])
	}

	return json.Marshal(t.types)
}

// Properties is a map of property names to definitions.
type Properties = map[string]*Property

// Property describes a JSON schema (sub-)property.
type Property struct {
	Description string `json:"description"`
	Types       Types  `json:"type"`
	Items       *Items `json:"items,omitempty"`
	// Can't use the Properties type here because of an issue with
	// type alias + recursion
	Properties *map[string]Property `json:"properties,omitempty"`
}

// Items describes the subtype of a JSON array.
type Items struct {
	Type Types `json:"type"`
}

// NewSchema creates a new JSON schema.
func NewSchema(title, description string) *Schema {
	return &Schema{
		Version:     "https://json-schema.org/draft/2020-12/schema",
		Title:       title,
		Description: description,
		Properties:  make(Properties),
		Type:        NewTypes("object"),
	}
}

// AddProperty adds a property to the top level of a JSON schema.
func (s *Schema) AddProperty(name, description string, types Types) {
	s.Properties[name] = &Property{
		Description: description,
		Types:       types,
	}
}

// AddArrayProperty adds an array property to the top level of a JSON schema.
func (s *Schema) AddArrayProperty(name, description, itemType string) {
	s.Properties[name] = &Property{
		Description: description,
		Types:       NewTypes("array"),
		Items:       &Items{NewTypes(itemType)},
	}
}
