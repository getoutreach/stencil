// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the logic and type for a template
// that is being rendered by stencil.
package functions

import (
	"bytes"
	"os"
	"path"
	"strings"
	"time"

	"github.com/getoutreach/stencil/internal/modules"
)

// Template is a file that has been processed by stencil
type Template struct {
	// parsed denotes if this template has been parsed or not
	parsed bool

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
func NewTemplate(m *modules.Module, fpath string, mode os.FileMode, modTime time.Time, contents []byte) (*Template, error) {
	f, err := NewFile(strings.TrimSuffix(fpath, ".tpl"), mode, modTime)
	if err != nil {
		return nil, err
	}

	return &Template{
		Module:   m,
		Path:     fpath,
		Contents: contents,
		Files:    []*File{f},
	}, nil
}

// ImportPath returns the path to this template, this is meant to denote
// which module this template is attached to
func (t *Template) ImportPath() string {
	return path.Join(t.Module.Name, t.Path)
}

// Parse parses the provided template and makes it available to be Rendered
// in the context of the current module.
func (t *Template) Parse(st *Stencil) error {
	// Return stub types to make the 'compiler' happy, we pass
	// the real values in render.
	funcs := Default
	funcs["stencil"] = func() *TplStencil { return nil }
	funcs["file"] = func() *TplFile { return nil }

	// Add the current template to the template object on the module that we're
	// attached to. This enables us to call functions in other templates within our
	// 'module context'.
	if _, err := t.Module.GetTemplate().New(t.ImportPath()).Funcs(funcs).
		Parse(string(t.Contents)); err != nil {
		return err
	}

	t.parsed = true

	return nil
}

// Render renders the provided template, the produced files
// are rendered onto the Files field of the template struct.
func (t *Template) Render(st *Stencil, args map[string]interface{}) error {
	// Parse the template if we haven't already
	if !t.parsed {
		if err := t.Parse(st); err != nil {
			return err
		}
	}

	// Create global stencil/file objects that will be mutated by
	// any templates rendered during the render context of this template.
	tplst := &TplStencil{st, st.m}
	tplf := &TplFile{t.Files[0], t}

	// Note: We "overwrite" the stub functions that were used during Parse()
	// time.
	funcs := Default
	funcs["stencil"] = func() *TplStencil { return tplst }
	funcs["file"] = func() *TplFile { return tplf }

	// Execute a specific file because we're using a shared template, if we attempt to render
	// the entire template we'll end up just rendering the base template (<module>) which is empty
	var buf bytes.Buffer
	if err := t.Module.GetTemplate().Funcs(funcs).ExecuteTemplate(&buf, t.ImportPath(), funcs); err != nil {
		return err
	}

	// If we're writing only a single file, and the contents is empty
	// then we should write the output of the rendered template.
	if len(t.Files) == 1 && len(t.Files[0].Bytes()) == 0 {
		t.Files[0].SetContents(buf.String())
	}

	return nil
}
