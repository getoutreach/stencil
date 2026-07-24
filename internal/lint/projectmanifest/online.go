// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Online project-manifest validation: builds an argument index
// from resolved modules and validates provided values against declared schemas
// (O2), enforces required arguments (O3), and resolves from: indirection (O4).

package projectmanifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/getoutreach/stencil/internal/dotnotation"
	"github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// ValidateOnline runs the offline checks (Validate) then appends the online
// argument checks (O2–O8) against the already-resolved modules, in a
// deterministic total order (offline findings first; online sorted by Path,
// then message). It executes no templates and performs no module resolution —
// the command layer resolves each module's Manifest(ctx) and passes it in via
// ResolvedModule. O1 (resolution / per-module manifest failure) is produced by
// the command layer, so this is only reached on a fully-resolved module set.
func ValidateOnline(res *LoadResult, mods []ResolvedModule) []lint.Finding {
	offline := Validate(res)

	idx, o4 := buildArgIndex(mods)
	var online []lint.Finding
	online = append(online, o4...)                                  // O4
	online = append(online, checkArguments(res, idx)...)            // O2, O3
	online = append(online, checkReplacements(res, mods)...)        // O5, O8
	online = append(online, checkSchemaConflicts(idx)...)           // O6
	online = append(online, checkUndeclaredArgs(res, idx, mods)...) // O7
	sortOnline(online)

	return append(offline, online...)
}

// sortOnline sorts online findings by Path, then Message (a stable secondary
// key). Since O2/O3/O4 all key on arguments.<name>, Path ordering plus the
// Message tie-break is deterministic.
func sortOnline(findings []lint.Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Message < findings[j].Message
	})
}

// uriIsLocal reports whether a replacement value is a local path (no scheme, or
// a file:// URL), mirroring internal/modules' unexported uriIsLocal. Reimplemented
// here to avoid exporting it from the modules package.
func uriIsLocal(uri string) bool {
	return !strings.Contains(uri, "://") || strings.HasPrefix(uri, "file://")
}

// checkReplacements implements O5 (a replacement key matching no resolved module
// is inert → warning) and O8 (a replacement key that DOES match a resolved
// module, whose value is a local path that does not exist → error). Remote
// values on matched keys are not checked (a bad remote surfaces as O1). The two
// checks share the matched-key computation and are the only replacement checks;
// offline says nothing about replacements. Keys are processed in sorted order.
func checkReplacements(res *LoadResult, mods []ResolvedModule) []lint.Finding {
	var f lint.Findings
	if res.Manifest == nil || len(res.Manifest.Replacements) == 0 {
		return f.Items()
	}
	resolved := make(map[string]struct{}, len(mods))
	for _, m := range mods {
		resolved[m.ImportPath] = struct{}{}
	}

	for _, key := range slices.Sorted(maps.Keys(res.Manifest.Replacements)) {
		value := res.Manifest.Replacements[key]
		if _, matched := resolved[key]; !matched {
			// O5: inert replacement key.
			f.Warnf("replacements."+key,
				"replacement for %q matches no module in the dependency graph and is "+
					"ignored; remove it, or correct the import path to a resolved module", key)
			continue
		}
		// Matched key: O8 checks a local path's existence.
		if uriIsLocal(value) {
			path := strings.TrimPrefix(value, "file://")
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				f.Errorf("replacements."+key,
					"replacement for %q points at a local path that does not exist: %s; "+
						"create the path, or correct the replacement", key, path)
			}
		}
	}
	return f.Items()
}

