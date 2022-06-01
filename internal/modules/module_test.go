// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the modules package

package modules_test

import (
	"context"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"gotest.tools/v3/assert"
)

func TestCanFetchModule(t *testing.T) {
	ctx := context.Background()
	m, err := modules.New(ctx, "", &configuration.TemplateRepository{Name: "github.com/getoutreach/stencil-base", Version: "main"})
	assert.NilError(t, err, "failed to call New()")

	manifest, err := m.Manifest(ctx)
	assert.NilError(t, err, "failed to call Manifest() on module")
	assert.Equal(t, manifest.Type, configuration.TemplateRepositoryTypeStd, "failed to validate returned manifest")

	fs, err := m.GetFS(ctx)
	assert.NilError(t, err, "failed to call GetFS() on module")

	_, err = fs.Stat("manifest.yaml")
	assert.NilError(t, err, "failed to validate returned manifest from fs")
}

func TestCanGetLatestModule(t *testing.T) {
	_, err := modules.New(context.Background(), "", &configuration.TemplateRepository{Name: "github.com/getoutreach/stencil-base"})
	assert.NilError(t, err, "failed to call New()")
}

func TestReplacementLocalModule(t *testing.T) {
	sm := &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name: "github.com/getoutreach/stencil-base",
			},
		},
		Replacements: map[string]string{
			"github.com/getoutreach/stencil-base": "file://testdata",
		},
	}

	mods, err := modules.GetModulesForService(context.Background(), sm)
	assert.NilError(t, err, "expected GetModulesForService() to not error")
	assert.Equal(t, len(mods), 1, "expected exactly one module to be returned")
	assert.Equal(t, mods[0].URI, sm.Replacements["github.com/getoutreach/stencil-base"],
		"expected module to use replacement URI")
}

func TestCanFetchDeprecatedModule(t *testing.T) {
	sm := &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				URL: "https://github.com/getoutreach/stencil-base",
			},
		},
	}

	mods, err := modules.GetModulesForService(context.Background(), sm)
	assert.NilError(t, err, "expected GetModulesForService() to not error")
	assert.Equal(t, len(mods), 1, "expected exactly one module to be returned")
	assert.Equal(t, mods[0].Name, "github.com/getoutreach/stencil-base")
}
