// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"reflect"

	"github.com/getoutreach/stencil/internal/dotnotation"
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
//
//   {{- stencil.Arg "name" }}
func (s *TplStencil) Arg(pth string) (interface{}, error) {
	if pth == "" {
		return s.Args(), nil
	}

	mf, err := s.t.Module.Manifest(context.TODO())
	if err != nil {
		// In theory this should never happen because we've
		// already parsed the manifest. But, just in case
		// we handle this here.
		return nil, err
	}

	if _, ok := mf.Arguments[pth]; !ok {
		return "", fmt.Errorf("module %q doesn't list argument %q as an argument in it's manifest", s.t.Module.Name, pth)
	}

	mapInf := make(map[interface{}]interface{})
	for k, v := range s.s.m.Arguments {
		mapInf[k] = v
	}

	// if not set then we return a default value based on the denoted type
	v, err := dotnotation.Get(mapInf, pth)
	if err != nil {
		switch mf.Arguments[pth].Type {
		case "list":
			v = []string{}
		case "boolean":
			v = false
		case "string":
			v = ""
		default:
			return "", fmt.Errorf("module %q argument %q has invalid type %q", s.t.Module.Name, pth, mf.Arguments[pth].Type)
		}
	}

	return v, nil
}

// Args returns all arguments passed to stencil from the service's
// manifest. Note: This doesn't set default values and is instead
// representative of _all_ data passed in it's raw form.
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
