package functions

import (
	"context"
	"testing"

	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-billy/v5/memfs"
	"gotest.tools/v3/assert"
)

func TestBasicE2ERender(t *testing.T) {
	fs := memfs.New()

	// create a stub template
	f, err := fs.Create("test-template.tpl")
	assert.NilError(t, err, "failed to create stub template")
	f.Write([]byte("{{ .App.Name }}"))
	f.Close()

	st := NewStencil(&configuration.ServiceManifest{
		Name:      "test",
		Arguments: map[string]interface{}{},
	}, []*modules.Module{
		modules.NewWithFS("testing", fs),
	})

	tpls, err := st.Render(context.Background())
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpls), 1, "expected Render() to return a single template")
	assert.Equal(t, len(tpls[0].Files), 1, "expected Render() template to return a single file")
	assert.Equal(t, tpls[0].Files[0].String(), "test", "expected Render() to return correct output")
}
