// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"bytes"
	"fmt"
	"path"
	"reflect"

	"github.com/imdario/mergo"
)

// TplStencil contains the global functions available to a template for
// interacting with stencil.
type TplStencil struct {
	// s is the underlying stencil object that this is attached to
	s *Stencil

	// t is the current template in the context of our render
	t *Template
}

// GetModuleHook returns a module block in the scope of this
// module
func (s *TplStencil) GetModuleHook(name string) interface{} {
	return s.s.sharedData[path.Join(s.t.Module.Name, name)]
}

// AddToModuleHook adds to a hook in another module
func (s *TplStencil) AddToModuleHook(module, name string, data interface{}) (string, error) {
	// Only modify on first pass
	if !s.s.isFirstPass {
		return "", nil
	}

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return "", fmt.Errorf("third parameter, data, must be set")
	}

	// we only allow slices or maps to allow multiple templates to
	// write to the same block
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Map {
		return "", fmt.Errorf("unsupported module block data type %q, supported types are slice and map", v.Kind())
	}

	k := path.Join(module, name)
	if _, ok := s.s.sharedData[k]; !ok {
		s.s.sharedData[k] = data
		return "", nil
	}

	return "", mergo.Merge(s.s.sharedData[k], data, mergo.WithAppendSlice)
}

// Arg returns the value of an argument in the service's
// manifest.
// Note: Only the top-level arguments are supported.
//
//   {{- stencil.Arg "name" }}
func (s *TplStencil) Arg(pth string) interface{} {
	if pth == "" {
		return s.Args()
	}

	return s.s.m.Arguments[pth]
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
