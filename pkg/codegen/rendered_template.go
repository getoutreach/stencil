package codegen

import (
	"bytes"
	"io"
	"strings"
	"text/template"

	"github.com/getoutreach/stencil/pkg/configuration"
)

// RenderedTemplate is a file that has been processed by stencil
type RenderedTemplate struct {
	io.Reader

	// Skipped marks if this file should be skipped or not
	Skipped bool

	// SkipReason is the reason this file was skipped
	// only valid when Skip is true and is optional
	SkipReason string

	// Deleted marks if this file has been deleted or not
	Deleted bool

	// Path is the path this template should be written to
	// Note: This is a relative path, b.Dir is injected
	// at the writeFile step as the base.
	Path string

	// Warnings is an array of warnings that were created
	// while rendering this template
	Warnings []string
}

// AddDeprecationNotice adds a warning to the rendered template file
func (rt *RenderedTemplate) AddDeprecationNotice(msg string) string {
	if rt.Warnings == nil {
		rt.Warnings = []string{msg}
		return ""
	}

	rt.Warnings = append(rt.Warnings, msg)
	return ""
}

// MarkStatic marks this rendered template as being static
func (rt *RenderedTemplate) MarkStatic() string {
	rt.Skipped = true
	rt.SkipReason = "File was marked as static"
	return ""
}

// Skip marks this template as skipped with a user-providable
// reason.
func (rt *RenderedTemplate) Skip(reason string) string {
	rt.Skipped = true
	rt.SkipReason = reason
	return ""
}

// Delete marks this rendered template as being deleted
func (rt *RenderedTemplate) Delete() string {
	rt.Deleted = true
	return ""
}

// SetPath sets the path of this template
func (rt *RenderedTemplate) SetPath(name string) string {
	rt.Path = name
	return ""
}

// Stencil provides the basic functions for
// stencil templates
type Stencil struct {
	*template.Template
	m *configuration.ServiceManifest

	// Files is a list of files that this rendered produced
	Files []*RenderedTemplate

	// File is the current file that is being rendered by this
	// renderer.
	File *RenderedTemplate
}

// ApplyTemplate executes a template inside of the current rendered
// template
func (s *Stencil) ApplyTemplate(name string) (string, error) {
	var buf bytes.Buffer
	err := s.Template.ExecuteTemplate(&buf, name, nil)
	return buf.String(), err
}

// Arg returns an argument from the ServiceManifest
func (s *Stencil) Arg(name string) interface{} {
	return s.m.Arguments[name]
}

// InstallFile changes the current active rendered file and writes
// the provided contents to it. This changes the scope of the
// current "File" being rendered.
func (s *Stencil) InstallFile(name, contents string) string {
	rt := &RenderedTemplate{
		Path:   name,
		Reader: strings.NewReader(contents),
	}
	s.Files = append(s.Files, rt)
	s.File = rt
	return ""
}
