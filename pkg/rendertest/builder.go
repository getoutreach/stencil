// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the TemplateBuilder, which configures and
// runs the render of one template.

package rendertest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/inprocext"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/sirupsen/logrus"
)

// errBlocksAndExistingFile is returned when WithBlocks and WithExistingFile
// set the same file.
var errBlocksAndExistingFile = errors.New("WithBlocks and WithExistingFile both set the file")

// defaultServiceName is the name in the default service manifest.
const defaultServiceName = "testing"

// TemplateBuilder configures the render of one template. Its methods only
// store configuration and return the builder, so calls can be chained. Run
// renders the template.
type TemplateBuilder struct {
	// suite holds the module under test and its dependencies
	suite *Suite

	// path is the path to the template under test
	path string

	// manifest is the service manifest set with WithManifest
	manifest *configuration.ServiceManifest

	// name is the service name set with WithName
	name string

	// args are the template arguments
	args map[string]any

	// blocks are the blocks of the existing output file
	blocks map[string]string

	// hooks are the module hook values of the module under test
	hooks map[string][]any

	// existingFiles are the files that exist before the render
	existingFiles map[string][]byte

	// boxConfig is the box config that templates see
	boxConfig *box.Config

	// extensions are the in-process extensions, by name
	extensions map[string]apiv1.Implementation

	// templates are the helper templates
	templates []string
}

// NewTemplate returns a TemplateBuilder for the template at path. The path is
// relative to the working directory of the test, which for a module's
// template tests is its templates directory.
func (s *Suite) NewTemplate(path string) *TemplateBuilder {
	return &TemplateBuilder{
		suite:         s,
		path:          path,
		args:          map[string]any{},
		blocks:        map[string]string{},
		hooks:         map[string][]any{},
		existingFiles: map[string][]byte{},
		extensions:    map[string]apiv1.Implementation{},
	}
}

// WithManifest sets the service manifest. The default has the name
// "testing" and lists the module under test. Arguments set with WithArgs or
// WithRawArgs are added to the arguments of m, which is not modified.
func (b *TemplateBuilder) WithManifest(m *configuration.ServiceManifest) *TemplateBuilder {
	b.manifest = m
	return b
}

// WithName sets the name in the service manifest.
func (b *TemplateBuilder) WithName(name string) *TemplateBuilder {
	b.name = name
	return b
}

// WithArgs sets template arguments with argument fixtures.
func (b *TemplateBuilder) WithArgs(opts ...ArgOption) *TemplateBuilder {
	Args(opts...)(b.args)
	return b
}

// WithRawArgs sets the top-level template arguments in args.
func (b *TemplateBuilder) WithRawArgs(args map[string]any) *TemplateBuilder {
	maps.Copy(b.args, args)
	return b
}

// WithBlocks sets the blocks of the existing file at the default output path
// of the template, which is its path without ".tpl". The file exists, so
// file.Static skips it. To set blocks of a file at another path, use
// WithExistingFile with contents that have the blocks.
func (b *TemplateBuilder) WithBlocks(blocks map[string]string) *TemplateBuilder {
	maps.Copy(b.blocks, blocks)
	return b
}

// WithHooks adds values to the module hooks of the module under test, by hook
// name, as if templates of other modules had added them.
func (b *TemplateBuilder) WithHooks(hooks map[string][]any) *TemplateBuilder {
	for name, values := range hooks {
		b.hooks[name] = append(b.hooks[name], values...)
	}
	return b
}

// WithExistingFile sets a file that exists before the render, at a path
// relative to the root of the generated repository.
func (b *TemplateBuilder) WithExistingFile(path string, contents []byte) *TemplateBuilder {
	b.existingFiles[path] = contents
	return b
}

// WithExistingFiles sets files that exist before the render, by path.
func (b *TemplateBuilder) WithExistingFiles(files map[string][]byte) *TemplateBuilder {
	maps.Copy(b.existingFiles, files)
	return b
}

// WithBoxConfig sets the box config that templates see as .Runtime.Box. The
// default is an empty config.
func (b *TemplateBuilder) WithBoxConfig(c *box.Config) *TemplateBuilder {
	b.boxConfig = c
	return b
}

// WithExtension registers an in-process extension. As with a real extension,
// every response is encoded to JSON and decoded again, so templates see the
// same types.
func (b *TemplateBuilder) WithExtension(name string, ext apiv1.Implementation) *TemplateBuilder {
	b.extensions[name] = inprocext.Wrap(ext)
	return b
}

// WithTemplates adds helper templates of the module under test, at paths
// relative to the working directory of the test. They are rendered with the
// template under test, but their files are not in the Result.
func (b *TemplateBuilder) WithTemplates(paths ...string) *TemplateBuilder {
	b.templates = append(b.templates, paths...)
	return b
}

