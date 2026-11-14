// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the FileResult, a file that a render
// produced.

package rendertest

import (
	"os"

	"github.com/getoutreach/stencil/internal/codegen"
)

// FileResult is a file that a render produced.
type FileResult struct {
	// path is the path of the file
	path string

	// contents are the rendered contents
	contents []byte

	// mode is the file mode
	mode os.FileMode

	// warnings are the warnings that the render added to the file
	warnings []string

	// blocks are the blocks in the rendered contents, by name. It is nil if
	// the contents have invalid blocks.
	blocks map[string]string
}

// newFileResult returns the FileResult of f.
func newFileResult(f *codegen.File) *FileResult {
	//nolint:errcheck // Why: Invalid blocks leave no blocks, which BlockValue reports.
	blocks, _ := codegen.ParseBlocks(f.Bytes())
	return &FileResult{
		path:     f.Name(),
		contents: f.Bytes(),
		mode:     f.Mode(),
		warnings: f.Warnings,
		blocks:   blocks,
	}
}

// Path returns the path of the file.
func (fr *FileResult) Path() string {
	return fr.path
}

// Contents returns the rendered contents of the file.
func (fr *FileResult) Contents() []byte {
	return fr.contents
}

// String returns the rendered contents of the file.
func (fr *FileResult) String() string {
	return string(fr.contents)
}

// Mode returns the mode of the file.
func (fr *FileResult) Mode() os.FileMode {
	return fr.mode
}

// Warnings returns the warnings that the render added to the file, such as
// deprecation notices.
func (fr *FileResult) Warnings() []string {
	return fr.warnings
}

// BlockValue returns the contents of the named block in the rendered
// contents, or "" if the block is not there.
func (fr *FileResult) BlockValue(name string) string {
	return fr.blocks[name]
}
