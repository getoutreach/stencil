package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/getoutreach/stencil/pkg/configuration"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

func TestUpdateModule(t *testing.T) {
	tt := []struct {
		Name                      string
		RemoveNativeExtensionFlag bool
		Given                     configuration.ServiceManifest
		Expected                  configuration.ServiceManifest
	}{
		{
			Name:                      "EnsureServiceNoChangeWithoutFlag",
			RemoveNativeExtensionFlag: true,
			Given: configuration.ServiceManifest{
				Name: "test",
				Modules: []*configuration.TemplateRepository{
					{
						Name:       "github.com/getoutreach/stencil-template-base",
						Prerelease: false,
					},
				},
				Arguments: map[string]interface{}{
					"description": "test module update",
					"releaseOptions": map[string]bool{
						"enablePrereleases": true,
					},
					"reportingTeam": "test_name",
				},
			},
			Expected: configuration.ServiceManifest{
				Name: "test",
				Modules: []*configuration.TemplateRepository{
					{
						Name:       "github.com/getoutreach/stencil-template-base",
						Prerelease: false,
					},
				},
				Arguments: map[string]interface{}{
					"description": "test module update",
					"releaseOptions": map[string]bool{
						"enablePrereleases": true,
					},
					"reportingTeam": "test_name",
				},
			},
		}, {
			Name:                      "EnsureNativeExtensionAddition",
			RemoveNativeExtensionFlag: false,
			Given: configuration.ServiceManifest{
				Name: "test",
				Modules: []*configuration.TemplateRepository{
					{
						Name:       "github.com/getoutreach/stencil-template-base",
						Prerelease: false,
					},
				},
				Arguments: map[string]interface{}{
					"description": "test module update",
					"releaseOptions": map[string]bool{
						"enablePrereleases": true,
					},
					"reportingTeam": "test_name",
				},
			},
			Expected: configuration.ServiceManifest{
				Name: "test",
				Modules: []*configuration.TemplateRepository{
					{
						Name:       "github.com/getoutreach/stencil-template-base",
						Prerelease: false,
					},
				},
				Arguments: map[string]interface{}{
					"description": "test module update",
					"releaseOptions": map[string]bool{
						"enablePrereleases": true,
						"force":             true,
					},
					"plugin":        true,
					"reportingTeam": "test_name",
				},
			},
		}, {
			Name:                      "EnsureNativeExtensionReversion",
			RemoveNativeExtensionFlag: true,
			Given: configuration.ServiceManifest{
				Name: "test",
				Modules: []*configuration.TemplateRepository{
					{
						Name:       "github.com/getoutreach/stencil-template-base",
						Prerelease: false,
					},
				},
				Arguments: map[string]interface{}{
					"description": "test module update",
					"releaseOptions": map[string]bool{
						"enablePrereleases": true,
						"force":             true,
					},
					"plugin":        true,
					"reportingTeam": "test_name",
				},
			},
			Expected: configuration.ServiceManifest{
				Name: "test",
				Modules: []*configuration.TemplateRepository{
					{
						Name:       "github.com/getoutreach/stencil-template-base",
						Prerelease: false,
					},
				},
				Arguments: map[string]interface{}{
					"description": "test module update",
					"releaseOptions": map[string]bool{
						"enablePrereleases": true,
					},
					"reportingTeam": "test_name",
				},
			},
		},
	}

	for _, test := range tt {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			var tm = &configuration.ServiceManifest{}
			var comp = &configuration.ServiceManifest{}

			// Create temporary service.yaml with valid values
			tempFile := filepath.Join(t.TempDir(), "service.yaml")
			b, err := yaml.Marshal(test.Given)
			assert.NilError(t, err, "failed to marshal given yaml")
			assert.NilError(t, os.WriteFile(tempFile, b, 0o777), "failed to write file")

			// Setup expected values
			b, err = yaml.Marshal(test.Expected)
			assert.NilError(t, err, "failed to marshal expected yaml")
			err = yaml.Unmarshal(b, tm)
			assert.NilError(t, err, "failed to unmarshal expected yaml")

			// Update the service.yaml and compare to expected
			err = readAndMergeServiceYaml(tempFile, test.RemoveNativeExtensionFlag)
			assert.NilError(t, err, "failed to read and update service.yaml")

			b, err = os.ReadFile(tempFile)
			assert.NilError(t, err, "failed to read service.yaml")

			err = yaml.Unmarshal(b, comp)
			assert.NilError(t, err, "failed to unmarshal service.yaml")

			assert.DeepEqual(t, tm, comp)
		})
	}
}
