// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for cloning a module, including retries.

package modules_test

import (
	"bytes"
	"net/http"
	"net/http/cgi"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

// runGit runs git with args in dir and fails the test if it fails.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{
		"-c", "user.name=test", "-c", "user.email=test@example.com",
		"-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false",
	}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	assert.NilError(t, err, "git %s failed: %s", strings.Join(args, " "), out)
}

// gitServer serves a git repository, named repo.git, over HTTP.
type gitServer struct {
	base, URL string

	// infoRefs counts the requests for the repository's refs. Every clone attempt
	// makes one.
	infoRefs atomic.Int32
}

// newGitServer serves a repository that has the file manifest.yaml, the tag v1.0.0
// and the branch main. It answers the first failFirst clone attempts with a 503.
func newGitServer(t *testing.T, failFirst int32) *gitServer {
	t.Helper()

	execPath, err := exec.CommandContext(t.Context(), "git", "--exec-path").Output()
	assert.NilError(t, err)
	backendPath := filepath.Join(strings.TrimSpace(string(execPath)), "git-http-backend")
	if _, err := os.Stat(backendPath); err != nil {
		t.Skipf("git-http-backend is not installed: %v", err)
	}

	work := filepath.Join(t.TempDir(), "work")
	assert.NilError(t, os.Mkdir(work, 0o755))
	runGit(t, work, "init", "-b", "main")
	assert.NilError(t, os.WriteFile(filepath.Join(work, "manifest.yaml"), []byte("name: example\n"), 0o644))
	runGit(t, work, "add", ".")
	runGit(t, work, "commit", "-m", "init")
	runGit(t, work, "tag", "v1.0.0")

	root := t.TempDir()
	runGit(t, root, "clone", "--bare", work, "repo.git")

	backend := &cgi.Handler{
		Path:       backendPath,
		Env:        []string{"GIT_PROJECT_ROOT=" + root, "GIT_HTTP_EXPORT_ALL=1"},
		InheritEnv: []string{"PATH"},
	}

	srv := &gitServer{}
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Every clone attempt starts with this request, so failing it fails the attempt.
		if strings.HasSuffix(r.URL.Path, "/info/refs") && srv.infoRefs.Add(1) <= failFirst {
			http.Error(w, "try again later", http.StatusServiceUnavailable)
			return
		}
		backend.ServeHTTP(w, r)
	}))
	t.Cleanup(httpServer.Close)

	srv.base = httpServer.URL
	srv.URL = srv.base + "/repo.git"
	return srv
}

// newClonedModule creates a module for url at version, and removes its cache before
// and after the test.
func newClonedModule(t *testing.T, url, version string) *modules.Module {
	t.Helper()

	cacheDir := modules.FSCacheDir(modules.PathSlug(url, version))
	assert.NilError(t, os.RemoveAll(cacheDir))
	t.Cleanup(func() { os.RemoveAll(cacheDir) })

	m, err := modules.New(t.Context(), url, &configuration.TemplateRepository{
		Name:    "example.test/repo",
		Version: version,
	})
	assert.NilError(t, err)
	return m
}

func TestGetFS_RetriesTransientCloneFailure(t *testing.T) {
	srv := newGitServer(t, 1)

	fs, err := newClonedModule(t, srv.URL, "v1.0.0").GetFS(t.Context())
	assert.NilError(t, err, "expected the second attempt to work")
	assertFSExists(t, fs)

	assert.Equal(t, srv.infoRefs.Load(), int32(2), "expected one failed attempt, then one that works")
}

func TestGetFS_DoesNotRetryMissingRepository(t *testing.T) {
	srv := newGitServer(t, 0)

	_, err := newClonedModule(t, srv.base+"/missing.git", "v1.0.0").GetFS(t.Context())
	assert.ErrorContains(t, err, "repository not found")

	assert.Equal(t, srv.infoRefs.Load(), int32(1), "expected no retry, because asking again gives the same answer")
}

func TestGetFS_DoesNotRetryMissingVersion(t *testing.T) {
	srv := newGitServer(t, 0)
	m := newClonedModule(t, srv.URL, "does-not-exist")

	_, err := m.GetFS(t.Context())
	assert.ErrorContains(t, err, "failed to find version as branch/tag")
	assert.Equal(t, srv.infoRefs.Load(), int32(2), "expected one attempt as a tag and one as a branch, with no retries")

	_, err = m.GetFS(t.Context())
	assert.ErrorContains(t, err, "failed to find version as branch/tag",
		"expected a failed clone not to leave an empty file system in the module")

	_, err = os.Stat(modules.FSCacheDir(modules.PathSlug(srv.URL, "does-not-exist")))
	assert.ErrorIs(t, err, os.ErrNotExist, "expected a failed clone not to leave a cache directory")
}

func TestGetFS_DoesNotLogCredentials(t *testing.T) {
	srv := newGitServer(t, 1)
	var logged bytes.Buffer
	logrus.StandardLogger().SetOutput(&logged)
	t.Cleanup(func() { logrus.StandardLogger().SetOutput(os.Stderr) })

	url := strings.Replace(srv.URL, "http://", "http://ghp_secret@", 1)
	_, err := newClonedModule(t, url, "v1.0.0").GetFS(t.Context())
	assert.NilError(t, err)

	assert.Assert(t, strings.Contains(logged.String(), "retrying"), "expected a warning for the retry")
	assert.Assert(t, !strings.Contains(logged.String(), "ghp_secret"), "the log has the token:\n%s", &logged)
}
