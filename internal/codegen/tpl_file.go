// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements the file struct passed to
// templates in Stencil.

package codegen

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// TplFile is the current file we're writing output to in a
// template. This can be changed via file.SetPath and written
// to by file.Install. When a template does not call file.SetPath
// a default file is created that matches the current template path
// with the extension '.tpl' removed from the path and operated on.
type TplFile struct {
	// f is the current file we're writing to
	f *File

	// t is the current template
	t *Template

	// log is the logger to use for debugging
	log logrus.FieldLogger
}

// Block returns the contents of a given block
//   ###Block(name)
//   Hello, world!
//   ###EndBlock(name)
//
//   ###Block(name)
//   {{- /* Only output if the block is set */}}
//   {{- if not (empty (file.Block "name")) }}
//   {{ file.Block "name" }}
//   {{- end }}
//   ###EndBlock(name)
//
//   ###Block(name)
//   {{ - /* Short hand syntax, but adds newline if no contents */}}
//   {{ file.Block "name" }}
//   ###EndBlock(name)
func (f *TplFile) Block(name string) string {
	return f.f.Block(name)
}

// SetPath changes the path of the current file
func (f *TplFile) SetPath(path string) error {
	return f.f.SetPath(path)
}

// SetContents sets the contents of the current file
// to the provided string.
func (f *TplFile) SetContents(contents string) error {
	f.f.SetContents(contents)
	return nil
}

// Skip skips the current file
func (f *TplFile) Skip(_ string) error {
	f.f.Skipped = true
	return nil
}

// Delete deletes the current file
func (f *TplFile) Delete() error {
	f.f.Deleted = true
	return nil
}

// Static marks the current file as static
func (f *TplFile) Static() error {
	// if the file already exists, skip it
	if _, err := os.Stat(f.f.path); err == nil {
		return f.Skip("Static file, output already exists")
	}

	return nil
}

// Path returns the current path of the file we're
// writing to.
//
//   {{ file.Path }}
func (f *TplFile) Path() string {
	return f.f.path
}

// Create creates a new file that is rendered by the current
// template. If the template has a single file with no contents
// this file replaces it.
//
//   {{- define "command" }}
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
//   # Generate a "<commandName>.go" file for each command in .arguments.commands
//   {{- range $_, $commandName := (stencil.Arg "commands") }}
//   {{- file.Create (printf "cmd/%s.go" $commandName) 0600 now }}
//   {{- stencil.ApplyTemplate "command" | file.SetContents }}
//   {{- end }}
func (f *TplFile) Create(path string, mode os.FileMode, modTime time.Time) error {
	var err error
	f.f, err = NewFile(path, mode, modTime)
	if err != nil {
		return err
	}

	f.t.Files = append(f.t.Files, f.f)
	return nil
}