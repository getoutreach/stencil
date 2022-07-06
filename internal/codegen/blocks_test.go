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

func TestDanglingBlock(t *testing.T) {
	_, err := parseBlocks("testdata/danglingblock-test.txt")
	assert.Error(t, err, "found dangling Block (dangles) in testdata/danglingblock-test.txt", "expected parseBlocks() to fail")
}

func TestDanglingEndBlock(t *testing.T) {
	_, err := parseBlocks("testdata/danglingendblock-test.txt")
	assert.Error(t, err,
		"invalid EndBlock, found EndBlock with name \"dangles\" while inside of block with name \"\", at testdata/danglingendblock-test.txt:8",
		"expected parseBlocks() to fail")
}
