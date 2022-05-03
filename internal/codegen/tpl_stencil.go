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

	"github.com/davecgh/go-spew/spew"
	"github.com/getoutreach/stencil/internal/dotnotation"
	"github.com/sirupsen/logrus"
)

// TplStencil contains the global functions available to a template for
// interacting with stencil.
type TplStencil struct {
	// s is the underlying stencil object that this is attached to
	s *Stencil

	// t is the current template in the context of our render
	t *Template

	log logrus.FieldLogger
}

// GetModuleHook returns a module block in the scope of this
// module
func (s *TplStencil) GetModuleHook(name string) []interface{} {
	k := path.Join(s.t.Module.Name, name)
	v := s.s.sharedData[k]

	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		WithField("data", spew.Sdump(v)).Debug("getting module hook")
	return v
}

// AddToModuleHook adds to a hook in another module
func (s *TplStencil) AddToModuleHook(module, name string, data interface{}) error {
	// Only modify on first pass
	if !s.s.isFirstPass {
		return nil
	}

	// key is <module>/<name>
	k := path.Join(module, name)
	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		WithField("data", spew.Sdump(data)).Debug("adding to module hook")

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return fmt.Errorf("third parameter, data, must be set")
	}

	// we only allow slices or maps to allow multiple templates to
	// write to the same block
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("unsupported module block data type %q, supported type is slice", v.Kind())
	}

	// convert the slice into a []interface{}
	interfaceSlice := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		interfaceSlice[i] = v.Index(i).Interface()
	}

	// if set, append, otherwise assign
	if _, ok := s.s.sharedData[k]; ok {
		s.s.sharedData[k] = append(s.s.sharedData[k], interfaceSlice...)
	} else {
		s.s.sharedData[k] = interfaceSlice
	}

	return nil
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
		return "", fmt.Errorf("module %q doesn't list argument %q as an argument in its manifest", s.t.Module.Name, pth)
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
			v = []interface{}{}
		case "boolean", "bool":
			v = false
		case "integer", "int":
			v = 0
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
// representative of _all_ data passed in its raw form.
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
	} else {
		// If no data was passed, pass through the values of the parent template
		data = s.t.args
	}

	var buf bytes.Buffer
	if err := s.t.Module.GetTemplate().ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
