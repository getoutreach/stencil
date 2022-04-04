// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements module specific code.

package modules

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/getoutreach/gobox/pkg/github"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	giturls "github.com/whilp/git-urls"
	"gopkg.in/yaml.v2"
)

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

// getLatestVersion returns the latest version of a git repository
// name should be a valid go import path, like github.com/<org>/<repo>
func getLatestVersion(ctx context.Context, name string) (string, error) {
	paths := strings.Split(name, "/")

	// github.com, getoutreach, stencil-base
	if len(paths) < 3 {
		return "", fmt.Errorf("invalid module path %q, expected github.com/<org>/<repo>", name)
	}

	gh, err := github.NewClient()
	if err != nil {
		return "", err
	}

	rel, _, err := gh.Repositories.GetLatestRelease(ctx, paths[1], paths[2])
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the latest release for module %q", name)
	}

	return rel.GetTagName(), nil
}

// New creates a new module at a specific revision. If a revision is
// not provided then the latest version is automatically used. A revision
// must be a valid git ref of semantic version. In the case of a semantic
// version, a range is also acceptable, e.g. ~1.0.0.
//
// If the uri is a file:// path, a version must be specified otherwise
// it will be treated as if it has no version. If uri is not specified
// it is default to HTTPS
func New(ctx context.Context, name, uri, version string) (*Module, error) {
	if uri == "" {
		uri = "https://" + name
	} else if strings.HasPrefix(uri, "file://") {
		version = "local-" + strings.TrimPrefix(uri, "file://")
	}

	if version == "" {
		var err error
		version, err = getLatestVersion(ctx, name)
		if err != nil {
			return nil, err
		}
	}

	return &Module{template.New(name).Funcs(sprig.TxtFuncMap()), name, uri, version, nil}, nil
}

// NewWithFS creates a module with the specified file system. This is
// generally only meant to be used in tests.
func NewWithFS(ctx context.Context, name string, fs billy.Filesystem) *Module {
	m, _ := New(ctx, name, "vfs://"+name, "vfs") //nolint:errcheck // Why: No errors
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
func (m *Module) RegisterExtensions(ctx context.Context, ext *extensions.Host) error {
	mf, err := m.Manifest(ctx)
	if err != nil {
		return err
	}

	// Only register extensions if we're a extension repository
	if mf.Type != configuration.TemplateRepositoryTypeExt {
		return nil
	}

	return ext.RegisterExtension(ctx, m.Name, m.Name, m.Version)
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

// GetFS returns a billy.Filesystem that contains the contents
// of this module.
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

	token, err := github.GetToken()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get github token")
	}

	m.fs = memfs.New()
	opts := &git.CloneOptions{
		URL:               m.URI,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
		Depth:             1,
		Auth: &githttp.BasicAuth{
			Username: "x-access-token",
			Password: string(token),
		},
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
