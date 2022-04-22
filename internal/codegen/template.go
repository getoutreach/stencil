// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the logic and type for a template
// that is being rendered by stencil.
package codegen

import (
	"bytes"
	"os"
	"path"
	"strings"
	"time"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/sirupsen/logrus"
)

// Template is a file that has been processed by stencil
type Template struct {
	// parsed denotes if this template has been parsed or not
	parsed bool

	// args are the arguments passed to the template
	args *Values

	// log is the logger to use for debug logging
	log logrus.FieldLogger

	// mode is the os file mode of the template, this is used
	// for the default file if not modified during render time
	mode os.FileMode

	// modTime is the modification time of the template, this is used
	// for the default file if not modified during render time
	modTime time.Time

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
func NewTemplate(m *modules.Module, fpath string, mode os.FileMode,
	modTime time.Time, contents []byte, log logrus.FieldLogger) (*Template, error) {
	return &Template{
		log:      log,
		mode:     mode,
		modTime:  modTime,
		Module:   m,
		Path:     fpath,
		Contents: contents,
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
	// Add the current template to the template object on the module that we're
	// attached to. This enables us to call functions in other templates within our
	// 'module context'.
	if _, err := t.Module.GetTemplate().New(t.ImportPath()).Funcs(NewFuncMap(nil, nil, t.log)).
		Parse(string(t.Contents)); err != nil {
		return err
	}

	t.parsed = true

	return nil
}

// Render renders the provided template, the produced files
// are rendered onto the Files field of the template struct.
func (t *Template) Render(st *Stencil, vals *Values) error {
	if len(t.Files) == 0 {
		f, err := NewFile(strings.TrimSuffix(t.Path, ".tpl"), t.mode, t.modTime)
		if err != nil {
			return err
		}
		t.Files = []*File{f}
	}

	// Parse the template if we haven't already
	if !t.parsed {
		if err := t.Parse(st); err != nil {
			return err
		}
	}

	t.args = vals

	// Execute a specific file because we're using a shared template, if we attempt to render
	// the entire template we'll end up just rendering the base template (<module>) which is empty
	var buf bytes.Buffer
	if err := t.Module.GetTemplate().Funcs(NewFuncMap(st, t, t.log)).
		ExecuteTemplate(&buf, t.ImportPath(), vals); err != nil {
		return err
	}

	// If we're writing only a single file, and the contents is empty
	// then we should write the output of the rendered template.
	//
	// This ensures that templates don't need to call file.Create
	// by default, only when they want to customize the output
	if len(t.Files) == 1 && len(t.Files[0].Bytes()) == 0 {
		t.Files[0].SetContents(buf.String())
	} else if len(t.Files) > 1 {
		// otherwise, remove the first file that was created when
		// we constructed the template. It's only used when we have
		// no calls to file.Create
		t.Files = t.Files[1:len(t.Files)]
	}

	return nil
}
