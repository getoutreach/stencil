// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for locking a directory.

package modules

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestExclusiveLockDirectoryCreatesTheDirectory(t *testing.T) {
	lock, err := exclusiveLockDirectory(t.Context(), filepath.Join(t.TempDir(), "missing", "dir"))
	assert.NilError(t, err)
	assert.NilError(t, lock.Unlock())
}

func TestExclusiveLockDirectoryWaitsForTheLock(t *testing.T) {
	dir := t.TempDir()
	held, err := exclusiveLockDirectory(t.Context(), dir)
	assert.NilError(t, err)

	// Longer than the delay between attempts to take the lock, so that there is a retry.
	const holdFor = 150 * time.Millisecond
	time.AfterFunc(holdFor, func() { _ = held.Unlock() })

	start := time.Now()
	lock, err := exclusiveLockDirectory(t.Context(), dir)
	assert.NilError(t, err)
	assert.Assert(t, time.Since(start) >= holdFor/2, "expected to wait for the holder of the lock")
	assert.NilError(t, lock.Unlock())
}

func TestExclusiveLockDirectoryStopsWaitingWhenContextIsDone(t *testing.T) {
	dir := t.TempDir()
	held, err := exclusiveLockDirectory(t.Context(), dir)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = held.Unlock() })

	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer cancel()

	_, err = exclusiveLockDirectory(ctx, dir)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
