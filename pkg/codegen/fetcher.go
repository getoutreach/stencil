// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the template repository fetching logic

package codegen

import (
	"context"

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/stencil/internal/vfs"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/getoutreach/stencil/pkg/gitauth"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
	"gopkg.in/yaml.v3"
)

// Fetcher handles all interactions between dependencies
// and fetching of dependencies for stencil. This involes
// creating a vfs, parsing repository manifests, etc.
type Fetcher struct {
	log         logrus.FieldLogger
	m           *configuration.ServiceManifest
	accessToken cfg.SecretData
	extensions  *extensions.Host
}

// NewFetcher creates a new fetcher instance
func NewFetcher(log logrus.FieldLogger, m *configuration.ServiceManifest, accessToken cfg.SecretData,
	extHost *extensions.Host) *Fetcher {
	return &Fetcher{log, m, accessToken, extHost}
}

// DownloadRepository downloads a remote repository into memory based
// on the URL provided. If this is a file:// repository fs.New is used
// instead to "chroot" it.
func (f *Fetcher) DownloadRepository(ctx context.Context, r *configuration.TemplateRepository) (billy.Filesystem, error) {
	u, err := giturls.Parse(r.URL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}

	var fs billy.Filesystem

	f.log.Infof("- Using repository '%s'", r.URL)
	if u.Scheme == "file" { // Support local paths
		fs = osfs.New(u.Path)
	} else {
		fs = memfs.New()
		opts := &git.CloneOptions{
			URL:               r.URL,
			RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			Depth:             1,
		}
		if err := gitauth.ConfigureAuth(f.accessToken, opts, f.log); err != nil {
			return nil, errors.Wrap(err, "failed to setup git authentication")
		}

		if r.Version != "" {
			opts.ReferenceName = plumbing.NewTagReferenceName(r.Version)
			opts.SingleBranch = true
		}

		// We don't use the git object here because all we care about is
		// the underlying filesystem object, which was created earlier
		if _, err := git.CloneContext(ctx, memory.NewStorage(), fs, opts); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// ParseRepositoryManifest parses a remote repository manifest
func (f *Fetcher) ParseRepositoryManifest(fs billy.Filesystem) (*configuration.TemplateRepositoryManifest, error) {
	mf, err := fs.Open("manifest.yaml")
	if err != nil {
		return nil, err
	}
	defer mf.Close()

	var manifest *configuration.TemplateRepositoryManifest
	err = yaml.NewDecoder(mf).Decode(&manifest)
	return manifest, err
}

// ResolveDependencies resolved the dependencies of a given template repository.
// It currently only supports one level dependency resolution and doesn't do any
// smart logic for ordering other than first wins.
func (f *Fetcher) ResolveDependencies(ctx context.Context, filesystems map[string]bool,
	r *configuration.TemplateRepositoryManifest) ([]billy.Filesystem, error) { //nolint:funlen // Why: will refactor later
	depFilesystems := make([]billy.Filesystem, 0)
	args := make(map[string]configuration.Argument)
	for _, d := range r.Modules {
		// If the filesystem already exists, then we can just skip it
		// since something already required it.
		if _, ok := filesystems[d.URL]; ok {
			continue
		}

		fs, err := f.DownloadRepository(ctx, d)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download repository '%s'", d.URL)
		}
		filesystems[d.URL] = true

		mf, err := f.ParseRepositoryManifest(fs)
		if err != nil {
			return nil, err
		}

		// register an extension if it's found
		if mf.Type == configuration.TemplateRepositoryTypeExt {
			//nolint:govet // Why: We're OK shadowing err
			err := f.extensions.RegisterExtension(ctx, d.URL, mf.Name, d.Version)
			if err != nil {
				return nil, errors.Wrap(err, "failed to download extension")
			}
		}

		for k, v := range mf.Arguments {
			args[k] = v
		}

		subDepFilesystems, err := f.ResolveDependencies(ctx, filesystems, mf)
		if err != nil {
			return nil, err
		}

		newFileSystems := make([]billy.Filesystem, 0)

		// only add our filesystem if we're not an extension
		if mf.Type != configuration.TemplateRepositoryTypeExt {
			newFileSystems = append(newFileSystems, fs)
		}

		// lookup our dependencies after ours so that we're able to override their files
		// I suspect that this will need to be changed to allow modules to selectively
		// override certain files.
		if len(subDepFilesystems) > 0 {
			newFileSystems = append(newFileSystems, subDepFilesystems...)
		}

		// if we have new filesystems, be sure to append them after our dependencies
		if len(newFileSystems) > 0 {
			depFilesystems = append(depFilesystems, newFileSystems...)
		}
	}

	return depFilesystems, nil
}

// CreateVFS creates a new virtual file system with the dependencies
// of the top level (service) module inside of it. This filesystem is
// ordered as such that the top level modules always go first, with
// the order in that list being preserved. After a VFS is created,
// all the manifests from said repositories are then parsed and
// returned to the caller in the order that they were layered.
func (f *Fetcher) CreateVFS(ctx context.Context) (billy.Filesystem,
	[]*configuration.TemplateRepositoryManifest, error) {
	// Create a shim template manifest from our service dependencies
	layers, err := f.ResolveDependencies(ctx, make(map[string]bool), &configuration.TemplateRepositoryManifest{
		Modules: f.m.Modules,
	})
	if err != nil {
		return nil, nil, err
	}

	manifests := make([]*configuration.TemplateRepositoryManifest, len(layers))
	for i, fs := range layers {
		m, err := f.ParseRepositoryManifest(fs)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to parse manifest")
		}

		manifests[i] = m
	}

	return vfs.NewLayeredFS(layers...), manifests, nil
}
