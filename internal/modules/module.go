// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements module specific code.

package modules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	giturls "github.com/chainguard-dev/git-urls"
	"github.com/getoutreach/gobox/pkg/cli/github"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	stgit "github.com/getoutreach/stencil/internal/git"
	"github.com/getoutreach/stencil/internal/retry"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/gofrs/flock"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.yaml.in/yaml/v3"
)

// localModuleVersion is the version string used for local modules.
const localModuleVersion = "local"

// ErrVersionRequired indicates a module was declared without a required version.
var ErrVersionRequired = errors.New("version must be specified for module")

// ErrImportPathMismatch indicates a module's declared import path does not match
// the path it was imported as.
var ErrImportPathMismatch = errors.New("module declares a different import path than it was imported as")

// ModuleCacheTTL defines the time-to-live duration for the module cache.
const ModuleCacheTTL = 30 * time.Minute

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

	// fsMu guards fs and serializes clones into the module cache directory,
	// because each clone attempt first removes that directory.
	fsMu sync.Mutex

	// fs is a cached filesystem
	fs billy.Filesystem

	// manifestMu guards manifest and serializes the first load of it.
	manifestMu sync.Mutex

	// manifest is the parsed manifest.yaml of this module. It is set by the
	// first successful load and is never returned outside this package.
	manifest *configuration.TemplateRepositoryManifest
}

// uriIsLocal returns true if the URI is a local file path.
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
		return nil, fmt.Errorf("%w: %q", ErrVersionRequired, tr.Name)
	}

	return &Module{
		t:       template.New(tr.Name).Funcs(sprig.TxtFuncMap()),
		Name:    tr.Name,
		URI:     uri,
		Version: tr.Version,
	}, nil
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

// GetTemplate returns the go template for this module.
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
//
// The manifest is read once per Module and then cached, so edits to a local
// module's manifest.yaml are not seen by an existing Module. Errors are not
// cached. Each call returns a copy that callers may modify, except for the
// slice inside Type, which is shared. It is safe for concurrent use.
func (m *Module) Manifest(ctx context.Context) (configuration.TemplateRepositoryManifest, error) {
	cached, err := m.cachedManifest(ctx)
	if err != nil {
		return configuration.TemplateRepositoryManifest{}, err
	}

	// The cached manifest is never modified once set, so it is copied
	// without holding the lock.
	return cloneManifest(cached), nil
}

// Argument returns a copy of the named argument declared in this module's
// manifest, and whether it is declared. It is cheaper than Manifest when only
// one argument is needed.
func (m *Module) Argument(ctx context.Context, name string) (configuration.Argument, bool, error) {
	cached, err := m.cachedManifest(ctx)
	if err != nil {
		return configuration.Argument{}, false, err
	}

	arg, ok := cached.Arguments[name]
	if !ok {
		return configuration.Argument{}, false, nil
	}

	return cloneArgument(&arg), true, nil
}

// ImportsModule returns whether this module's manifest lists the module with
// the given import path as a dependency.
func (m *Module) ImportsModule(ctx context.Context, importPath string) (bool, error) {
	cached, err := m.cachedManifest(ctx)
	if err != nil {
		return false, err
	}

	return slices.ContainsFunc(cached.Modules, func(tr *configuration.TemplateRepository) bool {
		return tr != nil && tr.Name == importPath
	}), nil
}

// GetFS returns a billy.Filesystem that contains the contents of this module.
// If the module URI starts with file://, it uses the local filesystem at the given path.
// Otherwise, it clones the module from a remote git repository into a temporary cache directory
// on the OS filesystem. This allows the module to be used as a billy.Filesystem.
func (m *Module) GetFS(ctx context.Context) (billy.Filesystem, error) {
	m.fsMu.Lock()
	defer m.fsMu.Unlock()

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

	cacheDir := m.FSCacheDir()

	if useCache(cacheDir) {
		m.fs = osfs.New(cacheDir)
		return m.fs, nil
	}

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

	fs, err := cloneModule(ctx, cacheDir, opts)
	if err != nil {
		// if tag not found try as a branch
		if !errors.Is(err, git.NoMatchingRefSpecError{}) {
			return nil, err
		}

		opts.ReferenceName = plumbing.NewBranchReferenceName(m.Version)
		if fs, err = cloneModule(ctx, cacheDir, opts); err != nil {
			return nil, errors.Wrap(err, "failed to find version as branch/tag")
		}
	}

	m.fs = fs
	return m.fs, nil
}

