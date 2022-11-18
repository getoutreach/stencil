package stenciltest

import (
	"testing"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestMain(t *testing.T) {
	st := &Template{
		path:                "testdata/test.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             false,
		log:                 logrus.New(),
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
		log:                 logrus.New(),
	}
	st.ErrorContains("sad")
	st.Run(false)

	st = &Template{
		path:                "testdata/error.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             false,
		log:                 logrus.New(),
	}
	st.ErrorContains("sad pikachu")
	st.Run(false)
}

func TestArgs(t *testing.T) {
	st := &Template{
		path:                "testdata/args.tpl",
		additionalTemplates: make([]string, 0),
		m: &configuration.TemplateRepositoryManifest{Name: "testing", Arguments: map[string]configuration.Argument{
			"hello": {
				Type: "string",
			},
		}},
		t:       t,
		persist: false,
		log:     logrus.New(),
	}
	st.Args(map[string]interface{}{"hello": "world"})
	st.Run(false)
}

func TestGetTemplateRepositoryNames(t *testing.T) {
	trs := []*configuration.TemplateRepository{
		{
			Name:    "test1",
			Version: "test",
		},
		{
			Name: "test2",
		},
	}

	result := getTemplateRepositoryNames(trs)

	assert.DeepEqual(t, result, []string{"test1", "test2"})
}

// Doing this just to bump up coverage numbers, we essentially test this w/ the Template
// constructors in each test.
func TestCoverageHack(t *testing.T) {
	st := New(t, "testdata/test.tpl")
	assert.Equal(t, st.path, "testdata/test.tpl")
	assert.Equal(t, st.persist, true)
	assert.Assert(t, !cmp.Equal(st.t, nil))
	assert.Equal(t, st.m.Name, "testing")
}
