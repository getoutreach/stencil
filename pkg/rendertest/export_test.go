// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file gives the tests of the package access to the
// snapshot mode.

package rendertest

import "testing"

// SetUpdateSnapshots sets the update mode of snapshots until the test ends. Do
// not call it from a parallel test.
func SetUpdateSnapshots(tb testing.TB, update bool) {
	tb.Helper()

	previous := updateSnapshots
	updateSnapshots = update
	tb.Cleanup(func() { updateSnapshots = previous })
}

// UpdateSnapshots returns whether snapshots are written instead of compared.
func UpdateSnapshots() bool {
	return updateSnapshots
}
