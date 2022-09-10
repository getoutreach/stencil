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
	assert.Assert(t, manifest.Type.Contains(configuration.TemplateRepositoryTypeTemplates), "failed to validate returned manifest")

	fs, err := m.GetFS(ctx)
	assert.NilError(t, err, "failed to call GetFS() on module")

	_, err = fs.Stat("manifest.yaml")
	assert.NilError(t, err, "failed to validate returned manifest from fs")
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

	mods, err := modules.GetModulesForService(context.Background(), "", sm)
	assert.NilError(t, err, "expected GetModulesForService() to not error")
	assert.Equal(t, len(mods), 1, "expected exactly one module to be returned")
	assert.Equal(t, mods[0].URI, sm.Replacements["github.com/getoutreach/stencil-base"],
		"expected module to use replacement URI")
}

func TestCanGetLatestVersion(t *testing.T) {
	ctx := context.Background()
	mods, err := modules.GetModulesForService(ctx, "", &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name: "github.com/getoutreach/stencil-base",
			},
		},
	})
	assert.NilError(t, err, "failed to call GetModulesForService()")
	assert.Equal(t, len(mods), 1, "expected exactly one module to be returned")
}

func TestHandleMultipleConstraints(t *testing.T) {
	ctx := context.Background()
	mods, err := modules.GetModulesForService(ctx, "", &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name:    "github.com/getoutreach/stencil-base",
				Version: "=<0.5.0",
			},
			{
				Name: "nested_constraint",
			},
		},
		Replacements: map[string]string{
			"nested_constraint": "file://testdata/nested_constraint",
		},
	})
	assert.NilError(t, err, "failed to call GetModulesForService()")
	assert.Equal(t, len(mods), 2, "expected exactly two modules to be returned")

	// should resolve to v0.3.2 because testdata wants latest patch of 0.3.0, while we want =<0.5.0
	// which is the latest patch of 0.3.0
	assert.Equal(t, mods[0].Version, "v0.3.2", "expected module to match")
}

func TestHandleNestedModules(t *testing.T) {
	ctx := context.Background()
	mods, err := modules.GetModulesForService(ctx, "", &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name: "a",
			},
		},
		Replacements: map[string]string{
			"a": "file://testdata/nested_modules/a",
			"b": "file://testdata/nested_modules/b",
		},
	})
	assert.NilError(t, err, "failed to call GetModulesForService()")
	assert.Equal(t, len(mods), 2, "expected exactly two modules to be returned")
	assert.Equal(t, mods[0].Name, "a", "expected module to match")
	assert.Equal(t, mods[1].Name, "b", "expected module to match")
}

func TestFailOnIncompatibleConstraints(t *testing.T) {
	ctx := context.Background()
	_, err := modules.GetModulesForService(ctx, "", &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name:    "github.com/getoutreach/stencil-base",
				Version: ">=0.5.0",
			},
			{
				// wants patch of 0.3.0
				Name: "nested_constraint",
			},
		},
		Replacements: map[string]string{
			"nested_constraint": "file://testdata/nested_constraint",
		},
	})
	assert.Error(t, err,
		//nolint:lll // Why: That's the error :(
		"failed to resolve module 'github.com/getoutreach/stencil-base' with constraints\n└─ testing-service (top-level) wants >=0.5.0\n  └─ nested_constraint wants ~0.3.0\n: no version found matching criteria",
		"expected GetModulesForService() to error")
}

func TestSupportChannelAndConstraint(t *testing.T) {
	ctx := context.Background()
	mods, err := modules.GetModulesForService(ctx, "", &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name:    "github.com/getoutreach/stencil-base",
				Channel: "rc",
				Version: "v0.6.0-rc.4",
			},
		},
	})
	assert.NilError(t, err, "failed to call GetModulesForService()")
	assert.Equal(t, len(mods), 1, "expected exactly one module to be returned")
	assert.Equal(t, mods[0].Version, "v0.6.0-rc.4", "expected module to match")
}

func TestCanUseBranch(t *testing.T) {
	ctx := context.Background()
	mods, err := modules.GetModulesForService(ctx, "", &configuration.ServiceManifest{
		Name: "testing-service",
		Modules: []*configuration.TemplateRepository{
			{
				Name:    "github.com/getoutreach/stencil-base",
				Channel: "main",
			},
		},
	})
	assert.NilError(t, err, "failed to call GetModulesForService()")
	assert.Equal(t, len(mods), 1, "expected exactly one module to be returned")
	assert.Equal(t, mods[0].Version, "main", "expected module to match")
}
