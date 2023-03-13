// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the public API for templates
// for stencil

package codegen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/pkg/errors"
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
	k := s.s.sharedData.key(s.t.Module.Name, name)
	v := s.s.sharedData.moduleHooks[k]

	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		WithField("data", spew.Sdump(v)).Debug("getting module hook")
	return v
}

// SetGlobal sets a global to be used in the context of the current template module
// repository. This is useful because sometimes you want to define variables inside
// of a helpers template file after doing manifest argument processing and then use
// them within one or more template files to be rendered; however, go templates limit
// the scope of symbols to the current template they are defined in, so this is not
// possible without external tooling like this function.
//
// This template function stores (and its inverse, GetGlobal, retrieves) data that is
// not strongly typed, so use this at your own risk and be averse to panics that could
// occur if you're using the data it returns in the wrong way.
//
//	{{- /* This writes a global into the current context of the template module repository */}}
//	{{ stencil.SetGlobal "IsGeorgeCool" true }}
func (s *TplStencil) SetGlobal(name string, data interface{}) error {
	// Only modify on first pass
	if !s.s.isFirstPass {
		return nil
	}

	k := s.s.sharedData.key(s.t.Module.Name, name)
	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		WithField("data", spew.Sdump(data)).Debug("adding to global store")

	s.s.sharedData.globals[k] = global{
		template: s.t.Path,
		value:    data,
	}

	return nil
}

// GetGlobal retrieves a global variable set by SetGlobal. The data returned from this function
// is unstructured so by averse to panics - look at where it was set to ensure you're dealing
// with the proper type of data that you think it is.
//
//	{{- /* This retrieves a global from the current context of the template module repository */}}
//	{{ $isGeorgeCool := stencil.GetGlobal "IsGeorgeCool" }}
func (s *TplStencil) GetGlobal(name string) interface{} {
	k := s.s.sharedData.key(s.t.Module.Name, name)

	if v, ok := s.s.sharedData.globals[k]; ok {
		s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
			WithField("data", spew.Sdump(v)).WithField("definingTemplate", v.template).
			Debug("retrieved data from global store")

		return v.value
	}

	s.log.WithField("template", s.t.ImportPath()).WithField("path", k).
		Warn("failed to retrieved data from global store")

	return nil
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

	k := s.s.sharedData.key(module, name)
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
	if _, ok := s.s.sharedData.moduleHooks[k]; ok {
		s.s.sharedData.moduleHooks[k] = append(s.s.sharedData.moduleHooks[k], interfaceSlice...)
	} else {
		s.s.sharedData.moduleHooks[k] = interfaceSlice
	}

	return nil, nil
}

// Deprecated: Use Arg instead.
// Args returns all arguments passed to stencil from the service's manifest
//
// Note: This doesn't set default values and is instead
// representative of _all_ data passed in its raw form.
//
// This is deprecated and will be removed in a future release.
//
//	{{- (stencil.Args).name }}
func (s *TplStencil) Args() map[string]interface{} {
	return s.s.m.Arguments
}

// ReadFile reads a file from the current directory and returns it's contents
//
//	{{ stencil.ReadFile "myfile.txt" }}
func (s *TplStencil) ReadFile(name string) (string, error) {
	f, ok := s.exists(name)
	if !ok {
		return "", errors.Errorf("file %q does not exist", name)
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// Exists returns true if the file exists in the current directory
//
//	{{- if stencil.Exists "myfile.txt" }}
//	{{ stencil.ReadFile "myfile.txt" }}
//	{{- end }}
func (s *TplStencil) Exists(name string) bool {
	f, ok := s.exists(name)
	if ok {
		f.Close() // close the file handle, since we don't need it
	}
	return ok
}

// exists returns a billy.File if the file exists, and true. If it doesn't,
// nil is returned and false.
func (s *TplStencil) exists(name string) (billy.File, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, false
	}

	f, err := osfs.New(cwd).Open(name)
	if err != nil {
		return nil, false
	}
	return f, true
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

// ReadBlocks parses a file and attempts to read the blocks from it, and their data.
//
// As a special case, if the file does not exist, an empty map is returned instead of an error.
//
// **NOTE**: This function does not guarantee that blocks are able to be read during runtime.
// for example, if you try to read the blocks of a file from another module there is no guarantee
// that that file will exist before you run this function. Nor is there the ability to tell stencil
// to do that (stencil does not have any order guarantees). Keep that in mind when using this function.
//
//	{{- $blocks := stencil.ReadBlocks "myfile.txt" }}
//	{{- range $name, $data := $blocks }}
//	  {{- $name }}
//	  {{- $data }}
//	{{- end }}
func (s *TplStencil) ReadBlocks(fpath string) (map[string]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// ensure that the file is within the current directory
	// and not attempting to escape it
	if _, err := osfs.New(cwd).Stat(fpath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}

		return nil, err
	}

	data, err := parseBlocks(fpath)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Debug logs the provided arguments under the DEBUG log level (must run stencil with --debug).
//
//  {{- $_ := stencil.Debug "I'm a log!" }}
func (s *TplStencil) Debug(args ...interface{}) error {
	s.log.WithField("path", s.t.Path).Debug(args...)

	// We have to return something...
	return nil
}
