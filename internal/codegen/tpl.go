// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for creating
// functions exposed to stencil codegen files.

package codegen

import (
	"text/template"

	"github.com/sirupsen/logrus"
)

// NewFuncMap returns the standard func map for a template
func NewFuncMap(st *Stencil, t *Template, log logrus.FieldLogger) template.FuncMap {
	// We allow tplst & tplf to be nil in the case of
	// .Parse() of a template, where they need to be present
	// but aren't actually executed by the template
	// (execute is the one that renders it)
	var tplst *TplStencil
	var tplf *TplFile
	if st != nil {
		tplst = &TplStencil{st, t, log}
	}
	if t != nil {
		tplf = &TplFile{t.Files[0], t, log}
	}

	// build the function map
	funcs := Default
	funcs["stencil"] = func() *TplStencil { return tplst }
	funcs["file"] = func() *TplFile { return tplf }
	return funcs
}
