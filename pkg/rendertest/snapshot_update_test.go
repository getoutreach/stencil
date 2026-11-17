// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the test of the build tag that turns on the
// update mode of snapshots. It is built only with the
// rendertest_update_snapshots build tag.

//go:build rendertest_update_snapshots

package rendertest_test

import (
	"testing"

	"github.com/getoutreach/stencil/pkg/rendertest"
	"gotest.tools/v3/assert"
)

func TestUpdateTagTurnsOnUpdateMode(t *testing.T) {
	assert.Assert(t, rendertest.UpdateSnapshots())
}
