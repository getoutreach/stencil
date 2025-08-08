// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the template file

package codegen

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "embed"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

//go:embed testdata/multi-file.tpl
var multiFileTemplate string

//go:embed testdata/multi-file-input.tpl
var multiFileInputTemplate string

//go:embed testdata/apply-template-passthrough.tpl
var applyTemplatePassthroughTemplate string

//go:embed testdata/generated-block/template.txt.tpl
var generatedBlockTemplate string

//go:embed testdata/generated-block/fake.txt
var fakeGeneratedBlockFile string

func TestSingleFileRender(t *testing.T) {
	m := modules.NewWithFS(context.Background(), "testing", memfs.New())

	log := logrus.New()
	tpl, err := NewTemplate(m, "virtual-file.tpl", 0o644, time.Now(), []byte("hello world!"), log)
	assert.NilError(t, err, "failed to create basic template")

	sm := &configuration.ServiceManifest{Name: "testing"}

	st := NewStencil(sm, []*modules.Module{m}, log)
	err = tpl.Render(st, NewValues(context.Background(), sm, nil, log))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, tpl.Files[0].String(), "hello world!", "expected Render() to modify first created file")
}

func TestMultiFileRender(t *testing.T) {
	fs := createFakeModuleFSWithManifest(t, "name: testing\narguments:\n  commands:\n    type: list")

	m := modules.NewWithFS(context.Background(), "testing", fs)

	log := logrus.New()
	tpl, err := NewTemplate(m, "multi-file.tpl", 0o644,
		time.Now(), []byte(multiFileTemplate), log)
	assert.NilError(t, err, "failed to create template")

	sm := &configuration.ServiceManifest{Name: "testing", Arguments: map[string]interface{}{
		"commands": []string{"hello", "world", "command"},
	}}

	st := NewStencil(sm, []*modules.Module{m}, log)
	err = tpl.Render(st, NewValues(context.Background(), sm, nil, log))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpl.Files), 3, "expected Render() to create 3 files")

	for i, f := range tpl.Files {
		assert.Equal(t, f.String(), "command", "rendered template %d contents differred", i)
	}
}

func TestMultiFileWithInputRender(t *testing.T) {
	log := logrus.New()
	fs := createFakeModuleFSWithManifest(t, "name: testing\narguments:\n  commands:\n    type: list")
	m := modules.NewWithFS(context.Background(), "testing", fs)
	tpl, err := NewTemplate(m, "multi-file-input.tpl", 0o644,
		time.Now(), []byte(multiFileInputTemplate), log)
	assert.NilError(t, err, "failed to create template")

	sm := &configuration.ServiceManifest{Name: "testing", Arguments: map[string]interface{}{
		"commands": []string{"hello", "world", "command"},
	}}

	st := NewStencil(sm, []*modules.Module{m}, log)
	err = tpl.Render(st, NewValues(context.Background(), sm, nil, log))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpl.Files), 3, "expected Render() to create 3 files")

	for i, f := range tpl.Files {
		assert.Equal(t, (sm.Arguments["commands"].([]string))[i], f.String(), "rendered template %d contents differred", i)
	}
}

func TestApplyTemplateArgumentPassthrough(t *testing.T) {
	fs := createFakeModuleFSWithManifest(t, "name: testing\narguments:\n  commands:\n    type: list")

	m := modules.NewWithFS(context.Background(), "testing", fs)

	log := logrus.New()
	tpl, err := NewTemplate(m, "apply-template-passthrough.tpl", 0o644,
		time.Now(), []byte(applyTemplatePassthroughTemplate), log)
	assert.NilError(t, err, "failed to create template")

	sm := &configuration.ServiceManifest{Name: "testing", Arguments: map[string]interface{}{
		"commands": []string{"hello", "world", "command"},
	}}

	st := NewStencil(sm, []*modules.Module{m}, log)
	err = tpl.Render(st, NewValues(context.Background(), sm, nil, log))
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpl.Files), 1, "expected Render() to create 1 files")

	assert.Equal(t, "testing", tpl.Files[0].String(), "rendered template contents differed")
}

func TestGeneratedBlock(t *testing.T) {
	tempDir := t.TempDir()
	fakeFilePath := filepath.Join(tempDir, "generated-block.txt")
	fs := createFakeModuleFSWithManifest(t, "name: testing\n")
	sm := &configuration.ServiceManifest{Name: "testing", Arguments: map[string]interface{}{}}
	m := modules.NewWithFS(context.Background(), "testing", fs)

	log := logrus.New()
	st := NewStencil(sm, []*modules.Module{m}, log)
	assert.NilError(t, os.WriteFile(fakeFilePath, []byte(fakeGeneratedBlockFile), 0o644),
		"failed to write generated file")

	tpl, err := NewTemplate(m, "generated-block/template.tpl", 0o644,
		time.Now(), []byte(generatedBlockTemplate), log)
	assert.NilError(t, err, "failed to create template")

	tplf, err := NewFile(fakeFilePath, 0o644, time.Now())
	assert.NilError(t, err, "failed to create file")

	// Add the file (fake) to the template so that the template uses it for blocks
	tpl.Files = []*File{tplf}
	tpl.Render(st, NewValues(context.Background(), sm, nil, log))

	assert.Equal(t, tpl.Files[0].String(), fakeGeneratedBlockFile, "expected fake to equal rendered output")
}
