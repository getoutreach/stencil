// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file orders modules so that a module comes after the
// modules that it depends on.

package modules

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/sirupsen/logrus"
)

// SortByDependencies returns mods ordered so that each module comes after the
// modules in mods that its manifest lists in modules. Among modules that are
// ready at the same time, the one with the lowest name comes first, so the
// order is the same on every run. The names in mods must be different, and
// mods is not modified.
//
// If modules depend on each other in a cycle, they can not all come after
// their dependencies. SortByDependencies logs a warning that names the cycle
// and puts the module of the cycle with the lowest name first.
func SortByDependencies(ctx context.Context, mods []*Module, log logrus.FieldLogger) ([]*Module, error) {
	remaining := slices.Clone(mods)
	slices.SortFunc(remaining, func(a, b *Module) int { return cmp.Compare(a.Name, b.Name) })

	// deps maps a module name to the names of the modules in mods that it
	// imports, in name order.
	deps := make(map[string][]string, len(remaining))
	for _, m := range remaining {
		for _, other := range remaining {
			if other.Name == m.Name {
				continue
			}
			imports, err := m.ImportsModule(ctx, other.Name)
			if err != nil {
				return nil, err
			}
			if imports {
				deps[m.Name] = append(deps[m.Name], other.Name)
			}
		}
	}

	done := make(map[string]bool, len(remaining))
	// blocker returns a dependency of name that is not in the result yet, or
	// "" if there is none.
	blocker := func(name string) string {
		for _, d := range deps[name] {
			if !done[d] {
				return d
			}
		}
		return ""
	}

	result := make([]*Module, 0, len(remaining))
	for len(remaining) > 0 {
		i := slices.IndexFunc(remaining, func(m *Module) bool { return blocker(m.Name) == "" })
		if i < 0 {
			// Every module waits for another one, so there is a cycle.
			cycle := findCycle(remaining[0].Name, blocker)
			first := slices.Min(cycle)
			log.Warnf("modules depend on each other in a cycle (%s), running %s first",
				strings.Join(slices.Concat(cycle, cycle[:1]), " -> "), first)
			i = slices.IndexFunc(remaining, func(m *Module) bool { return m.Name == first })
		}

		done[remaining[i].Name] = true
		result = append(result, remaining[i])
		remaining = slices.Delete(remaining, i, i+1)
	}

	return result, nil
}

// findCycle follows the blockers from start until a name repeats, and returns
// the names of the cycle in dependency order. Every blocker is a module that
// is not in the result yet and also has a blocker, so the walk ends in a cycle.
func findCycle(start string, blocker func(string) string) []string {
	var path []string
	for name := start; ; name = blocker(name) {
		if i := slices.Index(path, name); i >= 0 {
			return path[i:]
		}
		path = append(path, name)
	}
}
