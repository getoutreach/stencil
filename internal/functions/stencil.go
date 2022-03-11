// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the stencil function passed to templates
package functions

import (
	"time"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
)

// NewStencil creates a new, fully initialized Stencil renderer function
func NewStencil(m *configuration.ServiceManifest, modules []*modules.Module) *Stencil {
	return &Stencil{m, modules, nil, nil}
}

// Stencil provides the basic functions for
// stencil templates
type Stencil struct {
	m *configuration.ServiceManifest

	// Modules is a list of modules used in this stencil render
	Modules []*modules.Module

	// Templates is a list of all templates that this renderer rendered.
	Templates []*Template

	// Template is the current template that is being rendered by this
	// renderer.
	Template *Template
}

// GenerateLockfile generates a stencil.Lockfile based
// on the current state of the renderer.
func (s *Stencil) GenerateLockfile() *stencil.Lockfile {
	l := &stencil.Lockfile{
		Version:   app.Info().Version,
		Generated: time.Now().UTC(),
	}

	for _, tpl := range s.Templates {
		for _, f := range tpl.Files {
			l.Files = append(l.Files, &stencil.LockfileFileEntry{
				Name:     f.Name(),
				Template: tpl.Path,
				Module:   tpl.Module.Name,
			})
		}
	}

	for _, m := range s.Modules {
		l.Modules = append(l.Modules, &stencil.LockfileModuleEntry{
			Name:    m.Name,
			URL:     m.URI,
			Version: m.Version,
		})
	}

	return l
}
