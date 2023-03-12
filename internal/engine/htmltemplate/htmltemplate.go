// Copyright 2023 Outreach Corporation. All Rights Reserved.

// Description: This file implements the html/template go-template renderer
// as an engine.

// Package htmltemplate implements a html/template engine using the go-template
// renderer as an engine.
package htmltemplate

import (
	"html/template"
	"io"

	"github.com/Masterminds/sprig/v3"
)

// NewInstance returns a new instance of the html/template go-template engine
func NewInstance(moduleName string) (*Instance, error) {
	return &Instance{
		t: template.New(moduleName).Funcs(sprig.HtmlFuncMap()),
	}, nil
}

// Instance is an instance of a html/template go-template engine
type Instance struct {
	// t is the underlying template used by this engine instance
	t *template.Template
}

// Parse parses a template and adds it to the current template instance
func (i *Instance) Parse(name string, r io.Reader, fns map[string]any) error {
	contents, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	_, err = i.t.New(name).Funcs(fns).Parse(string(contents))
	return err
}

// Render renders a template into the provide writer
func (i *Instance) Render(name string, out io.Writer, fns map[string]any, args any) error {
	tpl := i.t
	if fns != nil {
		tpl = tpl.Funcs(fns)
	}
	return tpl.ExecuteTemplate(out, name, args)
}
