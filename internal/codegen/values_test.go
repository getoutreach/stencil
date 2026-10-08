// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the values file

package codegen

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	stencilgit "github.com/getoutreach/stencil/internal/git"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
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
			Ref:           plumbing.NewBranchReferenceName("main").String(),
			Commit:        cmt.String(),
			Dirty:         false,
			defaultBranch: "main",
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
	}, vals, cmp.AllowUnexported(git{}))

	branch, err := vals.Git.DefaultBranch()
	assert.NilError(t, err, "expected DefaultBranch() not to fail in a git repository")
	assert.Equal(t, branch, "main")
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
				for _, args := range [][]string{
					{"init", "-q"},
					{"config", "core.repositoryformatversion", "1"},
					{"config", "extensions.worktreeConfig", "true"},
				} {
					out, err := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...).CombinedOutput()
					assert.NilError(t, err, string(out))
				}
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
			_, err := st.Render(context.Background(), log)
			assert.ErrorContains(t, err, "failed to open git repository")
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestGetDefaultBranchWithRetry(t *testing.T) {
	errBoom := errors.New("boom")
	log := logrus.New()

	t.Run("does not retry when git reports no branch", func(t *testing.T) {
		calls := 0
		_, err := getDefaultBranchWithRetry(context.Background(), log,
			func(context.Context, string) (string, error) {
				calls++
				return "", stencilgit.ErrNoRemoteHeadBranch
			}, time.Millisecond)
		assert.ErrorIs(t, err, stencilgit.ErrNoRemoteHeadBranch)
		assert.Equal(t, calls, 1)
	})

	t.Run("succeeds after a transient failure", func(t *testing.T) {
		calls := 0
		db, err := getDefaultBranchWithRetry(context.Background(), log,
			func(context.Context, string) (string, error) {
				calls++
				if calls < 3 {
					return "", errBoom
				}
				return "trunk", nil
			}, time.Millisecond)
		assert.NilError(t, err)
		assert.Equal(t, db, "trunk")
		assert.Equal(t, calls, 3)
	})

	t.Run("fails once retries are exhausted", func(t *testing.T) {
		calls := 0
		_, err := getDefaultBranchWithRetry(context.Background(), log,
			func(context.Context, string) (string, error) {
				calls++
				return "", errBoom
			}, time.Millisecond)
		assert.ErrorIs(t, err, errBoom)
		assert.Equal(t, calls, 3)
	})
}
