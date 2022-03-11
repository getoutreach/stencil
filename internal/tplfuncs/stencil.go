// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package tplfuncs

import (
	"bytes"

	"github.com/getoutreach/stencil/internal/functions"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// Stencil contains the global functions available to a template for
// interacting with stencil.
type Stencil struct {
	s *functions.Stencil
	m *configuration.ServiceManifest
}

// Arg returns the value of an argument in the service's
// manifest.
// Note: Only the top-level arguments are supported.
//
//   {{- stencil.Arg "name" }}
func (s *Stencil) Arg(path string) interface{} {
	if path == "" {
		return s.Args()
	}

	return s.m.Arguments[path]
}

// Args returns all arguments passed to stencil from the service's
// manifest
//
//   {{- (stencil.Args).name }}
func (s *Stencil) Args() map[string]interface{} {
	return s.m.Arguments
}

// ApplyTemplate executes a template inside of the current module
// that belongs to the actively rendered template. It does not
// support rendering a template from another module.
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
	err := s.s.Template.Module.GetTemplate().ExecuteTemplate(&buf, name, nil)
	return buf.String(), err
}
