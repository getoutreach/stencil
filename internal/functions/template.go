// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the logic and type for a template
// that is being rendered by stencil.
package functions

import (
	"bytes"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/getoutreach/stencil/internal/modules"
)

// Template is a file that has been processed by stencil
type Template struct {
	// Module is the underlying module that's creating this template
	Module *modules.Module

	// Path is the path of this template relative to the owning module
	Path string

	// Files is a list of files that this template generated
	Files []*File

	// Contents is the content of this template
	Contents []byte
}

// NewTemplate creates a new Template with the current file being the same name
// with the extension .tpl being removed.
func NewTemplate(m *modules.Module, path string, mode os.FileMode, modTime time.Time, contents []byte) (*Template, error) {
	// TODO(jaredallard): create the first file when we create
	// a template.
	f, err := NewFile(strings.TrimSuffix(path, ".tpl"), mode, modTime)
	if err != nil {
		return nil, err
	}

	return &Template{
		Module:   m,
		Path:     path,
		Contents: contents,
		Files:    []*File{f},
	}, nil
}

// Render renders the provided template, the produced files
// are rendered onto the Files field of the template struct.
func (t *Template) Render(funcs template.FuncMap, args map[string]interface{}) error {
	// Add sprig functions
	if _, err := t.Module.GetTemplate().Funcs(sprig.TxtFuncMap()).
		Funcs(funcs).Parse(string(t.Contents)); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := t.Module.GetTemplate().Execute(&buf, args); err != nil {
		return err
	}

	// If we're writing only a single file, and the contents is empty
	// then we should write the output of the rendered template.
	if len(t.Files) == 1 && len(t.Files[0].Bytes()) == 0 {
		t.Files[0].SetContents(buf.String())
	}

	return nil
}
