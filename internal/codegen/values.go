// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains global variables that are
// exposed to templates at the root of the template arguments.
// (e.g. {{ .Repository.Name }})

package codegen

import (
	"context"

	"github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	stencilgit "github.com/getoutreach/stencil/internal/git"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	gogit "github.com/go-git/go-git/v5"
)

// runtime contains information about the current state
// of an application. This includes things like Golang
// version, stencil version, and other tool information.
type runtime struct {
	// Generator is the name of the tool that is generating this file
	// generally this would be "stencil", this value should be changed
	// if using stencil's codegen package outside of the stencil CLI.
	Generator string

	// GeneratorVersion is the current version of the generator being
	// used.
	GeneratorVersion string

	// Box is org wide configuration that is accessible if configured
	Box *box.Config

	// Modules contains a list of all modules that are being rendered
	// in a stencil run
	Modules modulesSlice
}

// git contains information about the current git repository
// that is being ran in
type git struct {
	// Ref is the current ref of the Git repository, this
	// is in the refs/<type>/<name> format
	Ref string

	// Commit is the current commit that this git repository is at
	Commit string

	// Dirty denotes if the current git repository is dirty or not.
	// Dirty is determined by having untracked changes to the current
	// index.
	Dirty bool

	// DefaultBranch is the default branch to use for this repository
	// generally this is equal to "main", but some repositories
	// use other values.
	DefaultBranch string
}

// config contains a small amount of configuration that
// originates from the service manifest and is propagated
// here.
type config struct {
	// Name is the name of this repository
	Name string

	// Deprecated: Versions should be treated as an argument via stencil.Arg instead
	// Versions are versions of applications to be used over baked in defaults
	Versions map[string]string
}

// module contains information about the current module that
// is rendering a template.
type module struct {
	// Name is the name of the current module
	Name string

	// Version is the version of the current module
	Version string
}

// stencilTemplate contains information about the current template
type stencilTemplate struct {
	// Name is the name of the template
	Name string
}

// modulesSlice is a list of modules with helpers on top of it
type modulesSlice []module

// ByName returns a module by name
func (m modulesSlice) ByName(name string) module {
	for _, mod := range m {
		if mod.Name == name {
			return mod
		}
	}

	return module{}
}

// Values is the top level container for variables being
// passed to a stencil template.
type Values struct {
	// Git is information about the current git repository, if there is one
	Git git

	// Runtime is information about the current runtime environment
	Runtime runtime

	// Config is strongly type values from the service manifest
	Config config

	// Module is information about the current module being rendered
	Module module

	// Template is the name of the template being rendered
	Template stencilTemplate
}

// Copy returns a copy of the current values
func (v *Values) Copy() *Values {
	nv := *v
	return &nv
}

// WithModule returns a copy of the current values with the
// provided module information being set.
func (v *Values) WithModule(name, version string) *Values {
	nv := v.Copy()
	nv.Module.Name = name
	nv.Module.Version = version
	return nv
}

// WithTemplate returns a copy of the current values with the
// provided template information being set.
func (v *Values) WithTemplate(name string) *Values {
	nv := v.Copy()
	nv.Template.Name = name
	return nv
}

// NewValues returns a fully initialized Values
// based on the current runtime environment.
func NewValues(ctx context.Context, sm *configuration.ServiceManifest, mods []*modules.Module) *Values {
	vals := &Values{
		Git: git{},
		Runtime: runtime{
			Generator:        app.Info().Name,
			GeneratorVersion: app.Info().Version,
			Modules:          modulesSlice{},
		},
		Config: config{
			Name:     sm.Name,
			Versions: sm.Versions,
		},
		Module:   module{},
		Template: stencilTemplate{},
	}

	for _, m := range mods {
		vals.Runtime.Modules = append(vals.Runtime.Modules, module{
			Name:    m.Name,
			Version: m.Version,
		})
	}

	var err error
	vals.Runtime.Box, err = box.LoadBox()
	if err != nil {
		vals.Runtime.Box = &box.Config{}
	}

	// If we're a repository, add repository information
	if r, err := gogit.PlainOpen(""); err == nil {
		db, err := stencilgit.GetDefaultBranch(ctx, "")
		if err != nil {
			db = "main"
		}
		vals.Git.DefaultBranch = db

		// Add HEAD information
		if pref, err := r.Head(); err == nil {
			vals.Git.Ref = pref.Name().String()
			vals.Git.Commit = pref.Hash().String()
		}

		// Check if the worktree is clean
		if wrk, err := r.Worktree(); err == nil {
			if stat, err := wrk.Status(); err == nil {
				vals.Git.Dirty = !stat.IsClean()
			}
		}
	}

	return vals
}
