// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements static offline validation of a Stencil project
// manifest (service.yaml) without resolving modules or accessing the network.

// Package projectmanifest implements static offline validation of a Stencil
// project manifest (service.yaml). Unlike the module-manifest linter it does a
// lenient decode only: a service.yaml's argument keys are defined by the
// modules it loads, so strict unknown-key rejection would be wrong. Argument
// value validation is an online concern handled elsewhere.
package projectmanifest

import (
	"bytes"
	"errors"
	"io"
	"sort"
	"strconv"

	semver "github.com/Masterminds/semver/v3"
	"go.yaml.in/yaml/v3"

	"github.com/getoutreach/stencil/internal/lint"
	"github.com/getoutreach/stencil/pkg/configuration"
)

// LoadResult holds the outcome of decoding a service.yaml for linting.
type LoadResult struct {
	// Manifest is the leniently-decoded manifest, or nil if the YAML could not
	// be decoded into a mapping at all.
	Manifest *configuration.ServiceManifest
	// Root is the first document's YAML node tree, used to resolve finding
	// paths to source lines. Nil if the bytes did not parse as a node.
	Root *yaml.Node
	// DecodeErr is the lenient-decode error (empty input, malformed YAML, or a
	// non-mapping top-level document), or nil. For empty input it is io.EOF.
	DecodeErr error
	// MultiDoc is true when the input contains more than one YAML document.
	MultiDoc bool
}

// Load reads a service.yaml from r and decodes it leniently (no strict
// unknown-key check), plus into a yaml.Node tree for source-line annotation.
// The returned error is non-nil only when reading from r itself fails.
func Load(r io.Reader) (*LoadResult, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	res := &LoadResult{}

	// Node decode for source positions. A failure here is non-fatal: field
	// checks still run on the struct path, just without line numbers.
	var node yaml.Node
	if err := yaml.Unmarshal(raw, &node); err == nil {
		res.Root = &node
	}

	// Lenient decode (retain the decoder to probe for a second document).
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	var mf configuration.ServiceManifest
	if err := dec.Decode(&mf); err != nil {
		// Empty, malformed, or non-mapping input: no usable manifest.
		res.DecodeErr = err
		return res, nil
	}
	res.Manifest = &mf

	// Probe for a second document on the advanced decoder.
	var discard any
	if err := dec.Decode(&discard); err == nil {
		res.MultiDoc = true
	}

	return res, nil
}

// Validate runs the offline project-manifest checks (F1–F7) and returns every
// finding, annotated with the source line of the referenced YAML key where
// resolvable. It never fails fast. res.Manifest may be nil (empty/malformed
// input), in which case only the F1 finding is returned.
func Validate(res *LoadResult) []lint.Finding {
	var f lint.Findings

	// F1: the document decoded into a mapping.
	if res.Manifest == nil {
		if errors.Is(res.DecodeErr, io.EOF) {
			f.Errorf("service.yaml",
				"service.yaml is empty; add at least a 'name' field (e.g. 'name: my-service')")
		} else {
			f.Errorf("service.yaml",
				"invalid service.yaml: %v; check the YAML syntax near this location",
				res.DecodeErr)
		}
		return f.Items()
	}

	checkName(&f, res.Manifest)
	checkModules(&f, res.Manifest)
	checkVersions(&f, res.Manifest)

	// Annotate each finding with its source line where resolvable.
	findings := f.Items()
	if res.Root != nil {
		for i := range findings {
			if findings[i].Line == 0 {
				findings[i].Line = resolvePath(res.Root, findings[i].Path)
			}
		}
	}
	return findings
}

// checkName implements F2. A service manifest's name is a service name and must
// be present and match the service-name regex (unlike a module manifest, whose
// name is an import path).
func checkName(f *lint.Findings, mf *configuration.ServiceManifest) {
	if mf.Name == "" {
		f.Errorf("name",
			"name is required; add a 'name' field matching `^[_a-z][_a-z0-9-]*$` "+
				"(e.g. 'name: my-service')")
		return
	}
	if !configuration.ValidateName(mf.Name) {
		f.Errorf("name",
			"'name' %q is invalid; use a value matching `^[_a-z][_a-z0-9-]*$` "+
				"(e.g. 'my-service')", mf.Name)
	}
}

// checkModules implements F3 (name required), F4 (version constraint valid),
// F5 (deprecated url), F6 (deprecated prerelease), in slice order.
func checkModules(f *lint.Findings, mf *configuration.ServiceManifest) {
	for i, m := range mf.Modules {
		path := modulePath(m, i)
		if m.Name == "" {
			f.Errorf(path,
				"module name is required; set 'name' to the module's import path "+
					"(e.g. 'github.com/getoutreach/stencil-base')")
		}
		if m.Version != "" {
			if _, err := semver.NewConstraint(m.Version); err != nil {
				f.Errorf(path+".version",
					"invalid version constraint: %v; use a valid semver constraint "+
						"(e.g. '>=1.0.0')", err)
			}
		}
		//nolint:staticcheck // Why: the linter intentionally reads deprecated fields to warn about their use.
		if m.URL != "" {
			f.Warnf(path+".url",
				"module field 'url' is deprecated; replace it with 'name' set to the "+
					"module's import path, then remove 'url' (not migrated automatically by --fix)")
		}
		//nolint:staticcheck // Why: the linter intentionally reads deprecated fields to warn about their use.
		if m.Prerelease {
			f.Warnf(path+".prerelease",
				"module field 'prerelease' is deprecated; use 'channel: rc' instead "+
					"(run 'stencil lint --fix' to migrate it automatically)")
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

// checkVersions implements F7: a versions entry with an empty-string value.
func checkVersions(f *lint.Findings, mf *configuration.ServiceManifest) {
	names := make([]string, 0, len(mf.Versions))
	for name := range mf.Versions {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if mf.Versions[name] == "" {
			f.Warnf("versions."+name,
				"'versions.%s' is empty; set a value for '%s' or remove the entry",
				name, name)
		}
	}
}
