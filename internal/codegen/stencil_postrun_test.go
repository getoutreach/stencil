// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the order of the post-run commands
// of modules.

package codegen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gotest.tools/v3/assert"
)

// postRunModule returns a module that depends on deps. It has two post-run
// commands, "echo <name>:one" and "echo <name>:two".
func postRunModule(t *testing.T, name string, deps ...string) *modules.Module {
	t.Helper()

	var manifest strings.Builder
	fmt.Fprintf(&manifest, "name: %s\nmodules:\n", name)
	for _, d := range deps {
		fmt.Fprintf(&manifest, "  - name: %s\n", d)
	}
	fmt.Fprintf(&manifest, "postRunCommand:\n  - name: one\n    command: echo %[1]s:one\n  - name: two\n    command: echo %[1]s:two\n", name)

	return modules.NewWithFS(t.Context(), name, createFakeModuleFSWithManifest(t, manifest.String()))
}

func TestPostRunOrder(t *testing.T) {
	tests := []struct {
		name        string
		mods        []*modules.Module
		want        []string
		wantWarning string // empty if there is no cycle
	}{
		{
			name: "without dependencies the order is by name",
			mods: []*modules.Module{postRunModule(t, "b"), postRunModule(t, "a")},
			want: []string{"echo a:one", "echo a:two", "echo b:one", "echo b:two"},
		},
		{
			// a is first by name, but it depends on z, and z depends on m.
			name: "dependencies run first",
			mods: []*modules.Module{postRunModule(t, "a", "z"), postRunModule(t, "m"), postRunModule(t, "z", "m")},
			want: []string{
				"echo m:one", "echo m:two", "echo z:one", "echo z:two", "echo a:one", "echo a:two",
			},
		},
		{
			name: "a cycle runs every command once",
			mods: []*modules.Module{postRunModule(t, "x", "y"), postRunModule(t, "y", "x"), postRunModule(t, "z", "x")},
			want: []string{
				"echo x:one", "echo x:two", "echo y:one", "echo y:two", "echo z:one", "echo z:two",
			},
			wantWarning: "modules depend on each other in a cycle (x -> y -> x), running x first",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &commandRecorder{}
			st := NewStencil(&configuration.ServiceManifest{Name: "test"}, tt.mods, logrus.New(), WithCommandRunner(r))

			log, hook := test.NewNullLogger()
			assert.NilError(t, st.PostRun(t.Context(), log))

			assert.DeepEqual(t, r.commands, tt.want)

			warnings := []string{}
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.WarnLevel {
					warnings = append(warnings, e.Message)
				}
			}
			wantWarnings := []string{}
			if tt.wantWarning != "" {
				wantWarnings = []string{tt.wantWarning}
			}
			assert.DeepEqual(t, warnings, wantWarnings)
		})
	}
}

func TestPostRunReturnsAnErrorForAModuleWithAnInvalidManifest(t *testing.T) {
	// The manifest of this module has another name than the module.
	broken := modules.NewWithFS(t.Context(), "broken", createFakeModuleFSWithManifest(t, "name: other\n"))

	st := NewStencil(&configuration.ServiceManifest{Name: "test"},
		[]*modules.Module{postRunModule(t, "a"), broken}, logrus.New(), WithCommandRunner(&commandRecorder{}))

	assert.ErrorContains(t, st.PostRun(t.Context(), logrus.New()), "declared")
}
