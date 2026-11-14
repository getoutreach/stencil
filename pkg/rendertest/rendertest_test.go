// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for rendering templates with a
// Suite.

package rendertest_test

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/getoutreach/stencil/pkg/rendertest"
	"gotest.tools/v3/assert"
)

var suite *rendertest.Suite

func TestMain(m *testing.M) {
	// Template tests run in the templates directory of a module.
	if err := os.Chdir("testdata/templates"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	suite = rendertest.NewSuite(m,
		rendertest.WithManifestPath("../module/manifest.yaml"),
		rendertest.WithModuleReplacements(map[string]string{"github.com/example/dep": "../dep"}),
	)
	os.Exit(suite.Run())
}

// renderOne renders the template under test, checks that it rendered
// without an error to one file, and returns that file.
func renderOne(t *testing.T, b *rendertest.TemplateBuilder) *rendertest.FileResult {
	t.Helper()

	result := b.Run(t)
	assert.NilError(t, result.Error())
	assert.Equal(t, len(result.Files()), 1)
	return result.Files()[0]
}

func TestArgs(t *testing.T) {
	t.Parallel()

	f := renderOne(t, suite.NewTemplate("args.tpl").WithArgs(rendertest.Arg("name", "my-service")))
	assert.Equal(t, f.Path(), "args")
	assert.Equal(t, f.String(), "name=my-service\n")
	assert.DeepEqual(t, f.Contents(), []byte("name=my-service\n"))
}

func TestNestedArgsAndArgumentFromADependency(t *testing.T) {
	t.Parallel()

	f := renderOne(t, suite.NewTemplate("nested.tpl").
		WithArgs(rendertest.Args(
			rendertest.Arg("name", "my-service"),
			rendertest.NestedArg("kubernetes.replicas", 1),
		), rendertest.NestedArg("kubernetes.replicas", 3)).
		WithRawArgs(map[string]any{"shared": "from-dep"}))
	assert.Equal(t, f.String(), "replicas=3 shared=from-dep\n")
}

func TestNestedArgReplacesValuesThatAreNotMaps(t *testing.T) {
	t.Parallel()

	args := map[string]any{"a": "not a map"}
	rendertest.NestedArg("a.b.c", 1)(args)
	assert.DeepEqual(t, args, map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}})
}

func TestRenderError(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("args.tpl").Run(t)
	assert.ErrorContains(t, result.Error(), "name")
	assert.Equal(t, len(result.Files()), 0)
}

func TestFileStates(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("files.tpl").Run(t)
	assert.NilError(t, result.Error())

	kept := result.File("kept.txt")
	assert.Assert(t, kept != nil)
	assert.Equal(t, kept.String(), "kept")
	assert.Equal(t, kept.Mode(), os.FileMode(0o600))
	assert.Assert(t, result.File("deleted.txt") == nil)
	assert.DeepEqual(t, result.DeletedPaths(), []string{"deleted.txt"})
	assert.DeepEqual(t, result.SkippedPaths(), []string{"skipped.txt"})
}

func TestPostRunCommandsAreRecorded(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("args.tpl").WithArgs(rendertest.Arg("name", "x")).Run(t)
	assert.NilError(t, result.Error())

	commands := result.PostRunCommands()
	slices.Sort(commands)
	assert.DeepEqual(t, commands, []string{"echo dep", "echo module"})
}

func TestHooks(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("hooks.tpl").
		WithHooks(map[string][]any{"consumed": {"a", "b"}}).
		Run(t)
	assert.NilError(t, result.Error())

	f := result.File("hooks")
	assert.Assert(t, f != nil)
	// The engine sorts hook values by their hash.
	assert.Equal(t, f.String(), "consumed=b\nconsumed=a\n")
	assert.DeepEqual(t, result.HookValues(), map[string][]any{
		"github.com/example/module/consumed": {"b", "a"},
		"github.com/example/dep/produced":    {"from-module"},
	})
}

func TestBlocks(t *testing.T) {
	t.Parallel()

	f := renderOne(t, suite.NewTemplate("blocks.tpl").WithBlocks(map[string]string{"custom": "line 1\nline 2"}))
	assert.Equal(t, f.BlockValue("custom"), "line 1\nline 2")
	assert.Equal(t, f.BlockValue("missing"), "")
}

