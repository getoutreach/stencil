// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements static validation of a Stencil template repository
// manifest (manifest.yaml) without resolving dependencies or accessing the
// network.

// Package manifest implements static validation of a Stencil template
// repository manifest (manifest.yaml) without resolving dependencies or
// accessing the network.
package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"

	"github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// Load reads a manifest from r and decodes it twice: once strictly
// (rejecting unknown keys) to surface structural problems, and once leniently
// so the returned manifest is populated for field-level checks even when the
// strict decode failed.
//
// strictErr is the strict-decode error (if any); for empty input it is io.EOF.
// mf is nil only when the lenient decode also fails (truly malformed YAML).
// multiDoc is true when r contains more than one YAML document. readErr is
// non-nil only when reading from r itself fails.
func Load(r io.Reader) (mf *configuration.TemplateRepositoryManifest,
	strictErr error, multiDoc bool, readErr error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, false, err
	}

	// Strict decode into a throwaway manifest to capture unknown-key/type errors.
	strictDec := yaml.NewDecoder(bytes.NewReader(raw))
	strictDec.KnownFields(true)
	var strictManifest configuration.TemplateRepositoryManifest
	strictErr = strictDec.Decode(&strictManifest)

	// Lenient decode so field checks can run regardless of strictErr. Retain the
	// decoder so we can probe for a second document afterward.
	lenientDec := yaml.NewDecoder(bytes.NewReader(raw))
	var lenientManifest configuration.TemplateRepositoryManifest
	if err := lenientDec.Decode(&lenientManifest); err != nil {
		// Empty or malformed input: no usable manifest for field checks.
		return nil, strictErr, false, nil
	}
	mf = &lenientManifest

	// Probe for a second document on the (already-advanced) lenient decoder.
	var discard any
	if err := lenientDec.Decode(&discard); err == nil {
		multiDoc = true
	}

	return mf, strictErr, multiDoc, nil
}

// Validate runs the manifest lint checks and returns every finding. It never
// fails fast. strictErr is the error (if any) returned by the strict YAML
// decode in Load; mf may be nil if the YAML could not be decoded at all, in
// which case only the strict-decode finding (check 1) is returned and checks
// 2-7 are skipped.
func Validate(mf *configuration.TemplateRepositoryManifest, strictErr error) []lint.Finding {
	var f lint.Findings

	// Check 1: strict decode succeeded.
	if strictErr != nil {
		if errors.Is(strictErr, io.EOF) {
			f.Errorf("manifest.yaml", "manifest is empty")
		} else {
			f.Errorf("manifest.yaml", "invalid manifest: %v", strictErr)
		}
	}

	if mf == nil {
		return f.Items()
	}

	checkName(&f, mf, strictErr)
	checkTypes(&f, mf)
	checkStencilVersion(&f, mf)
	checkArguments(&f, mf)
	checkModules(&f, mf)

	return f.Items()
}

// checkName implements check 2. A manifest's name is a Go import path
// (e.g. github.com/getoutreach/stencil-base), not a service name, so the only
// requirement is that it is present.
func checkName(f *lint.Findings, mf *configuration.TemplateRepositoryManifest, strictErr error) {
	if mf.Name == "" {
		// Suppress the redundant emptiness finding when the decode error already
		// references the name field (e.g. a non-scalar like a mapping given for name).
		if strictErr != nil && strings.Contains(strictErr.Error(), "name") {
			return
		}
		f.Errorf("name", "name is required")
	}
}

// checkTypes implements check 3.
func checkTypes(f *lint.Findings, mf *configuration.TemplateRepositoryManifest) {
	for _, token := range mf.Type.Types() {
		if !token.IsValid() {
			f.Errorf("type", "unknown type %q (valid: extension, templates)", token)
		}
	}
}

// checkStencilVersion implements check 5.
func checkStencilVersion(f *lint.Findings, mf *configuration.TemplateRepositoryManifest) {
	if mf.StencilVersion == "" {
		return
	}
	if _, err := semver.NewConstraint(mf.StencilVersion); err != nil {
		f.Errorf("stencilVersion", "invalid stencilVersion constraint: %v", err)
	}
}

// checkArguments implements checks 4, 6, and 7 (argument deprecations) in
// sorted key order, skipping arguments that reference another module via from:.
// Also emits an informational finding for each argument that sets the
// deprecated property.
func checkArguments(f *lint.Findings, mf *configuration.TemplateRepositoryManifest) {
	names := make([]string, 0, len(mf.Arguments))
	for name := range mf.Arguments {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		arg := mf.Arguments[name]
		if arg.From != "" {
			// from: arguments ignore all other fields at render time; skip
			// field-level checks to avoid false positives.
			continue
		}
		if arg.Schema != nil {
			if err := compileSchema(name, arg.Schema); err != nil {
				f.Errorf("arguments."+name+".schema", "invalid JSON schema: %v", err)
			}
		}
		if arg.Required && arg.Default != nil {
			f.Errorf("arguments."+name, "required argument must not set a default")
		}
		//nolint:staticcheck // Why: the linter intentionally reads deprecated fields to warn about their use.
		if arg.Type != "" {
			f.Warnf("arguments."+name+".type", "argument field 'type' is deprecated; use 'schema'")
		}
		//nolint:staticcheck // Why: the linter intentionally reads deprecated fields to warn about their use.
		if len(arg.Values) > 0 {
			f.Warnf("arguments."+name+".values", "argument field 'values' is deprecated; use 'schema'")
		}
		if arg.Deprecated != "" {
			f.Infof("arguments."+name, "argument %q is deprecated: %s", name, arg.Deprecated)
		}
	}
}

// checkModules implements check 7 for module deprecations, in slice order.
func checkModules(f *lint.Findings, mf *configuration.TemplateRepositoryManifest) {
	for i, m := range mf.Modules {
		path := modulePath(m, i)
		//nolint:staticcheck // Why: the linter intentionally reads deprecated fields to warn about their use.
		if m.URL != "" {
			f.Warnf(path+".url", "module field 'url' is deprecated; use 'name'")
		}
		//nolint:staticcheck // Why: the linter intentionally reads deprecated fields to warn about their use.
		if m.Prerelease {
			f.Warnf(path+".prerelease", "module field 'prerelease' is deprecated; use 'channel: rc'")
		}
	}
}

// modulePath builds the finding path for module i, preferring its name.
func modulePath(m *configuration.TemplateRepository, i int) string {
	if m.Name != "" {
		return "modules." + m.Name
	}
	return "modules[" + strconv.Itoa(i) + "]"
}

// compileSchema compiles a single argument schema (Draft 2020-12) without
// validating a value, surfacing malformed schemas. Mirrors the render-time
// compiler in internal/codegen/tpl_stencil_arg.go.
func compileSchema(name string, schema map[string]interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(schema); err != nil {
		return err
	}
	jsc := jsonschema.NewCompiler()
	jsc.Draft = jsonschema.Draft2020
	// Fail closed on any external $ref: lint must not read the filesystem or
	// network while compiling a manifest's schema. By default jsonschema's loader
	// registry includes a "file" scheme loader, which would read local files for a
	// file:// $ref; overriding LoadURL disables all external reference resolution.
	jsc.LoadURL = func(ref string) (io.ReadCloser, error) {
		return nil, fmt.Errorf("external $ref not allowed in lint: %s", ref)
	}
	url := "manifest.yaml/arguments/" + name
	if err := jsc.AddResource(url, buf); err != nil {
		return err
	}
	_, err := jsc.Compile(url)
	return err
}
