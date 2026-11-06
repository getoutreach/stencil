// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for retrying the resolution of a module version.

package modules_test

import (
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"gotest.tools/v3/assert"
)

// resolveExample resolves the module "example", which srv serves, at version.
func resolveExample(t *testing.T, srv *gitServer, version string) ([]*modules.Module, error) {
	t.Helper()

	// Keep the module and version caches of this test apart from the others.
	t.Setenv("TMPDIR", t.TempDir())

	return modules.GetModulesForService(t.Context(), &modules.ModuleResolveOptions{
		ServiceManifest: &configuration.ServiceManifest{
			Name:         "testing-service",
			Modules:      []*configuration.TemplateRepository{{Name: "example", Version: version}},
			Replacements: map[string]string{"example": srv.URL},
		},
		Log: newLogger(),
	})
}

func TestGetModulesForService_RetriesTransientResolveFailure(t *testing.T) {
	srv := newGitServer(t, 1)

	mods, err := resolveExample(t, srv, "")
	assert.NilError(t, err, "expected the second attempt to work")
	assert.Equal(t, len(mods), 1)
	assert.Equal(t, mods[0].Version, "v1.0.0")
}

func TestGetModulesForService_DoesNotRetryUnsatisfiedConstraint(t *testing.T) {
	srv := newGitServer(t, 0)

	_, err := resolveExample(t, srv, "^2.0.0")
	assert.ErrorContains(t, err, "no version found matching criteria")

	assert.Equal(t, srv.infoRefs.Load(), int32(1), "expected no retry, because asking again gives the same answer")
}
