// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the template file

package codegen

import (
	"context"
	"testing"
	"time"

	_ "embed"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5/memfs"
	"gotest.tools/v3/assert"
)

//go:embed testdata/multi-file.tpl
var multiFileTemplate string

//go:embed testdata/multi-file-input.tpl
var multiFileInputTemplate string

func TestSingleFileRender(t *testing.T) {
	m := modules.NewWithFS(context.Background(), "testing", memfs.New())

	tpl, err := NewTemplate(m, "virtual-file.tpl", 0o644, time.Now(), []byte("hello world!"))
	assert.NilError(t, err, "failed to create basic template")
	assert.Equal(t, len(tpl.Files), 1, "expected NewTemplate() to create first file")

	sm := &configuration.ServiceManifest{Name: "testing"}

	st := NewStencil(sm, []*modules.Module{m})
	err = tpl.Render(st, NewValues(context.Background(), sm))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, tpl.Files[0].String(), "hello world!", "expected Render() to modify first created file")
}

func TestMultiFileRender(t *testing.T) {
	m := modules.NewWithFS(context.Background(), "testing", memfs.New())

	tpl, err := NewTemplate(m, "multi-file.tpl", 0o644, time.Now(), []byte(multiFileTemplate))
	assert.NilError(t, err, "failed to create template")

	sm := &configuration.ServiceManifest{Name: "testing", Arguments: map[string]interface{}{
		"commands": []string{"hello", "world", "command"},
	}}

	st := NewStencil(sm, []*modules.Module{m})
	err = tpl.Render(st, NewValues(context.Background(), sm))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpl.Files), 3, "expected Render() to create 3 files")

	for i, f := range tpl.Files {
		assert.Equal(t, f.String(), "command", "rendered template %d contents differred", i)
	}
}

func TestMultiFileWithInputRender(t *testing.T) {
	m := modules.NewWithFS(context.Background(), "testing", memfs.New())

	tpl, err := NewTemplate(m, "multi-file-input.tpl", 0o644, time.Now(), []byte(multiFileInputTemplate))
	assert.NilError(t, err, "failed to create template")

	sm := &configuration.ServiceManifest{Name: "testing", Arguments: map[string]interface{}{
		"commands": []string{"hello", "world", "command"},
	}}

	st := NewStencil(sm, []*modules.Module{m})
	err = tpl.Render(st, NewValues(context.Background(), sm))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpl.Files), 3, "expected Render() to create 3 files")

	for i, f := range tpl.Files {
		assert.Equal(t, (sm.Arguments["commands"].([]string))[i], f.String(), "rendered template %d contents differred", i)
	}
}
