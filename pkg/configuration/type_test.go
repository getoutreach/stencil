// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the configuration pac

package configuration_test

import (
	"testing"

	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/pkg/configuration"
)

func TestTemplateRepositoryType(t *testing.T) {
	assert.NilError(t, nil)
	tests := []struct {
		Name           string
		In             string
		Contains       []configuration.TemplateRepositoryType
		DoesNotContain []configuration.TemplateRepositoryType
	}{
		{
			Name:           "empty",
			In:             "",
			Contains:       []configuration.TemplateRepositoryType{configuration.TemplateRepositoryTypeTemplates},
			DoesNotContain: []configuration.TemplateRepositoryType{configuration.TemplateRepositoryTypeExt},
		},
		{
			Name:           "templates",
			In:             "templates",
			Contains:       []configuration.TemplateRepositoryType{configuration.TemplateRepositoryTypeTemplates},
			DoesNotContain: []configuration.TemplateRepositoryType{configuration.TemplateRepositoryTypeExt},
		},
		{
			Name:           "extension",
			In:             "extension",
			Contains:       []configuration.TemplateRepositoryType{configuration.TemplateRepositoryTypeExt},
			DoesNotContain: []configuration.TemplateRepositoryType{configuration.TemplateRepositoryTypeTemplates},
		},
		{
			Name: "both",
			In:   "extension,templates",
			Contains: []configuration.TemplateRepositoryType{
				configuration.TemplateRepositoryTypeExt,
				configuration.TemplateRepositoryTypeTemplates,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			//t.Parallel()
			var ts configuration.TemplateRepositoryTypes
			err := yaml.Unmarshal([]byte(test.In), &ts)
			assert.NilError(t, err)
			for _, typ := range test.Contains {
				assert.Assert(t, ts.Contains(typ))
			}
			for _, typ := range test.DoesNotContain {
				assert.Assert(t, !ts.Contains(typ))
			}
			// rountrip marshal
			b, err := ts.MarshalYAML()
			assert.NilError(t, err)
			s, isString := b.(string)

			assert.Equal(t, true, isString)
			assert.Equal(t, test.In, s, "roundtrip marshal failed")
		})
	}
}
