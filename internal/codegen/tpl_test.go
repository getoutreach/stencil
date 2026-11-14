// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the template function map.

package codegen

import (
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestNewFuncMapDoesNotModifyDefault(t *testing.T) {
	// Concurrent calls write to the same map if NewFuncMap modifies Default,
	// which the race detector reports.
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			funcs := NewFuncMap(nil, nil, logrus.New())
			assert.Check(t, funcs["stencil"] != nil)
		})
	}
	wg.Wait()

	for _, name := range []string{"stencil", "file", "extensions"} {
		_, ok := Default[name]
		assert.Check(t, !ok, "expected Default to have no %q function", name)
	}
}
