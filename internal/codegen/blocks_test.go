// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the blocks file

package codegen

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseBlocks(t *testing.T) {
	blocks, err := parseBlocks("testdata/blocks-test.txt")
	assert.NilError(t, err, "expected parseBlocks() not to fail")
	assert.Equal(t, blocks["helloWorld"], "Hello, world!", "expected parseBlocks() to parse basic block")
}
