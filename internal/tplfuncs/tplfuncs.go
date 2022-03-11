// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the package documentation for the
// tplfuncs package.

// Package tplfuncs contains all functions that are exposed to templates
// in stencil.
package tplfuncs

import (
	"text/template"

	"github.com/getoutreach/stencil/internal/functions"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// New returns initialized template functions that can be passed to a template
func New(m *configuration.ServiceManifest, s *functions.Stencil,
	t *functions.Template) (*Stencil, *File) {
	return &Stencil{s, m}, &File{t.Files[0], t}
}

// NewFuncMap creates a new template.FuncMap from provided functions
func NewFuncMap(s *Stencil, f *File) template.FuncMap {
	funcs := functions.Default
	if s != nil {
		funcs["stencil"] = func() *Stencil { return s }
	}
	if f != nil {
		funcs["file"] = func() *File { return f }
	}
	return funcs
}
