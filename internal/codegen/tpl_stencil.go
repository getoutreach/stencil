// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"bytes"
	"fmt"
)

// TplStencil contains the global functions available to a template for
// interacting with stencil.
type TplStencil struct {
	// s is the underlying stencil object that this is attached to
	s *Stencil

	// t is the current template in the context of our render
	t *Template
}

// Arg returns the value of an argument in the service's
// manifest.
// Note: Only the top-level arguments are supported.
//
//   {{- stencil.Arg "name" }}
func (s *TplStencil) Arg(path string) interface{} {
	if path == "" {
		return s.Args()
	}

	return s.s.m.Arguments[path]
}

// Args returns all arguments passed to stencil from the service's
// manifest
//
//   {{- (stencil.Args).name }}
func (s *TplStencil) Args() map[string]interface{} {
	return s.s.m.Arguments
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
//   {{- stencil.ApplyTemplate "command" | file.SetContents }}
func (s *TplStencil) ApplyTemplate(name string, dataSli ...interface{}) (string, error) {
	// We check for dataSli here because we had to set it to a range of arguments
	// to allow it to be not set.
	if len(dataSli) > 1 {
		return "", fmt.Errorf("ApplyTemplate() only takes max two arguments, name and data")
	}

	var data interface{}
	if len(dataSli) == 1 {
		data = dataSli[0]
	}

	var buf bytes.Buffer
	err := s.t.Module.GetTemplate().ExecuteTemplate(&buf, name, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