// cloneModule clones a module into cacheDir and returns the filesystem that
// holds it. It retries transient failures. See stgit.IsTransientError.
func cloneModule(ctx context.Context, cacheDir string, opts *git.CloneOptions) (billy.Filesystem, error) {
	fs := osfs.New(cacheDir)

	err := retry.Do(ctx, func(ctx context.Context) error {
		// A failed attempt can leave a partial checkout, so every attempt starts
		// from an empty directory. Failing to prepare the directory is not retried.
		if err := os.RemoveAll(cacheDir); err != nil {
			return retry.Permanent(errors.Wrapf(err, "failed to remove stale module cache directory %q", cacheDir))
		}
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return retry.Permanent(errors.Wrapf(err, "failed to create module cache directory %q", cacheDir))
		}

		// We don't use the git object here because all we care about is
		// the underlying filesystem object.
		_, err := git.CloneContext(ctx, memory.NewStorage(), fs, opts)
		return stgit.RedactError(err)
	},
		retry.WithRetryable(stgit.IsTransientError),
		retry.WithLogging(logrus.StandardLogger(), "clone "+stgit.RedactURL(opts.URL)),
	)
	if err != nil {
		// Do not leave a partial checkout behind: useCache would take it for a
		// module that was cloned in full. The error that matters is the clone's.
		_ = os.RemoveAll(cacheDir)
		return nil, err
	}

	return fs, nil
}

func (m *Module) PathSlug() string {
	return PathSlug(m.URI, m.Version)
}

func (m *Module) FSCacheDir() string {
	return FSCacheDir(m.PathSlug())
}

// cachedManifest returns the cached manifest, loading it first if needed.
// The lock is held during the load so that concurrent callers do not parse
// the same manifest more than once. The result must not be modified.
func (m *Module) cachedManifest(ctx context.Context) (*configuration.TemplateRepositoryManifest, error) {
	m.manifestMu.Lock()
	defer m.manifestMu.Unlock()

	if m.manifest == nil {
		manifest, err := m.loadManifest(ctx)
		if err != nil {
			return nil, err
		}
		m.manifest = &manifest
	}

	return m.manifest, nil
}

// loadManifest downloads the module if not already downloaded and parses its
// manifest.yaml, without consulting the cache.
func (m *Module) loadManifest(ctx context.Context) (configuration.TemplateRepositoryManifest, error) {
	lockDir := FSLockDir(m.PathSlug())
	lock, err := exclusiveLockDirectory(ctx, lockDir)
	if err != nil {
		return configuration.TemplateRepositoryManifest{},
			errors.Wrapf(err, "failed to lock module cache directory %q", lockDir)
	}

	//nolint:errcheck // Why: Unlock error can be safely ignored here
	defer lock.Unlock()

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
			"%w: declared %q but imported as %q",
			ErrImportPathMismatch, manifest.Name, m.Name,
		)
	}

	return manifest, nil
}

// lockRetryDelay is how long exclusiveLockDirectory waits between attempts to take
// the lock.
const lockRetryDelay = 100 * time.Millisecond

// exclusiveLockDirectory creates a new flock lock for the specified directory. It
// waits for the lock if someone else holds it, until ctx is done.
func exclusiveLockDirectory(ctx context.Context, dir string) (*flock.Flock, error) {
	// flock creates the lock file, but not its directory.
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Wrapf(err, "failed to create directory %q", dir)
	}

	lock := flock.New(filepath.Join(dir, "ex_dir.lock"))
	if _, err := lock.TryLockContext(ctx, lockRetryDelay); err != nil {
		return nil, err
	}

	return lock, nil
}

// useCache determines if the specified path should be used as a module cache.
func useCache(path string) bool {
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > ModuleCacheTTL {
		return false
	}

	if files, err := os.ReadDir(path); err != nil || len(files) == 0 {
		return false
	}

	return true
}

// PathSlug returns a unique identifier for a module
// using the given URI and versioning constraints.
func PathSlug(uri, version string) string {
	if version == "" {
		version = "v0.0.0"
	}

	return regexp.MustCompile(`[^a-zA-Z0-9<>=.\[\]\-@]+`).ReplaceAllString(uri+"@"+version, "_")
}

// StencilCacheDir returns the directory where stencil caches its data.
func StencilCacheDir() string {
	return filepath.Join(os.TempDir(), "stencil_cache")
}

// CacheDir returns the cache directory for a module based on its type and ID.
func CacheDir(cacheType, moduleID string) string {
	return filepath.Join(StencilCacheDir(), cacheType, moduleID)
}

// FSCacheDir returns the cache directory for a module based on its ID.
func FSCacheDir(moduleID string) string {
	return CacheDir("module_fs", moduleID)
}

// VersionCacheDir returns the version cache directory for a module based on its ID.
func VersionCacheDir(moduleID string) string {
	return CacheDir("module_version", moduleID)
}

// VersionLockDir returns the lock directory for a module version based on its ID.
func VersionLockDir(moduleID string) string {
	return CacheDir("module_version_lock", moduleID)
}

// FSLockDir returns the lock directory for a module filesystem based on its ID.
func FSLockDir(moduleID string) string {
	return CacheDir("module_fs_lock", moduleID)
}
