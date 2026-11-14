// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file makes Snapshot write snapshots instead of
// comparing them. It is built only with the rendertest_update_snapshots
// build tag.

//go:build rendertest_update_snapshots

package rendertest

func init() {
	updateSnapshots = true
}
