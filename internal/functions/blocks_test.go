package functions

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseBlocks(t *testing.T) {
	blocks, err := parseBlocks("testdata/blocks-test.txt")
	assert.NilError(t, err, "expected parseBlocks() not to fail")
	assert.Equal(t, blocks["helloWorld"], "Hello, world!", "expected parseBlocks() to parse basic block")
}
