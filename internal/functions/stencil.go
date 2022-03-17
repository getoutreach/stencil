// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the stencil function passed to templates
package functions

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/go-git/go-billy/v5/util"
	"github.com/pkg/errors"
)

// NewStencil creates a new, fully initialized Stencil renderer function
func NewStencil(m *configuration.ServiceManifest, mods []*modules.Module) *Stencil {
	return &Stencil{m, mods}
}

// Stencil provides the basic functions for
// stencil templates
type Stencil struct {
	m *configuration.ServiceManifest

	// modules is a list of modules used in this stencil render
	modules []*modules.Module
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
func (s *Stencil) Render(ctx context.Context) ([]*Template, error) {
	args, err := s.makeTemplateParameters()
	if err != nil {
		return nil, err
	}

	tplfiles, err := s.getTemplates(ctx)
	if err != nil {
		return nil, err
	}

	tpls := make([]*Template, 0)

	// Add the templates to their modules template to allow them to be able to access
	// functions declared in the same module
	for _, t := range tplfiles {
		if err := t.Parse(s); err != nil {
			return nil, errors.Wrapf(err, "failed to parse template %q", t.ImportPath())
		}
	}

	// Now we render each file
	for _, t := range tplfiles {
		if err := t.Render(s, args); err != nil {
			return nil, errors.Wrapf(err, "failed to render template %q", t.ImportPath())
		}

		// append the rendered template to our list of templates processed
		tpls = append(tpls, t)
	}

	return tpls, nil
}

// makeTemplateParameters creates the map to be provided to the templates.
func (s *Stencil) makeTemplateParameters() (map[string]interface{}, error) {
	// TODO(jaredallard): head branch
	boxConf, err := box.LoadBox()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load box config")
	}

	return map[string]interface{}{
		"Metadata": map[string]string{
			"Generator": app.Info().Name,
			"Version":   app.Info().Version,
		},
		"App": map[string]string{
			"Name": s.m.Name,
		},
		"Repository": map[string]string{
			"HeadBranch": "main",
		},
		"Box": boxConf,
	}, nil
}

// getTemplates takes all modules attached to this stencil
// struct and returns all templates exposed by it.
func (s *Stencil) getTemplates(ctx context.Context) ([]*Template, error) {
	tpls := make([]*Template, 0)
	for _, m := range s.modules {
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
