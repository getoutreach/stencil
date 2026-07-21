// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Online project-manifest validation: builds an argument index
// from resolved modules and validates provided values against declared schemas
// (O2), enforces required arguments (O3), and resolves from: indirection (O4).

package projectmanifest

import (
	"sort"

	"github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// declaration is one module's declaration of an argument (post-from:), retained
// with the declaring module's import path (identity for messages/ordering) and
// its owning manifest (owner.Modules drives the O4 dependency-listing check).
type declaration struct {
	importPath string
	owner      *configuration.TemplateRepositoryManifest
	arg        configuration.Argument
}

// buildArgIndex walks every resolved module's declared arguments in deterministic
// order (by import path, then arg name), resolving from: redirects, and returns
// the index plus any O4 findings. Every declaration for a name is appended (O6
// equivalence comparison is added in PR 3b); O2/O3 consume all of them.
func buildArgIndex(mods []ResolvedModule) (map[string][]declaration, []lint.Finding) {
	sorted := make([]ResolvedModule, len(mods))
	copy(sorted, mods)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ImportPath < sorted[j].ImportPath })

	idx := map[string][]declaration{}
	var f lint.Findings

	for _, rm := range sorted {
		if rm.Manifest == nil {
			continue
		}
		names := make([]string, 0, len(rm.Manifest.Arguments))
		for name := range rm.Manifest.Arguments {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			arg := rm.Manifest.Arguments[name]
			if arg.From != "" {
				resolved, ok := resolveFrom(&f, sorted, rm, name, &arg)
				if !ok {
					continue // O4 finding recorded; skip indexing this declaration
				}
				arg = resolved
			}
			idx[name] = append(idx[name], declaration{
				importPath: rm.ImportPath,
				owner:      rm.Manifest,
				arg:        arg,
			})
		}
	}
	return idx, f.Items()
}

// resolveFrom mirrors render's resolveFrom (internal/codegen/tpl_stencil_arg.go):
// the owning module must list arg.From in its own Modules; the referenced module
// must be in the resolved set and declare the same arg name. Records an O4
// finding and returns ok=false on any failure.
func resolveFrom(f *lint.Findings, mods []ResolvedModule, owner ResolvedModule,
	name string, arg *configuration.Argument) (configuration.Argument, bool) {
	// The owning module must declare arg.From as a dependency.
	listed := false
	for _, m := range owner.Manifest.Modules {
		if m.Name == arg.From {
			listed = true
			break
		}
	}
	if !listed {
		f.Errorf("arguments."+name,
			"argument %q uses 'from: %s' but module %q does not list %q in its 'modules'; "+
				"add it as a dependency", name, arg.From, owner.ImportPath, arg.From)
		return configuration.Argument{}, false
	}
	// The referenced module must be in the resolved set.
	var ref *ResolvedModule
	for i := range mods {
		if mods[i].ImportPath == arg.From {
			ref = &mods[i]
			break
		}
	}
	if ref == nil || ref.Manifest == nil {
		f.Errorf("arguments."+name,
			"argument %q references module %q via 'from:', but it is not in the resolved "+
				"dependency graph", name, arg.From)
		return configuration.Argument{}, false
	}
	// The referenced module must declare the same argument name.
	refArg, ok := ref.Manifest.Arguments[name]
	if !ok {
		f.Errorf("arguments."+name,
			"argument %q references module %q via 'from:', but %q does not declare "+
				"argument %q", name, arg.From, arg.From, name)
		return configuration.Argument{}, false
	}
	return refArg, true
}
