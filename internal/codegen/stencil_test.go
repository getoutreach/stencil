package codegen

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/internal/modules/modulestest"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/sirupsen/logrus"
	"gotest.tools/v3/assert"
)

func TestBasicE2ERender(t *testing.T) {
	ctx := context.Background()

	// create stub manifest
	fs := createFakeModuleFSWithManifest(t, "name: testing")

	// create a stub template
	f, err := fs.Create("test-template.tpl")
	assert.NilError(t, err, "failed to create stub template")
	f.Write([]byte("{{ .Config.Name }}"))
	f.Close()

	st := NewStencil(&configuration.ServiceManifest{
		Name:      "test",
		Arguments: map[string]interface{}{},
	}, []*modules.Module{
		modules.NewWithFS(ctx, "testing", fs),
	}, logrus.New())

	tpls, err := st.Render(ctx, logrus.New())
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpls), 1, "expected Render() to return a single template")
	assert.Equal(t, len(tpls[0].Files), 1, "expected Render() template to return a single file")
	assert.Equal(t, tpls[0].Files[0].String(), "test", "expected Render() to return correct output")

	lock := st.GenerateLockfile(tpls)
	assert.DeepEqual(t, lock, &stencil.Lockfile{
		Version: app.Info().Version,
		Modules: []*stencil.LockfileModuleEntry{
			{
				Name:    "testing",
				URL:     "vfs://testing",
				Version: "vfs",
			},
		},
		Files: []*stencil.LockfileFileEntry{
			{
				Name:     "test-template",
				Template: "test-template.tpl",
				Module:   "testing",
			},
		},
	})
}

func TestModuleHookRender(t *testing.T) {
	ctx := context.Background()

	// create modules
	m1man := &configuration.TemplateRepositoryManifest{
		Name: "testing1",
	}
	m1, err := modulestest.NewModuleFromTemplates(m1man, "testdata/module-hook/m1.tpl")
	if err != nil {
		t.Errorf("failed to create module 1: %v", err)
	}
	m2man := &configuration.TemplateRepositoryManifest{
		Name: "testing2",
	}
	m2, err := modulestest.NewModuleFromTemplates(m2man, "testdata/module-hook/m2.tpl")
	if err != nil {
		t.Errorf("failed to create module 2: %v", err)
	}

	st := NewStencil(&configuration.ServiceManifest{
		Name:      "test",
		Arguments: map[string]interface{}{},
	}, []*modules.Module{m1, m2}, logrus.New())

	tpls, err := st.Render(ctx, logrus.New())
	assert.NilError(t, err, "expected Render() to not fail")
	assert.Equal(t, len(tpls), 2, "expected Render() to return a single template")
	assert.Equal(t, len(tpls[1].Files), 1, "expected Render() template to return a single file")
	assert.Equal(t, tpls[1].Files[0].String(), "a", "expected Render() to return correct output")
}

func ExampleStencil_PostRun() {
	fs := memfs.New()
	ctx := context.Background()

	// create a stub manifest
	f, _ := fs.Create("manifest.yaml")
	f.Write([]byte("name: testing\npostRunCommand:\n- command: echo \"hello\""))
	f.Close()

	nullLog := logrus.New()
	nullLog.SetOutput(io.Discard)

	st := NewStencil(&configuration.ServiceManifest{
		Name:      "test",
		Arguments: map[string]interface{}{},
	}, []*modules.Module{
		modules.NewWithFS(ctx, "testing", fs),
	}, logrus.New())
	err := st.PostRun(ctx, nullLog)
	if err != nil {
		fmt.Println(err)
	}

	// Output:
	// hello
}

func TestStencilPostRunError(t *testing.T) {
	const (
		name    = "TestStencilPostRunError"
		command = "invalidPostRunCommand"
	)

	var (
		ctx              = context.Background()
		logger           = logrus.New()
		manifestContents = fmt.Sprintf("name: %s\npostRunCommand:\n- command: %s\n", name, command)
		errMsg           = fmt.Sprintf("failed to run post run command for module %s and command %s", name, command)
	)

	fs := createFakeModuleFSWithManifest(t, manifestContents)
	st := NewStencil(
		&configuration.ServiceManifest{Name: name},
		[]*modules.Module{modules.NewWithFS(ctx, name, fs)},
		logger,
	)

	assert.ErrorContains(t, st.PostRun(ctx, logger), errMsg)
}
