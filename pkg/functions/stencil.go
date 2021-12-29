// Copyright 2021 Outreach Corporation. All Rights Reserved.

// Description: Implements the stencil function passed to templates
package functions

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/getoutreach/stencil/pkg/configuration"
)

// NewStencil creates a new, fully initialized Stencil
func NewStencil(t *template.Template, m *configuration.ServiceManifest,
	file *RenderedTemplate) *Stencil {
	return &Stencil{Template: t, m: m, Files: make([]*RenderedTemplate, 0),
		File: file}
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
// template.
//
//   {{- define "command"}}
//   package main
//
//   import "fmt"
//
//   func main() {
//     fmt.Println("hello, world!")
//   }
//
//   {{- end }}
//
//   {{- stencil.ApplyTemplate "command" | stencil.InstallFile "cmd/main.go" }}
func (s *Stencil) ApplyTemplate(name string) (string, error) {
	var buf bytes.Buffer
	err := s.Template.ExecuteTemplate(&buf, name, nil)
	return buf.String(), err
}

// Arg returns an argument from the ServiceManifest
//
//   {{- stencil.Arg "name" }}
func (s *Stencil) Arg(name string) interface{} {
	return s.m.Arguments[name]
}

// InstallFile changes the current active rendered file and writes
// the provided contents to it. This changes the scope of the
// current "File" being rendered.
//
//   {{- file.Skip "Virtual file that generates X files" }}
//   {{- define "command"}}
//   package main
//
//   import "fmt"
//
//   func main() {
//     fmt.Println("hello, world!")
//   }
//
//   {{- end }}
//
//
//   {{- /* Generate X number of files based on arguments.commands list of strings */}}
//   {{- range (stencil.Arg "commands") }}
//   {{- stencil.ApplyTemplate "command" | stencil.InstallFile (printf "cmd/%s.go" .) }}
//   {{- end }}
func (s *Stencil) InstallFile(name, contents string) string {
	rt := &RenderedTemplate{
		Path:   name,
		Reader: strings.NewReader(contents),
	}
	s.Files = append(s.Files, rt)
	s.File = rt
	return ""
}
