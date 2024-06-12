// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements fetching modules for a given
// service manifest.

// Package modules implements all logic needed for interacting
// with stencil modules and their interaction with a service generated
// by stencil.
package modules

import (
	"context"
	"sync"

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// resolvedModule is used to keep track of a module during the resolution
// stage, keeping track of the constraints that were used to resolve the
// module's version.
type resolvedModule struct {
	*Module

	// dontResolve is used to prevent a module from being resolved
	// if it's already been resolved.
	//
	// This is generally only used if a module doesn't make sense to be
	// resolved again. Examples: local modules and in-memory (vfs) modules.
	dontResolve bool

	// version is the version that was resolved for this module
	version *resolver.Version

	// history is the stack of the criteria that was used to resolve
	// this module. This is used to generate a useful error message if a
	// constraint, or other import condition, is violated.
	// This is sorted in descending order.
	history []resolution

	// mu protects the history slice
	mu sync.Mutex
}

// resolveModule is used to keep track of a module that needs to be resolved
type resolveModule struct {
	// conf is the configuration to be used to resolve the module
	conf *configuration.TemplateRepository

	// parent is the name of the module that imported this module
	parent string
}

// resolution is an entry in the resolution stack that was used to resolve a module
type resolution struct {
	// constraint is the string representation of the constraint
	// used by the parent module to resolve this module
	constraint string

	// channel is the channel that was used to resolve this module
	channel string

	// parentModule is the name of the module that imported this module
	parentModule string
}

// ModuleResolveOptions contains options for resolving modules
type ModuleResolveOptions struct {
	// Token is the token to use to resolve modules
	Token cfg.SecretData

	// Log is the logger to use
	Log logrus.FieldLogger

	// ServiceManifest is the manifest to resolve modules for.
	// This can only be supplied if Module is not set.
	ServiceManifest *configuration.ServiceManifest

	// Module is the module to resolve dependencies for.
	// This can only be supplied if ServiceManifest is not
	// set. This module is automatically added as a
	// Replacement.
	Module *Module

	// Replacements is a map of modules to use instead of ones specified
	// in the manifest. This is mainly meant for tests/importing of stencil
	// as this is not resolved and instead requires a module type to be
	// passed.
	Replacements map[string]*Module

	// ConcurrentResolvers is the number of concurrent resolvers to use
	// when resolving modules.
	ConcurrentResolvers int
}

// GetModulesForService returns a list of modules that have been resolved from the provided
// service manifest, respecting constraints and channels as needed.
//
//nolint:funlen // Why: Will be refactored in the future
func GetModulesForService(ctx context.Context, opts *ModuleResolveOptions) ([]*Module, error) {
	wl := newWorkList(opts)

	log := opts.Log

	g := errgroup.Group{}
	// setting it to a default value if we got here via a path other than the
	// CLI. E.g. a test
	if opts.ConcurrentResolvers == 0 {
		opts.ConcurrentResolvers = 5
	}
	g.SetLimit(opts.ConcurrentResolvers)

	for len(wl.tasks) > 0 {
		for {
			// get an item
			item := wl.pop()
			if item == nil {
				break
			}
			// if it's marked "dont resolve", skip it
			if item.inProgressResolution.dontResolve {
				continue
			}

			// resolve the module in a goroutine
			g.Go(func() error { return work(ctx, opts, item, &wl, log) })
		}
		// wait for all the goroutines to finish
		err := g.Wait()
		if err != nil {
			return nil, err
		}
		// check if we have any more modules to resolve
	}

	// convert the resolved modules to a list of modules
	modules := make([]*Module, 0, len(wl.resolved))
	for _, m := range wl.resolved {
		modules = append(modules, m.Module)
	}
	return modules, nil
}

// work does the actual work of resolving a module
func work(ctx context.Context, opts *ModuleResolveOptions, item *workItem, wl *workList,
	log logrus.FieldLogger,
) error {
	var m *Module
	var version *resolver.Version

	// if we're not using a local or in-memory replaced module, resolve the version
	// (local modules & in-memory modules should be treated as always satisfying constraints)
	if !uriIsLocal(item.uri) && opts.Replacements[item.importPath] == nil {
		var err error
		version, err = wl.getLatestModuleForConstraints(ctx, item, opts.Token)
		if err != nil {
			return err
		}
	} else {
		// for local + in-memory modules we don't have a version so just stub it
		version = &resolver.Version{
			Mutable: true,
		}

		// assume in-memory
		version.Tag = "in-memory"
		// ... but if uri is local, represent it as "local" instead of the full path
		if uriIsLocal(item.uri) {
			version.Tag = "local"
		}

		version.Branch = version.Tag

		// don't attempt to resolve this module again
		item.inProgressResolution.dontResolve = true
	}

	// if we have a replacement for the module in-memory, use that instead
	// of creating a new module that's to be resolved
	if _, ok := opts.Replacements[item.importPath]; ok {
		m = opts.Replacements[item.importPath]
	} else {
		var err error
		m, err = New(ctx, item.uri, &configuration.TemplateRepository{
			Name:    item.importPath,
			Channel: item.spec.conf.Channel,
			Version: version.GitRef(),
		})
		if err != nil {
			return err
		}
	}

	mf, err := m.Manifest(ctx)
	if err != nil {
		return err
	}

	// add the dependencies of this module to the stack to be resolved
	for i := range mf.Modules {
		wl.push(&resolveModule{
			conf:   mf.Modules[i],
			parent: item.importPath + "@" + version.String(),
		})
	}

	// set the module on our resolved module
	item.inProgressResolution.Module = m
	item.inProgressResolution.version = version

	log.WithFields(logrus.Fields{
		"module":  item.importPath,
		"version": version.GitRef(),
	}).Debug("resolved module")
	return nil
}
