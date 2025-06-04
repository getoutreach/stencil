// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements module specific code.

package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	giturls "github.com/chainguard-dev/git-urls"
	"github.com/getoutreach/gobox/pkg/cli/github"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// localModuleVersion is the version string used for local modules
const localModuleVersion = "local"

// Module is a stencil module that contains template files.
type Module struct {
	// t is a shared go-template that is used for this module. This is important
	// because this allows us to call shared templates across a single module.
	// Note: We don't currently support sharing templates across modules. Instead
	// the data passing system should be used for cases like this.
	t *template.Template

	// Name is the name of a module. This should be a valid go
	// import path. For example: github.com/getoutreach/stencil-base
	Name string

	// URI is the underlying URI being used to download this module
	URI string

	// Version is the version of this module
	Version string

	// fs is a cached filesystem
	fs billy.Filesystem
}

// uriIsLocal returns true if the URI is a local file path
func uriIsLocal(uri string) bool {
	return !strings.Contains(uri, "://") || strings.HasPrefix(uri, "file://")
}

// New creates a new module from a TemplateRepository. Version must be set and can
// be obtained via the gobox/pkg/cli/updater/resolver package, or by using the
// GetModulesForService function.
//
// uri is the URI for the module. If it is an empty string https://+name is used
// instead.
func New(ctx context.Context, uri string, tr *configuration.TemplateRepository) (*Module, error) {
	if uri == "" {
		uri = "https://" + tr.Name
	}

	// check if a url based on if :// is in the uri, this is kinda hacky
	// but the only way to do this with a URL+file path being allowed.
	// We also support the "older" file:// scheme.
	if uriIsLocal(uri) { // Assume it's a path.
		osPath := strings.TrimPrefix(uri, "file://")
		if _, err := os.Stat(osPath); err != nil {
			return nil, errors.Wrapf(err, "failed to find module %s at path %q", tr.Name, osPath)
		}

		// translate the path into a file:// URI
		uri = "file://" + osPath
		tr.Version = localModuleVersion
	}
	if tr.Version == "" {
		return nil, fmt.Errorf("version must be specified for module %q", tr.Name)
	}

	return &Module{template.New(tr.Name).Funcs(sprig.TxtFuncMap()), tr.Name, uri, tr.Version, nil}, nil
}

// NewWithFS creates a module with the specified file system. This is
// generally only meant to be used in tests.
func NewWithFS(ctx context.Context, name string, fs billy.Filesystem) *Module {
	//nolint:errcheck // Why: No errors
	m, _ := New(ctx, "vfs://"+name, &configuration.TemplateRepository{
		Name:    name,
		Version: "vfs",
	})
	m.fs = fs
	return m
}

// GetTemplate returns the go template for this module
func (m *Module) GetTemplate() *template.Template {
	return m.t
}

// RegisterExtensions registers all extensions provided
// by the given module. If the module is a local file
// URI then extensions will be sourced from the `./bin`
// directory of the base of the path.
func (m *Module) RegisterExtensions(ctx context.Context, log logrus.FieldLogger, ext *extensions.Host) error {
	mf, err := m.Manifest(ctx)
	if err != nil {
		return err
	}

	// Only register extensions if this repository declares extensions explicitly in its type.
	if !mf.Type.Contains(configuration.TemplateRepositoryTypeExt) {
		return nil
	}

	version := &resolver.Version{
		Tag: m.Version,
	}
	return ext.RegisterExtension(ctx, m.URI, m.Name, version)
}

// Manifest downloads the module if not already downloaded and returns a parsed
// configuration.TemplateRepositoryManifest of this module.
func (m *Module) Manifest(ctx context.Context) (configuration.TemplateRepositoryManifest, error) {
	fs, err := m.GetFS(ctx)
	if err != nil {
		return configuration.TemplateRepositoryManifest{}, errors.Wrap(err, "failed to download fs")
	}

	mf, err := fs.Open("manifest.yaml")
	if err != nil {
		return configuration.TemplateRepositoryManifest{}, err
	}
	defer mf.Close()

	var manifest configuration.TemplateRepositoryManifest
	if err := yaml.NewDecoder(mf).Decode(&manifest); err != nil {
		return configuration.TemplateRepositoryManifest{}, err
	}

	// ensure that the manifest name is equal to the import path
	if manifest.Name != m.Name {
		return configuration.TemplateRepositoryManifest{}, fmt.Errorf(
			"module declares its import path as %q but was imported as %q",
			manifest.Name, m.Name,
		)
	}

	return manifest, nil
}

// GetFS returns a billy.Filesystem that contains the contents of this module.
// If the module URI starts with file://, it uses the local filesystem at the given path.
// Otherwise, it clones the module from a remote git repository into a temporary cache directory
// on the OS filesystem. This allows the module to be used as a billy.Filesystem.
func (m *Module) GetFS(ctx context.Context) (billy.Filesystem, error) {
	if m.fs != nil {
		return m.fs, nil
	}

	u, err := giturls.Parse(m.URI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse module URI")
	}

	if u.Scheme == "file" {
		m.fs = osfs.New(strings.TrimPrefix(m.URI, "file://"))
		return m.fs, nil
	}

	cacheDir := filepath.Join(stencilCacheDir(), "module_fs", getModuleCacheDirectory(m.URI, m.Version))
	logrus.Println("cacheDir", cacheDir)

	if useModuleCache(cacheDir) {
		m.fs = osfs.New(cacheDir)
		return m.fs, nil
	}

	err = os.RemoveAll(cacheDir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to remove stale cache %q", cacheDir)
	}

	if err = os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "failed to create cache directory")
	}

	m.fs = osfs.New(cacheDir)
	opts := &git.CloneOptions{
		URL:               m.URI,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Depth:             1,
	}

	if token, err := github.GetToken(); err == nil {
		opts.Auth = &githttp.BasicAuth{
			Username: "x-access-token",
			Password: string(token),
		}
	} else {
		logrus.WithError(err).Warn("failed to get github token, will use an unauthenticated client")
	}

	if m.Version != "" {
		opts.ReferenceName = plumbing.NewTagReferenceName(m.Version)
		opts.SingleBranch = true
	}

	// We don't use the git object here because all we care about is
	// the underlying filesystem object, which was created earlier
	if _, err := git.CloneContext(ctx, memory.NewStorage(), m.fs, opts); err != nil {
		// if tag not found try as a branch
		if !errors.Is(err, git.NoMatchingRefSpecError{}) {
			return nil, err
		}

		opts.ReferenceName = plumbing.NewBranchReferenceName(m.Version)
		if _, err := git.CloneContext(ctx, memory.NewStorage(), m.fs, opts); err != nil {
			return nil, errors.Wrap(err, "failed to find version as branch/tag")
		}
	}

	return m.fs, nil
}

// useModuleCache returns true if the specified path should be used as a module cache
func useModuleCache(path string) bool {
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > 2*time.Minute {
		return false
	}

	return true
}

// getModuleCacheDirectory generates a directory name from the URI and branch
func getModuleCacheDirectory(uri, branch string) string {
	if branch == "" {
		branch = "v0.0.0"
	}

	return regexp.MustCompile(`[^a-zA-Z0-9@]+`).ReplaceAllString(uri+"@"+branch, "_")
}

func stencilCacheDir() string {
	return filepath.Join(os.TempDir(), "stencil_cache")
}
