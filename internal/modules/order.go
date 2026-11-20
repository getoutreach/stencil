// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file orders modules so that a module comes after the
// modules that it depends on.

package modules

import (
	"cmp"
	"context"
	"maps"
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
// their dependencies. SortByDependencies logs a warning that names the modules
// of the cycle and puts the one with the lowest name first. It breaks only a
// cycle that waits for no module outside it, so a cycle that depends on
// another cycle comes after that cycle.
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

	// reachable returns the modules that name depends on, directly or through
	// other modules, and that are not in the result yet.
	reachable := func(name string) map[string]bool {
		seen := map[string]bool{}
		for stack := []string{name}; len(stack) > 0; {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, d := range deps[current] {
				if !done[d] && !seen[d] {
					seen[d] = true
					stack = append(stack, d)
				}
			}
		}
		return seen
	}

	result := make([]*Module, 0, len(remaining))
	for len(remaining) > 0 {
		i := slices.IndexFunc(remaining, func(m *Module) bool { return len(reachable(m.Name)) == 0 })
		if i < 0 {
			// Every module waits for another one, so there is a cycle.
			cycle := findCycle(remaining, reachable)
			first := cycle[0]
			log.Warnf("modules depend on each other in a cycle (%s), running %s first",
				strings.Join(cycle, ", "), first)
			i = slices.IndexFunc(remaining, func(m *Module) bool { return m.Name == first })
		}

		done[remaining[i].Name] = true
		result = append(result, remaining[i])
		remaining = slices.Delete(remaining, i, i+1)
	}

	return result, nil
}

// findCycle returns the sorted names of the modules in the lowest-named cycle
// that waits for no module outside it. A module is in such a cycle if every
// module that it reaches reaches it back.
func findCycle(remaining []*Module, reachable func(string) map[string]bool) []string {
	for _, m := range remaining {
		names := slices.Sorted(maps.Keys(reachable(m.Name)))
		if !slices.ContainsFunc(names, func(name string) bool { return !reachable(name)[m.Name] }) {
			return names
		}
	}

	// Every module waits for another one, so a cycle is always found above.
	return []string{remaining[0].Name}
}
