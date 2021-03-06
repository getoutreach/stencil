// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the values file

package codegen

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/stencil/pkg/configuration"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
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

	r, err := gogit.PlainInit(tmpDir, false)
	assert.NilError(t, err, "expected gogit.PlainInit() not to fail")

	wrk, err := r.Worktree()
	assert.NilError(t, err, "expected gogit.(Repository).Worktree() not to fail")

	cmt, err := wrk.Commit("initial commit", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "Stencil",
			Email: "email@example.com",
			When:  time.Now(),
		},
	})
	assert.NilError(t, err, "expected worktree.Commit() not to fail")

	err = wrk.Checkout(&gogit.CheckoutOptions{
		Create: true,
		Branch: plumbing.NewBranchReferenceName("main"),
	})
	assert.NilError(t, err, "expected worktree.Checkout() not to fail")

	sm := &configuration.ServiceManifest{
		Name: "testing",
	}

	boxConf, _ := box.LoadBox()

	vals := NewValues(context.Background(), sm)
	assert.DeepEqual(t, &Values{
		Git: &git{
			Ref:           plumbing.NewBranchReferenceName("main").String(),
			Commit:        cmt.String(),
			Dirty:         false,
			DefaultBranch: "main",
		},
		Runtime: &runtime{
			Generator:        app.Info().Name,
			GeneratorVersion: app.Info().Version,
			Box:              boxConf,
		},
		Config: &config{
			Name: sm.Name,
		},
	}, vals)
}
