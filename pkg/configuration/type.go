// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: types of stencil repos

package configuration

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// TemplateRepositoryType specifies what type of a stencil repository the current one is.
type TemplateRepositoryType string

// This block contains all of the TemplateRepositoryType values
const (
	// TemplateRepositoryTypeExt denotes a repository as being
	// an extension repository. This means that it contains
	// a go extension. This repository may also contain go-templates if
	// this type is used together with the TemplateRepositoryTypeTemplates.
	TemplateRepositoryTypeExt TemplateRepositoryType = "extension"

	// TemplateRepositoryTypeTemplates denotes a repository as being a standard template repository.
	// When the same module/repo serves more than one type, join this explicit value with other
	// types, e.g. "templates,extension".
	TemplateRepositoryTypeTemplates TemplateRepositoryType = "templates"
)

// TemplateRepositoryTypes specifies what type of a stencil repository the current one is.
// Use Contains to check for a type - it has special handling for the default case.
// Even though it is a struct, it is marshalled and unmarshalled as a string with comma separated
// values of TemplateRepositoryType.
type TemplateRepositoryTypes struct {
	types []TemplateRepositoryType
}

// MarshalYAML marshals TemplateRepositoryTypes as a string with comma-separated values.
func (ts TemplateRepositoryTypes) MarshalYAML() (interface{}, error) {
	var csv []string
	for _, t := range ts.types {
		csv = append(csv, string(t))
	}
	return strings.Join(csv, ","), nil
}

// UnmarshalYAML unmarshals TemplateRepositoryTypes from a string with comma-separated values.
func (ts *TemplateRepositoryTypes) UnmarshalYAML(value *yaml.Node) error {
	var csv string
	if err := value.Decode(&csv); err != nil {
		return err
	}

	if csv == "" {
		// empty type defaults to templates only (we do not support repos with no purpose)
		// leave the slice empty for a consistent unmarshal/marshal roundtrip
		ts.types = nil
		return nil
	}

	items := strings.Split(csv, ",")
	types := []TemplateRepositoryType{}
	for _, i := range items {
		types = append(types, TemplateRepositoryType(i))
	}
	ts.types = types
	return nil
}

// Contains returns true if current repo needs to serve inpt type, with default assumed
// to be a templates-only repo (we do not support repos with no purpose).
func (ts TemplateRepositoryTypes) Contains(t TemplateRepositoryType) bool {
	if len(ts.types) == 0 {
		// empty types defaults to templates only (we do not support repos with no purpose)
		return t == TemplateRepositoryTypeTemplates
	}
	for _, ti := range ts.types {
		if ti == t {
			return true
		}
	}
	return false
}
