// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Online project-manifest validation: builds an argument index
// from resolved modules and validates provided values against declared schemas
// (O2), enforces required arguments (O3), and resolves from: indirection (O4).

package projectmanifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/getoutreach/stencil/internal/dotnotation"
	"github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// ValidateOnline runs the offline checks (Validate) then appends the online
// argument checks (O2–O4) against the already-resolved modules, in a
// deterministic total order (offline findings first; online sorted by Path,
// then declaring-module import path, then check ID). It executes no templates
// and performs no module resolution — the command layer resolves each module's
// Manifest(ctx) and passes it in via ResolvedModule. O1 (resolution / per-module
// manifest failure) is produced by the command layer, so this is only reached
// on a fully-resolved module set. PR 3b appends O5–O8.
func ValidateOnline(res *LoadResult, mods []ResolvedModule) []lint.Finding {
	offline := Validate(res)

	idx, o4 := buildArgIndex(mods)
	online := append(o4, checkArguments(res, idx)...)
	sortOnline(online)

	return append(offline, online...)
}

// sortOnline sorts online findings by Path, then by the module import path
// embedded in the message where present (a stable secondary key). Since
// O2/O3/O4 all key on arguments.<name>, Path ordering plus the message tie-break
// is deterministic.
func sortOnline(findings []lint.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Message < findings[j].Message
	})
}

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

// checkArguments implements O2 (value vs schema) and O3 (required). For each
// argument name in the index (sorted), it locates the provided value in the
// service.yaml arguments and validates it against EACH declaration's schema,
// attributing failures to the declaring module. A required declaration with no
// provided value and no default yields O3.
func checkArguments(res *LoadResult, idx map[string][]declaration) []lint.Finding {
	var f lint.Findings
	if res.Manifest == nil {
		return f.Items()
	}
	names := make([]string, 0, len(idx))
	for name := range idx {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		value, provided := providedValue(res.Manifest.Arguments, name)
		for _, d := range idx[name] {
			if provided {
				// O2: validate the value against this declaration's schema.
				if len(d.arg.Schema) > 0 {
					if err := validateValue(name, d.arg.Schema, value); err != nil {
						f.Errorf("arguments."+name,
							"argument %q does not satisfy the schema declared by module %q: %v",
							name, d.importPath, err)
					}
				}
			} else {
				// O3: required with no default and no value.
				if d.arg.Required && d.arg.Default == nil {
					f.Errorf("arguments."+name,
						"argument %q is required by module %q but is not set; "+
							"set 'arguments.%s' in service.yaml", name, d.importPath, name)
				}
			}
		}
	}
	return f.Items()
}

// providedValue reports whether name is provided in the service.yaml arguments
// (key present AND value not nil), returning the value. An explicit null counts
// as NOT provided. dotnotation.Get is used only to locate the value (supporting
// dotted argument names); an absent key returns an error (not provided), a
// present key returns (value, nil) — including (nil, nil) for an explicit null,
// which we treat as not provided.
func providedValue(args map[string]interface{}, name string) (interface{}, bool) {
	if args == nil {
		return nil, false
	}
	mapInf := make(map[interface{}]interface{}, len(args))
	for k, v := range args {
		mapInf[k] = v
	}
	v, err := dotnotation.Get(mapInf, name)
	if err != nil {
		return nil, false // key absent
	}
	if v == nil {
		return nil, false // explicit null: treat as not provided
	}
	return v, true
}

// validateValue compiles a single argument schema (Draft 2020-12) HERMETICALLY
// (external $ref rejected — no filesystem/network) and validates v against it.
// Mirrors internal/lint/manifest/compileSchema, extended to also validate a
// value. It intentionally diverges from render's non-hermetic validateArg.
func validateValue(name string, schema map[string]interface{}, v interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(schema); err != nil {
		return err
	}
	jsc := jsonschema.NewCompiler()
	jsc.Draft = jsonschema.Draft2020
	jsc.LoadURL = func(ref string) (io.ReadCloser, error) {
		return nil, fmt.Errorf("external $ref not allowed in lint: %s", ref)
	}
	url := "service.yaml/arguments/" + name
	if err := jsc.AddResource(url, buf); err != nil {
		return err
	}
	compiled, err := jsc.Compile(url)
	if err != nil {
		return err
	}
	return compiled.Validate(v)
}
