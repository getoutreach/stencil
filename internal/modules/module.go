// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements module specific code.

package modules

import (
	"context"
	"strings"

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
	// URI is a URI to the module. Currently supported formats are:
	//   https://<url>
	//   file://<path>
	URI string

	// Version is the version of this module
	Version string

	// fs is a cached filesystem
	fs billy.Filesystem
}

// getLatestVersion returns the latest version of a git repository
// uri should be a valid git URI, e.g. https://github.com/<org>/<repo>
func getLatestVersion(ctx context.Context, uri string) (string, error) {
	u, err := giturls.Parse(uri)
	if err != nil {
		return "", err
	}

	// file paths don't have a version
	if u.Scheme == "file" {
		return "", nil
	}

	paths := strings.Split(u.Path, "/")
	gh, err := github.NewClient()
	if err != nil {
		return "", err
	}

	rel, _, err := gh.Repositories.GetLatestRelease(ctx, paths[0], paths[1])
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the latest release for repo %q", u.Path)
	}

	return rel.GetTagName(), nil
}

// New creates a new module at a specific revision. If a revision is
// not provided then the latest version is automatically used. A revision
// must be a valid git ref of semantic version. In the case of a semantic
// version, a range is also acceptable, e.g. ~1.0.0.
//
// If the uri is a file:// path, a version must be specified otherwise
// it will be treated as if it has no version.
func New(uri, version string) (*Module, error) {
	if version == "" {
		var err error
		version, err = getLatestVersion(context.TODO(), uri)
		if err != nil {
			return nil, err
		}
	}

	return &Module{uri, version, nil}, nil
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

	return ext.RegisterExtension(ctx, m.URI, mf.Name, m.Version)
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
	err = yaml.NewDecoder(mf).Decode(&manifest)
	return manifest, err
}

// GetFS returns a billy.Filesystem that contains the contents
// of this module.
func (m *Module) GetFS(ctx context.Context) (billy.Filesystem, error) {
	if m.fs != nil {
		return m.fs, nil
	}

	u, err := giturls.Parse(m.URI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}

	if u.Scheme == "file" { // Support local paths
		m.fs = osfs.New(u.Path)
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