// checkUndeclaredArgs implements O7: a service.yaml argument whose key matches no
// declared argument (from the index) is flagged as likely a typo/leftover. It is
// suppressed entirely when any resolved module declares a dynamic/catch-all
// argument, since then an "undeclared" key may be legitimately consumed. Matching
// is at top-level-key granularity: a provided top-level key is undeclared iff no
// declared name has it as its first (dot-separated) segment, so a dotted declared
// name (e.g. aws.IRSA) matches a correctly-nested value. Provided arg keys are
// processed in sorted order.
func checkUndeclaredArgs(res *LoadResult, idx map[string][]declaration, mods []ResolvedModule) []lint.Finding {
	var f lint.Findings
	if res.Manifest == nil || len(res.Manifest.Arguments) == 0 {
		return f.Items()
	}
	if hasDynamicArg(mods) {
		return f.Items() // carve-out: catch-all arg present → suppress O7
	}

	// Build the set of top-level service.yaml keys and check each against the
	// declared set. A provided top-level key is undeclared iff NO declared name
	// resolves into it. Since declared names may be dotted (nested), we check
	// whether any declared name has this top-level key as its first segment.
	declaredTop := map[string]struct{}{}
	for name := range idx {
		top := name
		if i := strings.IndexByte(name, '.'); i >= 0 {
			top = name[:i]
		}
		declaredTop[top] = struct{}{}
	}

	for _, key := range slices.Sorted(maps.Keys(res.Manifest.Arguments)) {
		if _, ok := declaredTop[key]; ok {
			continue // this top-level key is (a prefix of) a declared name
		}
		f.Warnf("arguments."+key,
			"no resolved module declares argument %q; check for a typo or remove it", key)
	}
	return f.Items()
}

// hasDynamicArg reports whether any resolved module declares a catch-all argument
// — a schema that is an open object (type object with additionalProperties not
// disabled). Such a module may legitimately consume otherwise-undeclared keys, so
// O7 is suppressed when one is present.
func hasDynamicArg(mods []ResolvedModule) bool {
	for _, m := range mods {
		if m.Manifest == nil {
			continue
		}
		for _, arg := range m.Manifest.Arguments {
			if len(arg.Schema) == 0 {
				continue
			}
			t, _ := arg.Schema["type"].(string)
			if t != "object" {
				continue
			}
			// Open object: additionalProperties absent (default true), explicitly
			// true, a sub-schema, or present-but-null (treated as open).
			switch ap := arg.Schema["additionalProperties"].(type) {
			case nil:
				return true // key absent, or present-but-null → open
			case bool:
				if ap {
					return true // additionalProperties: true → open
				}
			case map[string]interface{}:
				return true // additionalProperties is a sub-schema → open
			}
		}
	}
	return false
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

// checkSchemaConflicts implements O6: for each argument name declared by 2+
// modules with NON-equivalent schemas, emit one warning naming the first two
// declarations (in sorted import-path order) whose schemas differ. Never fatal;
// both schemas still drive O2. Arg names processed in sorted order.
func checkSchemaConflicts(idx map[string][]declaration) []lint.Finding {
	var f lint.Findings
	for _, name := range slices.Sorted(maps.Keys(idx)) {
		decls := idx[name]
		if len(decls) < 2 {
			continue
		}
		// Sort declarations by import path for a deterministic "first two disagreeing".
		sorted := make([]declaration, len(decls))
		copy(sorted, decls)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].importPath < sorted[j].importPath })

		found := false
		for i := 0; i < len(sorted) && !found; i++ {
			for j := i + 1; j < len(sorted); j++ {
				if !schemaEquivalent(sorted[i].arg.Schema, sorted[j].arg.Schema) {
					f.Warnf("arguments."+name,
						"modules %q and %q disagree on the schema for argument %q; "+
							"align their manifests", sorted[i].importPath, sorted[j].importPath, name)
					found = true
					break
				}
			}
		}
	}
	return f.Items()
}

// schemaEquivalent reports whether two argument schemas are byte-equal after a
// JSON marshal. encoding/json sorts map keys recursively and preserves slice
// order, so the marshaled bytes are canonical and the comparison is
// order-insensitive for keys. A nil schema and an empty schema are both treated
// as "no schema" and are equivalent to each other. This is a conservative
// comparison: a cosmetic difference (e.g. an added description) counts as
// non-equivalent, which is acceptable for a warning.
func schemaEquivalent(a, b map[string]interface{}) bool {
	aEmpty, bEmpty := len(a) == 0, len(b) == 0
	if aEmpty || bEmpty {
		return aEmpty && bEmpty
	}
	ab, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bb, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ab, bb)
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
