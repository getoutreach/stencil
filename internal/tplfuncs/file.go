// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements the file struct passed to
// templates in Stencil.

package tplfuncs

import (
	"os"
	"time"

	"github.com/getoutreach/stencil/internal/functions"
)

// File is the current file we're writing output to in a
// template. This can be changed via file.SetPath and written
// to by file.Install. When a template does not call file.SetPath
// a default file is created that matches the current template path
// with the extension '.tpl' removed from the path and operated on.
type File struct {
	// f is the current file we're writing to
	f *functions.File

	// t is the current template
	t *functions.Template
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
func (f *File) Block(name string) string {
	return f.f.Block(name)
}

// SetPath changes the path of the current file
func (f *File) SetPath(path string) error {
	f.f.SetPath(path)
	return nil
}

// SetContents sets the contents of the current file
// to the provided string.
func (f *File) SetContents(contents string) error {
	f.f.SetContents(contents)
	return nil
}

// Skip skips the current file
func (f *File) Skip(_ string) error {
	f.f.Skipped = true
	return nil
}

// Delete deletes the current file
func (f *File) Delete() error {
	f.f.Deleted = true
	return nil
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
//   {{- file.Create (prinf "cmd/%s.go" $commandName) }}
//   {{- stencil.ApplyTemplate "commands" | file.SetContents }}
//   {{- end }}
func (f *File) Create(path string, mode os.FileMode, modTime time.Time) error {
	var err error
	f.f, err = functions.NewFile(path, mode, modTime)
	if err != nil {
		return err
	}

	// If we have a single file with zero contents, replace it
	if len(f.t.Files) == 1 && len(f.t.Files[0].Bytes()) == 0 {
		f.t.Files = []*functions.File{f.f}
		return nil
	}

	f.t.Files = append(f.t.Files, f.f)
	return nil
}
