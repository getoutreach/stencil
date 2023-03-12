package jet_test

import (
	"bytes"
	"strings"
	"testing"

	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/getoutreach/stencil/internal/engine/jet"
	"gotest.tools/v3/assert"
)

//go:embed testdata/shouldrender.jet
var shouldRenderData string

func TestShouldRender(t *testing.T) {
	i, err := jet.NewInstance("test")
	assert.NilError(t, err, "failed to create engine instance")

	err = i.Parse("test.jet", strings.NewReader(shouldRenderData), sprig.TxtFuncMap())
	assert.NilError(t, err, "failed to parse template")

	buf := new(bytes.Buffer)
	err = i.Render("test.jet", buf, sprig.TxtFuncMap(), map[string]interface{}{
		"Config": map[string]interface{}{
			"Name": "test",
		},
	})
	assert.NilError(t, err, "failed to render template")

	assert.Equal(t, buf.String(), "test", "expected template to render correctly")
}
