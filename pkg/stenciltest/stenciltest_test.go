package stenciltest

import (
	"fmt"
	"testing"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/getoutreach/stencil/pkg/stenciltest/config"
	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
)

var fal = false

func TestMain(t *testing.T) {
	st := &Template{
		path:                "testdata/test.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
	}
	st.Run(false)
}

func TestErrorHandling(t *testing.T) {
	st := &Template{
		path:                "testdata/error.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
	}
	st.ErrorContains("sad")
	st.Run(false)

	st = &Template{
		path:                "testdata/error.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
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
		persist: &fal,
	}
	st.Args(map[string]interface{}{"hello": "world"})
	st.Run(false)
}

func TestGoValidator(t *testing.T) {
	st := &Template{
		path:                "testdata/validator.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
	}
	st.Validator(config.NewGoValidator(func(contents string) error {
		if contents != "hello" {
			return fmt.Errorf("expected hello, got %s", contents)
		}

		// matched
		return nil
	}))
	st.Run(false)
}

func TestGoValidatorError(t *testing.T) {
	st := &Template{
		path:                "testdata/validator.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
	}
	st.Validator(config.NewGoValidator(func(contents string) error {
		if contents != "dontmatch" {
			return fmt.Errorf("expected dontmatch, got %s", contents)
		}

		// matched
		return nil
	}))
	st.ErrorContains("expected dontmatch, got hello")
	st.Run(false)
}

func TestCommandValidator(t *testing.T) {
	st := &Template{
		path:                "testdata/validator.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
	}
	st.Validator(config.Validator{
		Command: "./pkg/stenciltest/testdata/cmp.sh validator.tpl",
	})
	st.Run(false)
}

func TestCommandValidatorError(t *testing.T) {
	st := &Template{
		path:                "testdata/validator.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
	}
	st.Validator(config.Validator{
		Command: "./pkg/stenciltest/testdata/cmp.sh args.tpl",
	})
	st.ErrorContains("exit status 1")
	st.Run(false)
}

func TestInProcExtension(t *testing.T) {
	st := &Template{
		path:                "testdata/inproc.tpl",
		additionalTemplates: make([]string, 0),
		m:                   &configuration.TemplateRepositoryManifest{Name: "testing"},
		t:                   t,
		persist:             &fal,
		exts:                make(map[string]apiv1.Implementation),
	}
	st.Ext("inproc", &apiv1.EchoExtension{})
	st.Validator(config.NewGoValidator(func(contents string) error {
		if contents != "true" {
			return fmt.Errorf("expected true, got %q", contents)
		}

		// matched
		return nil
	}))
	st.Run(false)
}

// Doing this just to bump up coverage numbers, we essentially test this w/ the Template
// constructors in each test.
func TestCoverageHack(t *testing.T) {
	st := New(t, "testdata/test.tpl")
	assert.Equal(t, st.path, "testdata/test.tpl")
	assert.Assert(t, !cmp.Equal(st.t, nil))
	assert.Equal(t, st.m.Name, "testing")
}