// Run renders the template and returns the Result. A render error is in the
// Result, while a failure to set up the render fails the test.
func (b *TemplateBuilder) Run(tb testing.TB) *Result {
	tb.Helper()

	s := b.suite
	if !s.loaded {
		tb.Fatal("rendertest: the suite is not loaded, call Suite.Run in TestMain")
	}

	ctx := tb.Context()
	m, err := newModule(ctx, s.name, "", s.manifestYAML, append([]string{b.path}, b.templates...)...)
	if err != nil {
		tb.Fatalf("rendertest: failed to create the module under test: %v", err)
	}

	fs, err := b.newFS()
	if err != nil {
		tb.Fatalf("rendertest: failed to create the existing files: %v", err)
	}

	log := logrus.New()
	log.SetOutput(testWriter{tb})
	log.SetLevel(logrus.WarnLevel)

	boxConfig := b.boxConfig
	if boxConfig == nil {
		boxConfig = &box.Config{}
	}
	runner := &commandRecorder{}
	opts := make([]codegen.StencilOption, 0, 4+len(b.hooks))
	opts = append(opts,
		codegen.WithFS(fs),
		// An empty ref and commit, the default branch main and a clean tree.
		codegen.WithGitInfo("", "", "main", false),
		codegen.WithBoxConfig(boxConfig),
		codegen.WithCommandRunner(runner),
	)
	for name, values := range b.hooks {
		opts = append(opts, codegen.WithModuleHook(m.Name, name, values))
	}

	st := codegen.NewStencil(b.serviceManifest(m.Name), append([]*modules.Module{m}, s.deps...), log, opts...)
	defer st.Close()
	for name, ext := range b.extensions {
		st.RegisterInprocExtensions(name, ext)
	}

	return b.render(ctx, st, log, runner)
}

// render renders with st and returns the Result.
func (b *TemplateBuilder) render(ctx context.Context, st *codegen.Stencil, log logrus.FieldLogger,
	runner *commandRecorder) *Result {
	tpls, err := st.Render(ctx, log)
	if err != nil {
		return &Result{err: err}
	}
	if err := st.PostRun(ctx, log); err != nil {
		return &Result{err: err}
	}

	r := &Result{postRunCommands: runner.commands, hookValues: st.ModuleHooks()}
	for _, tpl := range tpls {
		if tpl.Path != filepath.ToSlash(b.path) {
			continue
		}
		for _, f := range tpl.Files {
			switch {
			case f.Deleted:
				r.deletedPaths = append(r.deletedPaths, f.Name())
			case f.Skipped:
				r.skippedPaths = append(r.skippedPaths, f.Name())
			default:
				r.files = append(r.files, newFileResult(f))
			}
		}
	}
	return r
}

// newFS returns an in-memory filesystem with the existing files and the
// file with the blocks.
func (b *TemplateBuilder) newFS() (billy.Filesystem, error) {
	fs := memfs.New()
	for p, contents := range b.existingFiles {
		if err := util.WriteFile(fs, p, contents, 0o644); err != nil {
			return nil, err
		}
	}

	if len(b.blocks) == 0 {
		return fs, nil
	}

	p := strings.TrimSuffix(filepath.ToSlash(b.path), ".tpl")
	if _, ok := b.existingFiles[p]; ok {
		return nil, fmt.Errorf("%w: %q", errBlocksAndExistingFile, p)
	}
	return fs, util.WriteFile(fs, p, blocksFile(b.blocks), 0o644)
}

// blocksFile returns the contents of a file with the given blocks.
func blocksFile(blocks map[string]string) []byte {
	var buf bytes.Buffer
	for _, name := range slices.Sorted(maps.Keys(blocks)) {
		fmt.Fprintf(&buf, "## <<Stencil::Block(%s)>>\n%s\n## <</Stencil::Block>>\n", name, blocks[name])
	}
	return buf.Bytes()
}

// serviceManifest returns the service manifest for the render of module.
func (b *TemplateBuilder) serviceManifest(module string) *configuration.ServiceManifest {
	sm := configuration.ServiceManifest{
		Name:    defaultServiceName,
		Modules: []*configuration.TemplateRepository{{Name: module}},
	}
	if b.manifest != nil {
		sm = *b.manifest
	}
	if b.name != "" {
		sm.Name = b.name
	}

	args := map[string]any{}
	maps.Copy(args, sm.Arguments)
	maps.Copy(args, b.args)
	sm.Arguments = args

	return &sm
}

// commandRecorder records post-run commands instead of running them.
type commandRecorder struct {
	// commands are the recorded commands, in order
	commands []string
}

// Run implements codegen.CommandRunner.
func (r *commandRecorder) Run(_ context.Context, _, command string) error {
	r.commands = append(r.commands, command)
	return nil
}

// testWriter writes log output to a test log.
type testWriter struct {
	t testing.TB
}

// Write implements io.Writer.
func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(strings.TrimSuffix(string(p), "\n"))
	return len(p), nil
}
