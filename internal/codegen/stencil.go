// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the stencil function passed to templates
package codegen

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/go-git/go-billy/v5/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// NewStencil creates a new, fully initialized Stencil renderer function
func NewStencil(m *configuration.ServiceManifest, mods []*modules.Module) *Stencil {
	return &Stencil{m, extensions.NewHost(), mods, true, make(map[string]interface{})}
}

// Stencil provides the basic functions for
// stencil templates
type Stencil struct {
	m *configuration.ServiceManifest

	ext *extensions.Host

	// modules is a list of modules used in this stencil render
	modules []*modules.Module

	// isFirstPass denotes if the renderer is currently in first
	// pass mode
	isFirstPass bool

	// sharedData stores data that is injected by templates from modules
	sharedData map[string]interface{}
}

// RegisterExtensions registers all extensions on the currently loaded
// modules.
func (s *Stencil) RegisterExtensions(ctx context.Context) error {
	for _, m := range s.modules {
		if err := m.RegisterExtensions(ctx, s.ext); err != nil {
			return errors.Wrapf(err, "failed to load extensions from module %q", m.Name)
		}
	}

	return nil
}

// GenerateLockfile generates a stencil.Lockfile based
// on a list of templates.
func (s *Stencil) GenerateLockfile(tpls []*Template) *stencil.Lockfile {
	l := &stencil.Lockfile{
		Version:   app.Info().Version,
		Generated: time.Now().UTC(),
	}

	for _, tpl := range tpls {
		for _, f := range tpl.Files {
			l.Files = append(l.Files, &stencil.LockfileFileEntry{
				Name:     f.Name(),
				Template: tpl.Path,
				Module:   tpl.Module.Name,
			})
		}
	}

	for _, m := range s.modules {
		l.Modules = append(l.Modules, &stencil.LockfileModuleEntry{
			Name:    m.Name,
			URL:     m.URI,
			Version: m.Version,
		})
	}

	return l
}

// Render renders all templates using the ServiceManifest that was
// provided to stencil at creation time, returned is the templates
// that were produced and their associated files.
func (s *Stencil) Render(ctx context.Context, log logrus.FieldLogger) ([]*Template, error) {
	tplfiles, err := s.getTemplates(ctx, log)
	if err != nil {
		return nil, err
	}

	// IDEA(jaredallard): Consider sharing this somehow, it's going
	// to be pretty memory inefficient
	//
	// copy the templates into a separate slice for the first pass
	// to prevent extra mutations
	firstPass := make([]*Template, len(tplfiles))
	for i, t := range tplfiles {
		nt := *t
		firstPass[i] = &nt
	}

	vals := NewValues(ctx, s.m)
	tpls := make([]*Template, 0)

	// Add the templates to their modules template to allow them to be able to access
	// functions declared in the same module
	for _, t := range tplfiles {
		log.Debugf("Parsing template %s", t.ImportPath())
		if err := t.Parse(s); err != nil {
			return nil, errors.Wrapf(err, "failed to parse template %q", t.ImportPath())
		}
	}

	// Render the first pass, this is used to populate shared data
	for _, t := range firstPass {
		log.Debugf("First pass render of template %s", t.ImportPath())
		if err := t.Render(s, vals); err != nil {
			return nil, errors.Wrapf(err, "failed to render template %q", t.ImportPath())
		}
	}
	s.isFirstPass = false

	for _, t := range tplfiles {
		log.Debugf("Second pass render of template %s", t.ImportPath())
		if err := t.Render(s, vals); err != nil {
			return nil, errors.Wrapf(err, "failed to render template %q", t.ImportPath())
		}

		// append the rendered template to our list of templates processed
		tpls = append(tpls, t)
	}

	return tpls, nil
}

// PostRun runs all post run commands specified in the modules that
// this service depends on
func (s *Stencil) PostRun(ctx context.Context, log logrus.FieldLogger) error {
	log.Info("Running post-run command(s)")
	for _, m := range s.modules {
		mf, err := m.Manifest(ctx)
		if err != nil {
			return err
		}

		for _, cmdStr := range mf.PostRunCommand {
			log.Infof(" - %s", cmdStr.Name)
			//nolint:gosec // Why: This is by design
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

// getTemplates takes all modules attached to this stencil
// struct and returns all templates exposed by it.
func (s *Stencil) getTemplates(ctx context.Context, log logrus.FieldLogger) ([]*Template, error) {
	tpls := make([]*Template, 0)
	for _, m := range s.modules {
		log.Debugf("Fetching module %q", m.Name)
		fs, err := m.GetFS(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read module filesystem %q", m.Name)
		}

		log.Debugf("Discovering templates from module %q", m.Name)
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

			log.Debugf("Discovered template %q", path)
			tpl, err := NewTemplate(m, path, inf.Mode(), inf.ModTime(), tplContents)
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