func TestBlocksAndExistingFileAtTheSamePath(t *testing.T) {
	t.Parallel()

	ft := &fatalRecorder{T: t}
	func() {
		defer func() { _ = recover() }()
		suite.NewTemplate("blocks.tpl").
			WithBlocks(map[string]string{"custom": "value"}).
			WithExistingFile("blocks", []byte("existing")).
			Run(ft)
	}()
	assert.Assert(t, ft.fatal, "expected Run to fail the test")
}

func TestExistingFiles(t *testing.T) {
	t.Parallel()

	f := renderOne(t, suite.NewTemplate("existing.tpl").
		WithExistingFiles(map[string][]byte{"input.txt": []byte("from memory")}))
	assert.Equal(t, f.String(), "from memory\n")

	result := suite.NewTemplate("existing.tpl").
		WithExistingFile("input.txt", []byte("from memory")).
		WithExistingFile("existing", []byte("static")).
		Run(t)
	assert.NilError(t, result.Error())
	assert.DeepEqual(t, result.SkippedPaths(), []string{"existing"})
}

func TestHelperTemplates(t *testing.T) {
	t.Parallel()

	result := suite.NewTemplate("helpers.tpl").
		WithTemplates("_helpers.tpl").
		WithArgs(rendertest.Arg("name", "my-service")).
		Run(t)
	assert.NilError(t, result.Error())
	assert.Equal(t, len(result.Files()), 1, "expected no files of helper templates")
	assert.Equal(t, result.File("helpers").String(), "hello, my-service global value\n")
	assert.Equal(t, len(result.SkippedPaths()), 0, "expected no skipped files of helper templates")
}

// greeter is an extension with one function, greet.
type greeter struct{}

// greeting is the response of greet. Templates see it as a map.
type greeting struct {
	Text string `json:"text"`
}

func (greeter) GetConfig() (*apiv1.Config, error) {
	return &apiv1.Config{}, nil
}

func (greeter) GetTemplateFunctions() ([]*apiv1.TemplateFunction, error) {
	return []*apiv1.TemplateFunction{{Name: "greet", NumberOfArguments: 1}}, nil
}

func (greeter) ExecuteTemplateFunction(t *apiv1.TemplateFunctionExec) (any, error) {
	return greeting{Text: fmt.Sprintf("hello, %v", t.Arguments[0])}, nil
}

func TestExtension(t *testing.T) {
	t.Parallel()

	f := renderOne(t, suite.NewTemplate("extension.tpl").WithExtension("testext", greeter{}))
	assert.Equal(t, f.String(), "map[text:hello, world]\n")
}

func TestValues(t *testing.T) {
	t.Parallel()

	f := renderOne(t, suite.NewTemplate("values.tpl"))
	assert.Equal(t, f.String(), "testing main  \n")

	f = renderOne(t, suite.NewTemplate("values.tpl").
		WithManifest(&configuration.ServiceManifest{Name: "from-manifest"}).
		WithBoxConfig(&box.Config{Org: "example"}))
	assert.Equal(t, f.String(), "from-manifest main  example\n")

	f = renderOne(t, suite.NewTemplate("values.tpl").
		WithManifest(&configuration.ServiceManifest{Name: "from-manifest"}).
		WithName("from-name"))
	assert.Equal(t, f.String(), "from-name main  \n")
}

func TestWithManifestKeepsItsArguments(t *testing.T) {
	t.Parallel()

	m := &configuration.ServiceManifest{Name: "svc", Arguments: map[string]any{"name": "from-manifest"}}
	f := renderOne(t, suite.NewTemplate("args.tpl").WithManifest(m))
	assert.Equal(t, f.String(), "name=from-manifest\n")

	f = renderOne(t, suite.NewTemplate("args.tpl").WithManifest(m).WithArgs(rendertest.Arg("name", "from-args")))
	assert.Equal(t, f.String(), "name=from-args\n")
	assert.Equal(t, m.Arguments["name"], "from-manifest", "expected the manifest not to be modified")
}

// fatalRecorder is a testing.TB that records a call to Fatal or Fatalf and
// stops the caller with a panic.
type fatalRecorder struct {
	*testing.T

	fatal bool
}

func (r *fatalRecorder) Fatal(...any) {
	r.fatal = true
	panic("fatal")
}

func (r *fatalRecorder) Fatalf(string, ...any) {
	r.fatal = true
	panic("fatal")
}

func BenchmarkSingleTemplate(b *testing.B) {
	for b.Loop() {
		if err := suite.NewTemplate("args.tpl").WithArgs(rendertest.Arg("name", "x")).Run(b).Error(); err != nil {
			b.Fatal(err)
		}
	}
}
