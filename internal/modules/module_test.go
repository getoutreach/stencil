// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the modules package

package modules_test

import (
	"context"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"gotest.tools/v3/assert"
)

func TestCanFetchModule(t *testing.T) {
	m, err := modules.New("git@github.com:getoutreach/stencil-base", "main")
	assert.NilError(t, err, "failed to call New()")

	manifest, err := m.Manifest(context.Background())
	assert.NilError(t, err, "failed to call Manifest() on module")
	assert.Equal(t, manifest.Name, "stencil-base", "failed to validate returned manifest")

	fs, err := m.GetFS(context.Background())
	assert.NilError(t, err, "failed to call GetFS() on module")

	_, err = fs.Stat("manifest.yaml")
	assert.NilError(t, err, "failed to validate returned manifest from fs")
}

func TestCanGetLatestModule(t *testing.T) {
	_, err := modules.New("git@github.com:getoutreach/stencil-base", "")
	assert.NilError(t, err, "failed to call New()")
}
