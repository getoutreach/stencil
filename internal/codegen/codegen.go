// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description

// Package codegen has code generators for Go projects
//
// This is intended for use with stencil but can also be used
// outside of it.
//
// Using configuration.ServiceManifest, a list of template repositories
// is created and cloned into a layered filesystem with the sub-dependencies
// of the root dependency (the module) being used first, and so on. This layered
// fs is then walked to find all files with a `.tpl` extension. These are rendred
// and turned into functions.RenderedTemplate objects, and then written to disk
// based on the template's function calls.
//
// This is the core of stencil
package codegen

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/stencil/internal/functions"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/go-git/go-billy/v5/util"
)

// Builder is the heart of stencil, running it is akin to running
// stencil. Builder handles fetching stencil dependencies and running
// the actual templating engine and writing the results to disk. Also
// handled is the extension framework
type Builder struct {
	// dir is the path to write templates to
	dir string

	// manifest is the service manifest that is being used
	// for this template render
	manifest *configuration.ServiceManifest

	// extensions is the extensions host that handles all extensions
	// exposed to templates for this builder
	extensions *extensions.Host

	// log is the logger used for logging output
	log logrus.FieldLogger

	// modules is a list of all modules that this builder is utilizing
	modules []*modules.Module
}

// NewBuilder returns a new builder
func NewBuilder(dir string, log logrus.FieldLogger, s *configuration.ServiceManifest) *Builder {
	_, err := stencil.LoadLockfile("")
	if !errors.Is(err, os.ErrNotExist) {
		log.WithError(err).Warn("failed to load lockfile")
	}

	return &Builder{
		dir:        dir,
		manifest:   s,
		extensions: extensions.NewHost(),
		log:        log,
	}
}

// Run fetches dependencies of the root modules and builds the layered filesystem,
// after that GenerateFiles is called to actually walk the filesystem and render
// the templates. This step also does minimal post-processing of the dependencies
// manifests
func (b *Builder) Run(ctx context.Context) ([]string, error) {
	if err := b.processManifest(); err != nil {
		return nil, errors.Wrap(err, "failed to process service manifest")
	}

	var err error
	b.modules, err = modules.GetModulesForService(ctx, b.manifest)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process modules list")
	}

	warnings, err := b.GenerateFiles(ctx)
	if err != nil {
		return nil, err
	}

	return warnings, b.runPostRunCommands(ctx)
}

// runPostRunCommands runs the postRunCommands set by
// dependencies
func (b *Builder) runPostRunCommands(ctx context.Context) error {
	b.log.Info("Running post run commands")
	for _, m := range b.modules {
		mf, err := m.Manifest(ctx)
		if err != nil {
			return err
		}

		for _, cmdStr := range mf.PostRunCommand {
			b.log.Infof("- %s", cmdStr.Name)

			//nolint:gosec // Why: That's the literal design.
			cmd := exec.CommandContext(ctx, "/usr/bin/env", "bash", "-c", cmdStr.Command)
			cmd.Stdin = os.Stdin
			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout
			if err := cmd.Run(); err != nil {
				return errors.Wrapf(err, "failed to run post run command for module %q", m.Name)
			}
		}
	}

	return nil
}

// setDefaultArguments translates a few manifest values
// into arguments that can be accessed via stencil.Arg
func (b *Builder) setDefaultArguments() error {
	if b.manifest.Arguments == nil {
		b.manifest.Arguments = make(map[string]interface{})
	}
	b.manifest.Arguments["name"] = b.manifest.Name
	return nil
}

// processManifest handles processing any fields in the manifest, i.e validation
func (b *Builder) processManifest() error {
	if err := b.setDefaultArguments(); err != nil {
		return err
	}

	return nil
}

// getTemplates returns a map of templates
func (b *Builder) getTemplates(ctx context.Context) ([]*functions.Template, error) {
	tpls := make([]*functions.Template, 0)

	for _, m := range b.modules {
		fs, err := m.GetFS(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read module filesystem %q", m.Name)
		}

		err = util.Walk(fs, "", func(path string, inf os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip files without a .tpl extension
			if filepath.Ext(path) != ".tpl" {
				return nil
			}

			f, err := fs.Open(path)
			if err != nil {
				return errors.Wrapf(err, "failed to open template %q from module %q", path, m.Name)
			}
			defer f.Close()

			tplContents, err := io.ReadAll(f)
			if err != nil {
				return errors.Wrapf(err, "failed to read template %q from module %q", path, m.Name)
			}

			tpl, err := functions.NewTemplate(m, path, inf.Mode(), inf.ModTime(), tplContents)
			if err != nil {
				return errors.Wrapf(err, "failed to create template %q from module %q", path, m.Name)
			}
			tpls = append(tpls, tpl)

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return tpls, nil
}

// GenerateFiles walks the vfs generated by Run() and renders the templates
func (b *Builder) GenerateFiles(ctx context.Context) ([]string, error) {
	data, err := b.makeTemplateParameters(ctx)
	if err != nil {
		return nil, err
	}

	st := functions.NewStencil(b.manifest, b.modules)

	b.log.Info("Rendering templates")
	tpls, err := b.getTemplates(ctx)
	if err != nil {
		return nil, err
	}

	return nil, b.writeFiles(st)
}

// writeFiles writes the files to disk
func (b *Builder) writeFiles(st *functions.Stencil) error {
	b.log.Infof("Writing template(s) to disk")
	for _, tpl := range st.Templates {
		b.log.Debugf(" -> %s (%s)", tpl.Module.Name, tpl.Path)
		for _, f := range tpl.Files {
			action := "Created"
			if f.Deleted {
				action = "Deleted"
			} else if f.Skipped {
				action = "Skipped"
			} else if _, err := os.Stat(f.Name()); err == nil {
				action = "Updated"
			}

			b.log.Infof("  -> %s %s", action, f.Name())
		}
	}

	l := st.GenerateLockfile()
	yaml.NewEncoder(os.Stdout).Encode(l)

	return nil
}

// makeTemplateParameters creates the map to be provided to the templates.
func (b *Builder) makeTemplateParameters(_ context.Context) (map[string]interface{}, error) { //nolint:funlen
	// TODO: head branch
	boxConf, err := box.LoadBox()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load box config")
	}

	return map[string]interface{}{
		"Metadata": map[string]string{
			"Generator": app.Info().Name,
			"Version":   app.Info().Version,
		},

		"Repository": map[string]string{
			"HeadBranch": "main",
		},

		"Box": boxConf,

		"Manifest":  b.manifest,
		"Arguments": b.manifest.Arguments,
	}, nil
}
