package vfs_test

import (
	"io"
	"os"
	"testing"

	"github.com/getoutreach/stencil/internal/vfs"
	"github.com/go-git/go-billy/v5/memfs"
	"gotest.tools/v3/assert"
)

func TestVFSBaseFunctionality(t *testing.T) {
	fs1 := memfs.New()
	fs2 := memfs.New()

	f1magicStr := "こんにちは"

	f1, err := fs1.Create("file1.txt") //nolint:errcheck,ineffassign,staticcheck
	f1.Write([]byte(f1magicStr))
	f1.Close()
	fs2.Create("file2.txt")

	fs := vfs.NewLayeredFS(fs1, fs2)
	f, err := fs.Open("file2.txt")
	assert.NilError(t, err, "expected err == nil when accessing second fs")
	f.Close()

	f, err = fs.Open("file3.txt")
	if f != nil {
		f.Close()
	}
	assert.Error(t, err, (os.ErrNotExist).Error(), "expected to error on non-existent file")

	// check that overlapping files are always loaded in the same order
	// write to a file at the same path as fs1 on fs2 and see if we get fs1
	duplicate, err := fs2.Create("file1.txt") //nolint:errcheck,ineffassign,staticcheck
	duplicate.Write([]byte("さよなら"))
	duplicate.Close()

	// open the files to compare
	f, err = fs.Open("file1.txt")
	assert.NilError(t, err, "couldn't read file in ordering test")
	b, err := io.ReadAll(f)
	assert.NilError(t, err, "failed to read contents of file")

	assert.Equal(t, string(b), f1magicStr, "ordering test failed, got %v instead of %s", string(b), f1magicStr)
}
