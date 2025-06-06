// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements the stenciltest framework
// for testing templates generated by stencil.

// Package stenciltest contains code for testing templates
package stenciltest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"github.com/getoutreach/gobox/pkg/cli/github"
	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"gotest.tools/v3/assert"
)

// Template is a template that is being tested by the stenciltest framework.
type Template struct {
	// path is the path to the template.
	path string

	// aditionalTemplates is a list of additional templates to add to the renderer,
	// but not to snapshot.
	additionalTemplates []string

	// m is the template repository manifest for this test
	m *configuration.TemplateRepositoryManifest

	// t is a testing object.
	t *testing.T

	// args are the arguments to the template.
	args map[string]interface{}

	// mods are the modules passed to the template's service manifest.
	mods []*configuration.TemplateRepository

	// exts holds the inproc extensions
	exts map[string]apiv1.Implementation

	// errStr is the string an error should contain, if this is set then the template
	// MUST error.
	errStr string

	// log is the logger to use when running stencil
	log logrus.FieldLogger

	// persist denotes if we should save a snapshot or not
	// This is meant for tests.
	persist bool
}

// New creates a new test for a given template.
func New(t *testing.T, templatePath string, additionalTemplates ...string) *Template {
	// GOMOD: <module path>/go.mod
	b, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("failed to determine path to manifest: %v", err)
	}
	basepath := strings.TrimSuffix(strings.TrimSpace(string(b)), "/go.mod")

	b, err = os.ReadFile(filepath.Join(basepath, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	var m configuration.TemplateRepositoryManifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}

	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)

	return &Template{
		t:                   t,
		m:                   &m,
		path:                templatePath,
		additionalTemplates: additionalTemplates,
		persist:             true,
		log:                 log,
		mods:                []*configuration.TemplateRepository{},
		exts:                map[string]apiv1.Implementation{},
	}
}

// Args sets the arguments to the template.
func (t *Template) Args(args map[string]interface{}) *Template {
	t.args = args
	return t
}

// AddModule adds a module to the service manifest modules list.
func (t *Template) AddModule(tr *configuration.TemplateRepository) *Template {
	t.mods = append(t.mods, tr)
	return t
}

// Ext registers an in-proc extension with the current stencil template. The stenciltest library
// does not load the real extensions (because extensions can invoke outbound network calls).
// It is up to the unit test to provide each extension used by their template with this API.
// Unit tests can decide if they can use the real implementation of the extension AS IS or if a
// mock extension is needed to feed fake data per test case.
//
// Note: even though input extension is registered inproc, its response to ExecuteTemplateFunction
// will be encoded as JSON and decoded back as a plain inteface{} to simulate the GRPC transport
// layer between stencil and the same extension. Refer to the inprocExt struct docs for details.
func (t *Template) Ext(name string, ext apiv1.Implementation) *Template {
	t.exts[name] = inprocExt{ext: ext}
	return t
}

// ErrorContains denotes that this test run should fail, and the message
// should contain the provided string.
//
//	t.ErrorContains("i am an error")
func (t *Template) ErrorContains(msg string) {
	t.errStr = msg
}

// getModuleDependencies returns modules that are dependencies of the current module
// the top-level manifest should be used to create the module that is passed in to ensure
// that the version criteria is met.
func (t *Template) getModuleDependencies(ctx context.Context, m *modules.Module) ([]*modules.Module, error) {
	token, err := github.GetToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get github token: %v", err)
	}

	mods, err := modules.GetModulesForService(ctx, &modules.ModuleResolveOptions{
		Token:  token,
		Module: m,
		Log:    t.log,
	})
	return mods, err
}

// Run runs the test.
func (t *Template) Run(save bool) {
	t.t.Run(t.path, func(got *testing.T) {
		m, err := modulestest.NewModuleFromTemplates(t.m, append([]string{t.path}, t.additionalTemplates...)...)
		if err != nil {
			got.Fatalf("failed to create module from template %q", t.path)
		}

		mods, err := t.getModuleDependencies(context.Background(), m)
		if err != nil {
			got.Fatalf("could not get modules: %v ", err)
		}
		mods = append(mods, m)

		// Reconcile any declared modules with the found module dependencies.
		for _, m := range mods {
			for _, tm := range t.mods {
				if m.Name == tm.Name {
					m.Version = tm.Version
					break
				}
			}
		}

		t.mods = append([]*configuration.TemplateRepository{{Name: m.Name}}, t.mods...)

		mf := &configuration.ServiceManifest{
			Name:      "testing",
			Arguments: t.args,
			Modules:   t.mods,
		}
		st := codegen.NewStencil(mf, mods, t.log)

		for name, ext := range t.exts {
			st.RegisterInprocExtensions(name, ext)
		}

		tpls, err := st.Render(context.Background(), t.log)
		if err != nil {
			if t.errStr != "" {
				// if t.errStr was set then we expected an error, since that
				// was set via t.ErrorContains()
				if err == nil {
					got.Fatal("expected error, got nil")
				}
				assert.ErrorContains(t.t, err, t.errStr, "expected render to fail with error containing %q", t.errStr)
			} else {
				got.Fatalf("failed to render: %v", err)
			}
		}

		for _, tpl := range tpls {
			// skip templates that aren't the one we are testing
			if tpl.Path != t.path {
				continue
			}

			for _, f := range tpl.Files {
				// skip the snapshot
				if !t.persist {
					continue
				}

				// Create snapshots with a .snapshot ext to keep them away from linters, see Jira for more details.
				// TODO(jaredallard)[DTSS-2086]: figure out what to do with the snapshot codegen.File directive
				snapshotName := f.Name() + ".snapshot"
				// Run each template file as a sub-test, if the sub-test fails, it will report its own error.
				// Even if one of the templates fail, we let the test continue - otherwise devs need to go thru
				// 'onion peeling' excercise to create a bulk of new snapshots or re-run test multiple times to
				// reveal distinct errors for all the templates changed in one PR.
				got.Run(snapshotName, func(got *testing.T) {
					snapshot := cupaloy.New(cupaloy.ShouldUpdate(func() bool { return save }), cupaloy.CreateNewAutomatically(true))
					snapshot.SnapshotT(got, f)
				})
			}

			// only ever process one template
			break
		}
	})
}

// RegenerateSnapshots determines whether to regenerate template
// snapshots based on the presence of the CI environment variable.
// Example usage:
//
//	func TestMyTemplate(t *testing.T) {
//		st := stenciltest.New(t, "path/to/template")
//		// ... test setup
//		st.Run(stenciltest.RegenerateSnapshots())
//	}
func RegenerateSnapshots() bool {
	return os.Getenv("CI") == ""
}
