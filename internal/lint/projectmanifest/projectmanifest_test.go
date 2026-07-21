// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for project manifest (service.yaml) loading and validation.

package projectmanifest_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	projectmanifest "github.com/getoutreach/stencil/internal/lint/projectmanifest"
)

func TestLoadValid(t *testing.T) {
	res, err := projectmanifest.Load(strings.NewReader("name: my-service\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "my-service", res.Manifest.Name)
	assert.Assert(t, res.Root != nil)
	assert.Equal(t, false, res.MultiDoc)
	assert.NilError(t, res.DecodeErr)
}

func TestLoadEmptyInput(t *testing.T) {
	res, err := projectmanifest.Load(strings.NewReader("   \n# just a comment\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest == nil)
	assert.Assert(t, res.DecodeErr != nil) // io.EOF
}

func TestLoadMalformed(t *testing.T) {
	res, err := projectmanifest.Load(strings.NewReader("name: [unterminated\n"))
	assert.NilError(t, err) // read succeeded; decode failure is in DecodeErr
	assert.Assert(t, res.Manifest == nil)
	assert.Assert(t, res.DecodeErr != nil)
}

func TestLoadNonMapping(t *testing.T) {
	// A top-level scalar/sequence is not a mapping: Manifest nil, DecodeErr set.
	res, err := projectmanifest.Load(strings.NewReader("- a\n- b\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest == nil)
	assert.Assert(t, res.DecodeErr != nil)
}

func TestLoadMultiDocument(t *testing.T) {
	res, err := projectmanifest.Load(
		strings.NewReader("name: my-service\n---\nname: second\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "my-service", res.Manifest.Name) // only doc 1
	assert.Equal(t, true, res.MultiDoc)
}
