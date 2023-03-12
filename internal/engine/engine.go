// Copyright 2023 Outreach Corporation. All Rights Reserved.

// Description: This file contains the engines for rendering templates.

// Package engine contains logic for the template engines supported by stencil.
package engine

import (
	"github.com/getoutreach/stencil/internal/engine/htmltemplate"
	"github.com/getoutreach/stencil/internal/engine/jet"
	"github.com/getoutreach/stencil/internal/engine/texttemplate"
)

// Name is the name of a template engine
type Name string

// Contains the name of valid template rendering engines
const (
	// NameTextTemplate is the name of the text/template go-template engine
	NameTextTemplate Name = "go-template:text/template"

	// NameHTMLTemplate is the name of the html/template go-template engine
	NameHTMLTemplate Name = "go-template:html/template"

	// NameJet is the name of the jet templating language engine
	NameJet Name = "jet"
)

// engines is a map of all the engines that are available
var engines = map[Name]NewInstance{
	NameTextTemplate: func(moduleName string) (Instance, error) {
		return texttemplate.NewInstance(moduleName)
	},
	NameHTMLTemplate: func(moduleName string) (Instance, error) {
		return htmltemplate.NewInstance(moduleName)
	},
	NameJet: func(moduleName string) (Instance, error) {
		return jet.NewInstance(moduleName)
	},
}

// engineForExtensions is a map to engine names for a given extension
var engineForExtensions = map[Name][]string{
	NameHTMLTemplate: {".htpl"},
	NameTextTemplate: {".tpl"},
	NameJet:          {".jet"},
}

// GetEngine returns a NewInstance function for the given engine name
func GetEngine(name Name) (NewInstance, bool) {
	eng, ok := engines[name]
	return eng, ok
}

// GetEngineNameForExtension returns the engine name for a given
// extension
func GetEngineNameForExtension(ext string) Name {
	for name, exts := range engineForExtensions {
		for _, e := range exts {
			if e == ext {
				return name
			}
		}
	}

	return ""
}

// GetEngineExtensions returns all valid template extensions for
// all engines.
func GetEngineExtensions() []string {
	all := []string{}
	for _, exts := range engineForExtensions {
		all = append(all, exts...)
	}
	return all
}
