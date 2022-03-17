// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Contains tests for the file file

package codegen

import (
	"io"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"gotest.tools/v3/assert"
)

func TestFile_Size(t *testing.T) {
	fs := memfs.New()
	f, err := fs.Create("hello.txt")
	assert.NilError(t, err, "failed to create test file in memory")

	_, err = f.Write([]byte("hello world"))
	assert.NilError(t, err, "failed to write test data to test file in memory")
	assert.NilError(t, f.Close(), "failed to close file")

	f, err = fs.Open("hello.txt")
	assert.NilError(t, err, "failed to open test file in memory")

	data, err := io.ReadAll(f)
	assert.NilError(t, err, "failed to read data from test file")
	assert.NilError(t, f.Close(), "failed to close file")

	inf, err := fs.Stat("hello.txt")
	assert.NilError(t, err, "failed to state test file in memory")

	mockF := &File{contents: data}
	assert.Equal(t, mockF.Size(), inf.Size(), "(File).Size() was not equal to memory fs (os.FileInfo).Size()")
}

func TestFileBasic(t *testing.T) {
	cnts := "hello, world"
	f := &File{}
	f.SetContents(cnts)
	assert.Equal(t, cnts, string(f.contents), "expected SetContents() to set contents")
	assert.Equal(t, cnts, f.String(), "expected String() to return proper contents")
}
