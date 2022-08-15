// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/getoutreach/stencil/internal/dotnotation"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
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

// GetModuleHook returns a module block in the scope of this module
//
// This is incredibly useful for allowing other modules to write
// to files that your module owns. Think of them as extension points
// for your module. The value returned by this function is always a
// []interface{}, aka a list.
//
//	{{- /* This returns a []interface{} */}}
//	{{ $hook := stencil.GetModuleHook "myModuleHook" }}
//	{{- range $hook }}
//	  {{ . }}
//	{{- end }}
func (s *TplStencil) GetModuleHook(name string) []interface{} {
	k := path.Join(s.t.Module.Name, name)
	v := s.s.sharedData[k]

	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		WithField("data", spew.Sdump(v)).Debug("getting module hook")
	return v
}

// AddToModuleHook adds to a hook in another module
//
// This functions write to module hook owned by another module for
// it to operate on. These are not strongly typed so it's best practice
// to look at how the owning module uses it for now. Module hooks must always
// be written to with a list to ensure that they can always be written to multiple
// times.
//
//	{{- /* This writes to a module hook */}}
//	{{ stencil.AddToModuleHook "github.com/myorg/repo" "myModuleHook" (list "myData") }}
func (s *TplStencil) AddToModuleHook(module, name string, data interface{}) (out, err error) {
	// Only modify on first pass
	if !s.s.isFirstPass {
		return nil, nil
	}

	// key is <module>/<name>
	k := path.Join(module, name)
	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		WithField("data", spew.Sdump(data)).Debug("adding to module hook")

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		err := fmt.Errorf("third parameter, data, must be set")
		return err, err
	}

	// we only allow slices or maps to allow multiple templates to
	// write to the same block
	if v.Kind() != reflect.Slice {
		err := fmt.Errorf("unsupported module block data type %q, supported type is slice", v.Kind())
		return err, err
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

	return nil, nil
}

// Arg returns the value of an argument in the service's manifest
//
//	{{- stencil.Arg "name" }}
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
	arg := mf.Arguments[pth]

	mapInf := make(map[interface{}]interface{})
	for k, v := range s.s.m.Arguments {
		mapInf[k] = v
	}

	// if not set then we return a default value based on the denoted type
	v, err := dotnotation.Get(mapInf, pth)
	if err != nil {
		if arg.Default != nil {
			return arg.Default, nil
		}

		if arg.Required {
			return nil, fmt.Errorf("module %q requires argument %q but is not set", s.t.Module.Name, pth)
		}

		// json schema convention is to define "type" as the top level key.
		typ, ok := arg.Schema["type"]
		if !ok {
			// fallback to the deprecated arg.Type
			typ = arg.Type //nolint:staticcheck // Why: Compat

			// If arg.Type isn't set then we have no type information
			// so return nothing. This is likely problematic so a linter
			// should warn on this.
			if arg.Type == "" { //nolint:staticcheck // Why: Compat
				return nil, nil
			}
		}
		typs, ok := typ.(string)
		if !ok {
			return nil, fmt.Errorf("module %q argument %q has invalid type: %v", s.t.Module.Name, pth, typ)
		}

		switch typs {
		case "map", "object":
			v = make(map[interface{}]interface{})
		case "list", "array":
			v = []interface{}{}
		case "boolean", "bool":
			v = false
		case "integer", "int", "number":
			v = 0
		case "string":
			v = ""
		default:
			return "", fmt.Errorf("module %q argument %q has invalid type %q", s.t.Module.Name, pth, typs)
		}
	}

	// validate the data
	if arg.Schema != nil {
		if err := s.validateArg(pth, &arg, v); err != nil {
			return nil, err
		}
	}

	return v, nil
}

// validateArg validates an argument against the schema
func (s *TplStencil) validateArg(pth string, arg *configuration.Argument, v interface{}) error {
	schemaBuf := new(bytes.Buffer)
	if err := json.NewEncoder(schemaBuf).Encode(arg.Schema); err != nil {
		return errors.Wrap(err, "failed to encode schema into JSON")
	}

	jsc := jsonschema.NewCompiler()
	jsc.Draft = jsonschema.Draft2020

	schemaURL := "manifest.yaml/arguments/" + pth
	if err := jsc.AddResource(schemaURL, schemaBuf); err != nil {
		return errors.Wrapf(err, "failed to add argument '%s' json schema to compiler", pth)
	}

	schema, err := jsc.Compile(schemaURL)
	if err != nil {
		return errors.Wrapf(err, "failed to compile argument '%s' schema", pth)
	}

	if err := schema.Validate(v); err != nil {
		var validationError *jsonschema.ValidationError
		if errors.As(err, &validationError) {
			// If there's only one error, return it directly, otherwise
			// return the full list of errors.
			errs := validationError.DetailedOutput().Errors
			out := ""
			if len(errs) == 1 {
				out = errs[0].Error
			} else {
				out = fmt.Sprintf("%#v", validationError.DetailedOutput().Errors)
			}

			return fmt.Errorf("module %q argument %q validation failed: %s", s.t.Module.Name, pth, out)
		}

		return errors.Wrapf(err, "module %q argument %q validation failed", s.t.Module.Name, pth)
	}

	return nil
}

// Deprecated: Use Arg instead.
// Args returns all arguments passed to stencil from the service's manifest
//
// Note: This doesn't set default values and is instead
// representative of _all_ data passed in its raw form.
//
//	{{- (stencil.Args).name }}
func (s *TplStencil) Args() map[string]interface{} {
	return s.s.m.Arguments
}

// ReadFile reads a file from the current directory and returns it's contents
//
//	{{ stencil.ReadFile "myfile.txt" }}
func (s *TplStencil) ReadFile(name string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	f, err := osfs.New(cwd).Open(name)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %q", name)
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// ApplyTemplate executes a template inside of the current module
//
// This function does not support rendering a template from another module.
//
//	{{- define "command"}}
//	package main
//
//	import "fmt"
//
//	func main() {
//	  fmt.Println("hello, world!")
//	}
//
//	{{- end }}
//
//	{{- stencil.ApplyTemplate "command" | file.SetContents }}
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
