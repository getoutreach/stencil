// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains helpers for creating
// functions exposed to stencil codegen files.

package codegen

import (
	"context"
	"text/template"

	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/sirupsen/logrus"
)

// NewFuncMap returns the standard func map for a template
func NewFuncMap(ctx context.Context, st *Stencil, t *Template, log logrus.FieldLogger) (template.FuncMap, error) {
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

	extCaller, err := st.ext.GetExtensionCaller(ctx)
	if err != nil {
		return nil, err
	}

	// build the function map
	funcs := Default
	funcs["stencil"] = func() *TplStencil { return tplst }
	funcs["file"] = func() *TplFile { return tplf }
	funcs["extensions"] = func() *extensions.ExtensionCaller { return extCaller }
	return funcs, nil
}
