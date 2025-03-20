package codegen

import (
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"gotest.tools/v3/assert"
)

// createFakeModuleFSWithManifest creates an in-memory filesystem with a single
// file named "manifest.yaml" containing the provided manifest contents. This
// is useful for testing purposes where a mock filesystem is needed.
//
// Parameters:
//   - t: The testing object used for assertions.
//   - manifestContents: A string representing the contents to be written to the manifest file.
//
// Returns:
//   - A billy.Filesystem representing the in-memory filesystem with the manifest file.
func createFakeModuleFSWithManifest(t *testing.T, manifestContents string) billy.Filesystem {
	t.Helper()
	fs := memfs.New()
	mf, err := fs.Create("manifest.yaml")
	assert.NilError(t, err)
	_, err = mf.Write([]byte(manifestContents))
	assert.NilError(t, err)
	assert.NilError(t, mf.Close())

	return fs
}
