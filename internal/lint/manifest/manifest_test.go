// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for stencil manifest loading and validation.

package manifest_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	lintmanifest "github.com/getoutreach/stencil/internal/lint/manifest"
)

func TestLoadValid(t *testing.T) {
	mf, strictErr, multiDoc, readErr := lintmanifest.Load(strings.NewReader("name: testing\n"))
	assert.NilError(t, readErr)
	assert.NilError(t, strictErr)
	assert.Equal(t, false, multiDoc)
	assert.Assert(t, mf != nil)
	assert.Equal(t, "testing", mf.Name)
}

func TestLoadUnknownKeyStrictFailsButLenientPopulates(t *testing.T) {
	// 'nme' is an unknown key: strict decode fails, but lenient decode still
	// populates the rest so field checks can run.
	mf, strictErr, _, readErr := lintmanifest.Load(strings.NewReader("name: testing\nnme: oops\n"))
	assert.NilError(t, readErr)
	assert.Assert(t, strictErr != nil)
	assert.Assert(t, mf != nil)
	assert.Equal(t, "testing", mf.Name)
}

func TestLoadNestedUnknownKey(t *testing.T) {
	// An unknown key inside an argument must also trip strict decoding.
	in := "name: testing\narguments:\n  foo:\n    scema: {}\n"
	_, strictErr, _, readErr := lintmanifest.Load(strings.NewReader(in))
	assert.NilError(t, readErr)
	assert.Assert(t, strictErr != nil)
}

func TestLoadEmptyInput(t *testing.T) {
	mf, strictErr, _, readErr := lintmanifest.Load(strings.NewReader("   \n# just a comment\n"))
	assert.NilError(t, readErr)
	assert.Assert(t, strictErr != nil) // io.EOF
	assert.Assert(t, mf == nil)
}

func TestLoadMultiDocument(t *testing.T) {
	mf, strictErr, multiDoc, readErr := lintmanifest.Load(
		strings.NewReader("name: testing\n---\nname: second\n"))
	assert.NilError(t, readErr)
	assert.NilError(t, strictErr)
	assert.Assert(t, mf != nil)
	assert.Equal(t, "testing", mf.Name) // only doc 1 is read
	assert.Equal(t, true, multiDoc)
}
