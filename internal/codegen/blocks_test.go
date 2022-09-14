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
	assert.Equal(t, blocks["e2e"], "content", "expected parseBlocks() to parse e2e block")
}

func TestDanglingBlock(t *testing.T) {
	_, err := parseBlocks("testdata/danglingblock-test.txt")
	assert.Error(t, err, "found dangling Block (dangles) in testdata/danglingblock-test.txt", "expected parseBlocks() to fail")
}

func TestDanglingEndBlock(t *testing.T) {
	_, err := parseBlocks("testdata/danglingendblock-test.txt")
	assert.Error(t, err,
		"invalid EndBlock when not inside of a block, at testdata/danglingendblock-test.txt:8",
		"expected parseBlocks() to fail")
}

func TestBlockInsideBlock(t *testing.T) {
	_, err := parseBlocks("testdata/blockinsideblock-test.txt")
	assert.Error(t, err,
		"invalid Block when already inside of a block, at testdata/blockinsideblock-test.txt:3",
		"expected parseBlocks() to fail")
}

func TestWrongEndBlock(t *testing.T) {
	_, err := parseBlocks("testdata/wrongendblock-test.txt")
	assert.Error(t, err,
		"invalid EndBlock, found EndBlock with name \"wrongend\" while inside of block with name \"helloWorld\", at testdata/wrongendblock-test.txt:3", //nolint:lll
		"expected parseBlocks() to fail")
}

func TestParseV2Blocks(t *testing.T) {
	blocks, err := parseBlocks("testdata/v2blocks-test.txt")
	assert.NilError(t, err, "expected parseBlocks() not to fail")
	assert.Equal(t, blocks["helloWorld"], "Hello, world!", "expected parseBlocks() to parse basic block")
	assert.Equal(t, blocks["e2e"], "content", "expected parseBlocks() to parse e2e block")
}
