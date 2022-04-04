package stenciltest

import (
	"testing"

	"github.com/getoutreach/stencil/pkg/configuration"
)

func TestMain(t *testing.T) {
	st := &Template{
		path:                "testdata/test.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             false,
	}
	st.Run(false)
}

func TestErrorHandling(t *testing.T) {
	st := &Template{
		path:                "testdata/error.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             false,
	}
	st.ErrorContains("sad")
	st.Run(false)

	st = &Template{
		path:                "testdata/error.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             false,
	}
	st.ErrorContains("sad pikachu")
	st.Run(false)
}
