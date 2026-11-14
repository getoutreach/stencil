// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the Suite, which loads the module under
// test and its dependencies once per test package.

package rendertest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getoutreach/gobox/pkg/cli/github"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v3"
)

// Suite holds the manifest of the module under test and its dependency
// modules, for all tests in a package. Create it with NewSuite in TestMain.
// After Run loads it, it is read-only and safe for concurrent use.
type Suite struct {
	// m runs the tests of the package
	m *testing.M

	// manifestPath is the path to the manifest.yaml of the module under test
	manifestPath string

	// replacements maps import paths to local module directories
	replacements map[string]string

	// resolveModules is whether dependency modules are resolved
	resolveModules bool

	// loaded is whether Run has loaded the suite
	loaded bool

	// name is the name of the module under test
	name string

	// manifestYAML is the contents of the manifest of the module under test
	manifestYAML []byte

	// deps are the dependency modules. They have a manifest and no templates.
	deps []*modules.Module
}

// SuiteOption configures a Suite.
type SuiteOption func(*Suite)

// WithManifestPath sets the path to the manifest.yaml of the module under
// test. The default is manifest.yaml in the root directory of the Go module
// that holds the test package.
func WithManifestPath(p string) SuiteOption {
	return func(s *Suite) {
		s.manifestPath = p
	}
}

// WithModuleReplacements sets local directories to use for modules, by
// import path. Replaced modules are read from disk instead of the network,
// so a suite whose dependencies are all replaced does not use the network.
func WithModuleReplacements(replacements map[string]string) SuiteOption {
	return func(s *Suite) {
		s.replacements = replacements
	}
}

// WithoutModuleResolution renders without dependency modules. Templates that
// read the arguments or the manifests of dependency modules fail.
func WithoutModuleResolution() SuiteOption {
	return func(s *Suite) {
		s.resolveModules = false
	}
}

// NewSuite returns a Suite for the tests that m runs. Call Run to load it
// and run the tests.
func NewSuite(m *testing.M, opts ...SuiteOption) *Suite {
	s := &Suite{m: m, resolveModules: true}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run reads the module manifest, resolves the dependency modules and then
// runs the tests. It returns the exit code for os.Exit. Resolving modules
// uses the network once, unless WithModuleReplacements covers every
// dependency or WithoutModuleResolution is set.
func (s *Suite) Run() int {
	if err := s.load(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "rendertest:", err)
		return 1
	}
	return s.m.Run()
}

// load reads the manifest of the module under test and resolves its
// dependencies.
func (s *Suite) load(ctx context.Context) error {
	if s.manifestPath == "" {
		p, err := defaultManifestPath(ctx)
		if err != nil {
			return err
		}
		s.manifestPath = p
	}

	b, err := os.ReadFile(s.manifestPath)
	if err != nil {
		return errors.Wrap(err, "failed to read the module manifest")
	}
	var mf configuration.TemplateRepositoryManifest
	if err := yaml.Unmarshal(b, &mf); err != nil {
		return errors.Wrapf(err, "failed to parse the module manifest %q", s.manifestPath)
	}
	s.name = mf.Name
	s.manifestYAML = b

	if s.resolveModules {
		if s.deps, err = s.resolveDependencies(ctx); err != nil {
			return err
		}
	}

	s.loaded = true
	return nil
}

// defaultManifestPath returns the path to manifest.yaml in the root
// directory of the Go module that holds the working directory.
func defaultManifestPath(ctx context.Context) (string, error) {
	b, err := exec.CommandContext(ctx, "go", "env", "GOMOD").Output()
	if err != nil {
		return "", errors.Wrap(err, "failed to find the Go module of the tests")
	}

	gomod := strings.TrimSpace(string(b))
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("the tests are not in a Go module, set the manifest path with WithManifestPath")
	}
	return filepath.Join(filepath.Dir(gomod), "manifest.yaml"), nil
}

// resolveDependencies resolves the dependency modules of the module under
// test, and returns a copy of each that has only its manifest.
func (s *Suite) resolveDependencies(ctx context.Context) ([]*modules.Module, error) {
	// Only the manifest of the module under test is needed to resolve it.
	self, err := newModule(ctx, s.name, "", s.manifestYAML)
	if err != nil {
		return nil, err
	}

	replacements := make(map[string]*modules.Module, len(s.replacements))
	for name, dir := range s.replacements {
		m, err := modules.New(ctx, dir, &configuration.TemplateRepository{Name: name})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create the replacement of module %q", name)
		}
		replacements[name] = m
	}

	token, err := github.GetToken()
	if err != nil {
		token = ""
	}

	log := logrus.New()
	log.SetLevel(logrus.WarnLevel)
	resolved, err := modules.GetModulesForService(ctx, &modules.ModuleResolveOptions{
		Token:        token,
		Module:       self,
		Replacements: replacements,
		Log:          log,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve the dependency modules")
	}

	deps := make([]*modules.Module, 0, len(resolved))
	for _, m := range resolved {
		if m.Name == s.name {
			continue
		}

		fs, err := m.GetFS(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read module %q", m.Name)
		}
		mf, err := util.ReadFile(fs, "manifest.yaml")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read the manifest of module %q", m.Name)
		}

		dep, err := newModule(ctx, m.Name, m.Version, mf)
		if err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}

	return deps, nil
}

// newModule returns an in-memory module with the given manifest and with the
// given template files from disk, in its templates directory. If version is
// empty, the module keeps the version of in-memory modules.
func newModule(ctx context.Context, name, version string, manifest []byte, templates ...string) (*modules.Module, error) {
	fs := memfs.New()
	if err := util.WriteFile(fs, "manifest.yaml", manifest, 0o644); err != nil {
		return nil, errors.Wrap(err, "failed to write the module manifest")
	}
	// Without a templates directory, the engine warns that the templates are
	// in the root directory.
	if err := fs.MkdirAll("templates", 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create the templates directory")
	}

	for _, tpl := range templates {
		info, err := os.Stat(tpl)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read template %q", tpl)
		}
		b, err := os.ReadFile(tpl)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read template %q", tpl)
		}
		if err := util.WriteFile(fs, path.Join("templates", filepath.ToSlash(tpl)), b, info.Mode().Perm()); err != nil {
			return nil, errors.Wrapf(err, "failed to copy template %q", tpl)
		}
	}

	m := modules.NewWithFS(ctx, name, fs)
	if version != "" {
		m.Version = version
	}
	return m, nil
}
