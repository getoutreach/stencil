package codegen

import (
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/stencil/internal/vfs"
	"github.com/getoutreach/stencil/pkg/configuration"
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

type Fetcher struct {
	log logrus.FieldLogger
	m   *configuration.ServiceManifest

	sshKeyPath  string
	accessToken cfg.SecretData
}

func NewFetcher(log logrus.FieldLogger, m *configuration.ServiceManifest, sshKeyPath string, accessToken cfg.SecretData) *Fetcher {
	return &Fetcher{log, m, sshKeyPath, accessToken}
}

func (f *Fetcher) DownloadRepository(r *configuration.TemplateRepository) (billy.Filesystem, error) {
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
		if err := gitauth.ConfigureAuth(f.sshKeyPath, f.accessToken, opts, f.log); err != nil {
			return nil, errors.Wrap(err, "failed to setup git authentication")
		}

		if r.Version != "" {
			opts.ReferenceName = plumbing.NewTagReferenceName(r.Version)
			opts.SingleBranch = true
		}

		if _, err := git.Clone(memory.NewStorage(), fs, opts); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

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
func (f *Fetcher) ResolveDependencies(filesystems map[string]bool,
	r *configuration.TemplateRepositoryManifest) ([]billy.Filesystem, error) {
	depFilesystems := make([]billy.Filesystem, 0)
	args := make(map[string]configuration.Argument)
	for _, d := range r.Modules {
		// If the filesystem already exists, then we can just skip it
		// since something already required it.
		if _, ok := filesystems[d.URL]; ok {
			continue
		}

		fs, err := f.DownloadRepository(d)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download repository '%s'", d.URL)
		}
		filesystems[d.URL] = true

		mf, err := f.ParseRepositoryManifest(fs)
		if err != nil {
			return nil, err
		}
		for k, v := range mf.Arguments {
			args[k] = v
		}

		subDepFilesystems, err := f.ResolveDependencies(filesystems, mf)
		if err != nil {
			return nil, err
		}

		// append the resolved dependencies of the sub-dependencies to the array of dependencies
		// of the manifest we're operating on. Be sure to put the filesystem of this dependency after
		// it's sub-dependencies.
		depFilesystems = append(depFilesystems, append(subDepFilesystems, fs)...)
	}

	return depFilesystems, nil
}

func (f *Fetcher) CreateVFS() (billy.Filesystem, []*configuration.TemplateRepositoryManifest, error) {
	// Create a shim template manifest from our service dependencies
	layers, err := f.ResolveDependencies(make(map[string]bool), &configuration.TemplateRepositoryManifest{
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
