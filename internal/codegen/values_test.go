// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the values file

package codegen

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"gotest.tools/v3/assert"
)

func TestValues(t *testing.T) {
	tmpDir, err := os.MkdirTemp(t.TempDir(), "stencil-values-test")
	assert.NilError(t, err, "expected os.MkdirTemp() not to fail")

	wd, err := os.Getwd()
	assert.NilError(t, err, "expected os.Getwd() not to fail")

	// Change directory to the temporary directory, and restore the original
	// working directory when we're done.
	os.Chdir(tmpDir)
	defer func() { os.Chdir(wd) }()

	// Clone from an upstream repository so that the default branch can be
	// determined from the "origin" remote.
	upstream, err := gogit.PlainInitWithOptions(t.TempDir(), &gogit.PlainInitOptions{
		InitOptions: gogit.InitOptions{DefaultBranch: plumbing.Main},
	})
	assert.NilError(t, err, "expected gogit.PlainInitWithOptions() not to fail")

	upstreamWrk, err := upstream.Worktree()
	assert.NilError(t, err, "expected gogit.(Repository).Worktree() not to fail")

	cmt, err := upstreamWrk.Commit("initial commit", &gogit.CommitOptions{
		AllowEmptyCommits: true,
		Author: &object.Signature{
			Name:  "Stencil",
			Email: "email@example.com",
			When:  time.Now(),
		},
	})
	assert.NilError(t, err, "expected worktree.Commit() not to fail")

	_, err = gogit.PlainClone(tmpDir, false, &gogit.CloneOptions{
		URL: upstreamWrk.Filesystem.Root(),
	})
	assert.NilError(t, err, "expected gogit.PlainClone() not to fail")

	sm := &configuration.ServiceManifest{
		Name: "testing",
	}

	boxConf, _ := box.LoadBox()

	vals := NewValues(context.Background(), sm, []*modules.Module{
		{
			Name:    "testing",
			Version: "1.2.3",
		},
	}, logrus.New())
	assert.DeepEqual(t, &Values{
		Git: git{
			Ref:    plumbing.NewBranchReferenceName("main").String(),
			Commit: cmt.String(),
			Dirty:  false,
		},
		Runtime: runtime{
			Generator:        app.Info().Name,
			GeneratorVersion: app.Info().Version,
			Box:              boxConf,
			Modules: modulesSlice{
				{
					Name:    "testing",
					Version: "1.2.3",
				},
			},
		},
		Config: config{
			Name: sm.Name,
		},
	}, vals, cmpopts.IgnoreUnexported(git{}))

	branch, err := vals.Git.DefaultBranch()
	assert.NilError(t, err, "expected DefaultBranch() not to fail in a git repository")
	assert.Equal(t, branch, "main")
}

// gitInDir runs git with args in dir and fails the test on error.
func gitInDir(t *testing.T, dir string, args ...string) {
	t.Helper()
	out, err := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	assert.NilError(t, err, string(out))
}

func TestDefaultBranchWithoutOpenableRepository(t *testing.T) {
	log := logrus.New()
	man := &configuration.TemplateRepositoryManifest{Name: "testing"}
	m, err := modulestest.NewModuleFromTemplates(man, "testdata/values/default-branch.tpl")
	assert.NilError(t, err, "failed to create module")

	tests := map[string]struct {
		// setup returns the directory stencil runs in.
		setup   func(t *testing.T) string
		wantErr string
	}{
		"not a git repository": {
			setup: func(t *testing.T) string {
				t.Helper()
				return t.TempDir()
			},
			wantErr: "repository does not exist",
		},
		"unsupported git extension": {
			setup: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				gitInDir(t, dir, "init", "-q")
				gitInDir(t, dir, "config", "core.repositoryformatversion", "1")
				gitInDir(t, dir, "config", "extensions.worktreeConfig", "true")
				return dir
			},
			wantErr: "does not support extension",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Chdir(tt.setup(t))

			st := NewStencil(&configuration.ServiceManifest{
				Name:      "testing",
				Arguments: map[string]any{},
			}, []*modules.Module{m}, log)
			_, err := st.Render(t.Context(), log)
			assert.ErrorContains(t, err, "failed to open git repository")
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestDefaultBranchWithoutOriginRemoteIsNotRetried(t *testing.T) {
	dir := t.TempDir()
	gitInDir(t, dir, "init", "-q")
	t.Chdir(dir)

	log, hook := logtest.NewNullLogger()
	vals := NewValues(t.Context(), &configuration.ServiceManifest{Name: "testing"}, nil, log)

	_, err := vals.Git.DefaultBranch()
	assert.ErrorContains(t, err, `"origin" remote`)
	assert.Equal(t, len(hook.AllEntries()), 0, "expected no retries to be logged")
}

func TestDefaultBranchIsLookedUpOnFirstUseOnly(t *testing.T) {
	lookups := 0
	vals := &Values{Git: git{defaultBranch: sync.OnceValues(func() (string, error) {
		lookups++
		return "trunk", nil
	})}}
	assert.Equal(t, lookups, 0, "expected no lookup before the first use")

	// Copies of Values, such as the one made for each template, share one lookup.
	for _, v := range []*Values{vals, vals.WithModule("m", "1.0.0"), vals.WithTemplate("t.tpl")} {
		branch, err := v.Git.DefaultBranch()
		assert.NilError(t, err)
		assert.Equal(t, branch, "trunk")
	}
	assert.Equal(t, lookups, 1)
}

func TestNewValuesDoesNotLookUpTheDefaultBranch(t *testing.T) {
	// An unreachable origin would make an eager lookup fail, and log a retry.
	dir := t.TempDir()
	gitInDir(t, dir, "init", "-q")
	gitInDir(t, dir, "remote", "add", "origin", filepath.Join(t.TempDir(), "missing"))
	t.Chdir(dir)

	log, hook := logtest.NewNullLogger()
	NewValues(t.Context(), &configuration.ServiceManifest{Name: "testing"}, nil, log)
	assert.Equal(t, len(hook.AllEntries()), 0, "expected no lookup until a template reads the default branch")
}

func TestDefaultBranchIsReadFromOrigin(t *testing.T) {
	upstream := t.TempDir()
	gitInDir(t, upstream, "init", "-q", "-b", "trunk")
	gitInDir(t, upstream, "-c", "user.name=Stencil", "-c", "user.email=email@example.com",
		"commit", "-q", "--allow-empty", "-m", "initial commit")

	parent := t.TempDir()
	gitInDir(t, parent, "clone", "-q", upstream, "clone")
	t.Chdir(filepath.Join(parent, "clone"))

	vals := NewValues(t.Context(), &configuration.ServiceManifest{Name: "testing"}, nil, logrus.New())
	branch, err := vals.Git.DefaultBranch()
	assert.NilError(t, err)
	assert.Equal(t, branch, "trunk")
}
