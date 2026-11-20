// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for ordering modules by dependency.

package modules_test

import (
	"os"
	"slices"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"gotest.tools/v3/assert"
)

// moduleNames returns the names of mods.
func moduleNames(mods []*modules.Module) []string {
	names := make([]string, 0, len(mods))
	for _, m := range mods {
		names = append(names, m.Name)
	}
	return names
}

// sortNames sorts mods by dependency and returns the names in the result and
// the warnings that it logged. It sorts every rotation of mods, which must all
// give the same result, so that the result does not depend on the input order.
func sortNames(t *testing.T, mods []*modules.Module) (names, warnings []string) {
	t.Helper()

	for i := range mods {
		input := slices.Concat(mods[i:], mods[:i])
		before := moduleNames(input)

		log, hook := test.NewNullLogger()
		sorted, err := modules.SortByDependencies(t.Context(), input, log)
		assert.NilError(t, err)
		assert.DeepEqual(t, moduleNames(input), before) // the input is not modified

		entries := hook.AllEntries()
		w := make([]string, 0, len(entries))
		for _, e := range entries {
			assert.Equal(t, e.Level, logrus.WarnLevel)
			w = append(w, e.Message)
		}
		if i == 0 {
			names, warnings = moduleNames(sorted), w
			continue
		}
		assert.DeepEqual(t, moduleNames(sorted), names)
		assert.DeepEqual(t, w, warnings)
	}
	return names, warnings
}

func TestSortByDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mods        []*modules.Module
		want        []string
		wantWarning string // empty if there is no cycle
	}{
		{
			name: "no dependencies sorts by name",
			mods: []*modules.Module{newReplacedModule(t, "c"), newReplacedModule(t, "a"), newReplacedModule(t, "b")},
			want: []string{"a", "b", "c"},
		},
		{
			name: "dependency comes before its dependent",
			mods: []*modules.Module{newReplacedModule(t, "a", "z"), newReplacedModule(t, "z")},
			want: []string{"z", "a"},
		},
		{
			name: "chain",
			mods: []*modules.Module{
				newReplacedModule(t, "a", "b"), newReplacedModule(t, "b", "c"), newReplacedModule(t, "c"),
			},
			want: []string{"c", "b", "a"},
		},
		{
			name: "diamond",
			mods: []*modules.Module{
				newReplacedModule(t, "top", "a", "b"), newReplacedModule(t, "a", "shared"),
				newReplacedModule(t, "b", "shared"), newReplacedModule(t, "shared"),
			},
			want: []string{"shared", "a", "b", "top"},
		},
		{
			name: "independent modules keep name order between dependency chains",
			mods: []*modules.Module{
				newReplacedModule(t, "a", "y"), newReplacedModule(t, "b"), newReplacedModule(t, "y"),
			},
			want: []string{"b", "y", "a"},
		},
		{
			name: "dependency outside the set is ignored",
			mods: []*modules.Module{newReplacedModule(t, "b", "outside"), newReplacedModule(t, "a")},
			want: []string{"a", "b"},
		},
		{
			name: "module that imports itself is ignored",
			mods: []*modules.Module{newReplacedModule(t, "b", "b"), newReplacedModule(t, "a")},
			want: []string{"a", "b"},
		},
		{
			// e is ready first. The cycle a, b, c is broken at a. Then c has its
			// dependency, then b, and d comes after a.
			name: "cycle",
			mods: []*modules.Module{
				newReplacedModule(t, "a", "b"), newReplacedModule(t, "b", "c"), newReplacedModule(t, "c", "a"),
				newReplacedModule(t, "d", "a"), newReplacedModule(t, "e"),
			},
			want:        []string{"e", "a", "c", "b", "d"},
			wantWarning: "modules depend on each other in a cycle (a -> b -> c -> a), running a first",
		},
		{
			// a is the lowest name, but it only imports b, which is in a cycle with
			// c. The cycle is the one that is broken, and a is ready before c.
			name: "module that depends on a cycle",
			mods: []*modules.Module{
				newReplacedModule(t, "a", "b"), newReplacedModule(t, "b", "c"), newReplacedModule(t, "c", "b"),
			},
			want:        []string{"b", "a", "c"},
			wantWarning: "modules depend on each other in a cycle (b -> c -> b), running b first",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, warnings := sortNames(t, tt.mods)
			assert.DeepEqual(t, got, tt.want)

			wantWarnings := []string{}
			if tt.wantWarning != "" {
				wantWarnings = []string{tt.wantWarning}
			}
			assert.DeepEqual(t, warnings, wantWarnings)
		})
	}
}

func TestSortByDependenciesReturnsAManifestError(t *testing.T) {
	t.Parallel()

	// This module has no manifest.yaml.
	broken := modules.NewWithFS(t.Context(), "broken", memfs.New())

	_, err := modules.SortByDependencies(t.Context(),
		[]*modules.Module{newReplacedModule(t, "a"), broken}, logrus.New())
	assert.ErrorIs(t, err, os.ErrNotExist)
}
