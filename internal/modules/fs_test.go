package modules

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5"
	"github.com/stretchr/testify/require"
)

func TestGetFS_CacheUsage(t *testing.T) {
	ctx := context.Background()
	// Use a public repo for testing, or mock a repo if needed.
	repoName := "github.com/getoutreach/stencil-base"
	repoURL := "https://" + repoName
	version := "main" // Use a branch/tag that exists

	tr := &configuration.TemplateRepository{
		Name:    repoName,
		Version: version,
	}

	// Clean up cache before test
	cacheDir := filepath.Join(StencilCacheDir(), "module_fs", ModuleCacheDirectory(repoURL, version))
	_ = os.RemoveAll(cacheDir)

	// First call: should clone and create cache
	mod, err := New(ctx, repoURL, tr)
	require.NoError(t, err)
	fs1, err := mod.GetFS(ctx)
	require.NoError(t, err)
	assertFSExists(t, fs1)

	// Cache should now exist and be fresh
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	require.WithinDuration(t, time.Now(), info.ModTime(), 2*time.Minute)

	// Second call: should use cache, not re-clone
	mod2, err := New(ctx, repoURL, tr)
	require.NoError(t, err)
	fs2, err := mod2.GetFS(ctx)
	require.NoError(t, err)
	assertFSExists(t, fs2)
}

func assertFSExists(t *testing.T, fs billy.Filesystem) {
	_, err := fs.Stat(".")
	require.NoError(t, err)
}
