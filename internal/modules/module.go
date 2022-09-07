// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements module specific code.

package modules

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/getoutreach/gobox/pkg/cli/github"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	gogithub "github.com/google/go-github/v47/github"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
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

// getLatestVersion returns the latest version of a git repository
// name should be a valid go import path, like github.com/<org>/<repo>
func getLatestVersion(ctx context.Context, tr *configuration.TemplateRepository) (string, error) {
	paths := strings.Split(tr.Name, "/")

	// github.com, getoutreach, stencil-base
	if len(paths) < 3 {
		return "", fmt.Errorf("invalid module path %q, expected github.com/<org>/<repo>", tr.Name)
	}

	gh, err := github.NewClient(github.WithAllowUnauthenticated())
	if err != nil {
		return "", err
	}

	if tr.Prerelease { // Use the newest, first, release.
		rels, _, err := gh.Repositories.ListReleases(ctx, paths[1], paths[2], &gogithub.ListOptions{
			PerPage: 10,
		})
		if err != nil {
			return "", errors.Wrapf(err, "failed to get releases for module %q", tr.Name)
		}

		// HACK(jaredallard): find the first release that has -rc in it,
		// will be fixed in version resolver rewrite that has channel support
		// like the updater.
		for _, rel := range rels {
			// ensure release is an rc release
			if !strings.Contains(rel.GetTagName(), "-rc") {
				continue
			}

			return rel.GetTagName(), nil
		}

		// if we didn't find one, then just use the latest release below
	}

	// Use GetLatestRelease() to ensure it's the latest _released_ version.
	rel, _, err := gh.Repositories.GetLatestRelease(ctx, paths[1], paths[2])
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the latest release for module %q", tr.Name)
	}

	return rel.GetTagName(), nil
}

// New creates a new module from a TemplateRepository. If no version is specified
// then the latest released revision is used. If `prerelease` is set to true then
// the latest revision is used, regardless of whether it is a released version or
// not.
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
	if !strings.Contains(uri, "://") || strings.HasPrefix(uri, "file://") { // Assume it's a path.
		osPath := strings.TrimPrefix(uri, "file://")
		if _, err := os.Stat(osPath); err != nil {
			return nil, errors.Wrapf(err, "failed to find module %s at path %q", tr.Name, osPath)
		}

		// translate the path into a file:// URI
		uri = "file://" + osPath
		tr.Version = localModuleVersion
	}

	if tr.Version == "" {
		var err error
		tr.Version, err = getLatestVersion(ctx, tr)
		if err != nil {
			return nil, err
		}
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

	return ext.RegisterExtension(ctx, m.URI, m.Name, m.Version)
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

	m.fs = memfs.New()
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
