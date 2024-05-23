// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements fetching modules for a given
// service manifest.

// Package modules implements all logic needed for interacting
// with stencil modules and their interaction with a service generated
// by stencil.
package modules

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
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
	wl := buildWorkLists(opts)

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

// workList is a list of modules to resolve
type workList struct {
	tasks []*resolveModule
	// replacements replace the URL for a module's
	// provided import path.
	replacements map[string]string

	// resolved contains the current modules that have been selected and is used
	// to track previous resolutions/constraints for re-resolving modules.
	resolved map[string]*resolvedModule

	// mu protects all the collections in this struct
	mu  sync.Mutex
	log logrus.FieldLogger
}

// workItem is a single item in the work list
type workItem struct {
	importPath           string
	inProgressResolution *resolvedModule
	spec                 *resolveModule
	uri                  string
}

// pop removes and returns the first item in the work list
func (list *workList) pop() *workItem {
	list.mu.Lock()
	defer list.mu.Unlock()

	if len(list.tasks) == 0 {
		return nil
	}
	resolv := list.tasks[0]
	list.tasks = list.tasks[1:]

	if resolv.conf.Version != "" {
		if _, err := semver.NewConstraint(resolv.conf.Version); err != nil {
			// Attempt to resolve as a branch, which essentially is a channel.
			// This is a bit of a hack, ideally we'll consolidate this logic when
			// channels are ripped out of stencil later.
			resolv.conf.Channel = resolv.conf.Version
			resolv.conf.Version = ""
		}
	}

	// check if it has already been resolved
	importPath := resolv.conf.Name
	if _, ok := list.resolved[importPath]; !ok {
		list.resolved[importPath] = &resolvedModule{Module: &Module{}}
	}
	rm := list.resolved[importPath]

	// if the module has already been resolved and is marked as
	// "dontResolve", then re-use it.
	if rm.dontResolve {
		// this module has already been resolved and should always be used
		list.log.WithFields(logrus.Fields{
			"module": importPath,
		}).Debug("Using in-memory module")
		return &workItem{importPath: importPath, inProgressResolution: rm, spec: resolv}
	}

	// log the resolution attempt
	rm.history = append(rm.history, resolution{
		constraint:   resolv.conf.Version,
		channel:      resolv.conf.Channel,
		parentModule: resolv.parent,
	})
	list.log.WithFields(logrus.Fields{
		"module": importPath,
		"parent": resolv.parent,
	}).Debug("resolving module")

	uri := "https://" + importPath
	// use a different url for the module if it's been replaced
	if _, ok := list.replacements[importPath]; ok {
		uri = list.replacements[importPath]
	}

	return &workItem{importPath: importPath, inProgressResolution: rm, spec: resolv, uri: uri}
}

// push adds a module to the work list
func (list *workList) push(task *resolveModule) {
	list.mu.Lock()
	defer list.mu.Unlock()

	list.tasks = append(list.tasks, task)
}

// buildWorkLists constructs a list of modules to resolve and a map of string replacements
func buildWorkLists(opts *ModuleResolveOptions) workList {
	// start resolving the top-level modules
	modulesToResolve := make([]*resolveModule, 0)

	strReplacements := make(map[string]string)

	if opts.ServiceManifest != nil {
		sm := opts.ServiceManifest

		// for each module required by the service manifest
		// add it to the list of module to be resolved
		for i := range sm.Modules {
			modulesToResolve = append(modulesToResolve, &resolveModule{
				conf:   sm.Modules[i],
				parent: sm.Name + " (top-level)",
			})
		}

		// add the replacements to the string list of replacements
		for k, v := range sm.Replacements {
			strReplacements[k] = v
		}
	} else if opts.Module != nil {
		if opts.Replacements == nil {
			opts.Replacements = make(map[string]*Module)
		}

		// add the module to the replacements map so that it can be
		// used as a replacement for itself
		opts.Replacements[opts.Module.Name] = opts.Module

		// add ourself as the top-level module to resolve our
		// dependencies without duplicating code here
		modulesToResolve = append(modulesToResolve, &resolveModule{
			conf:   &configuration.TemplateRepository{Name: opts.Module.Name, Version: "vfs"},
			parent: opts.Module.Name + " (top-level)",
		})
	}
	return workList{
		tasks:        modulesToResolve,
		replacements: strReplacements,
		log:          opts.Log,
		resolved:     make(map[string]*resolvedModule),
	}
}

// getLatestModuleForConstraints returns the latest module that satisfies the provided constraints
func (list *workList) getLatestModuleForConstraints(ctx context.Context, item *workItem, token cfg.SecretData) (*resolver.Version, error) {
	constraints := make([]string, 0)
	m := item.spec

	list.mu.Lock()
	module, ok := list.resolved[m.conf.Name]
	list.mu.Unlock()

	for _, r := range module.history {
		if r.constraint != "" {
			constraints = append(constraints, r.constraint)
		}
	}

	channel := m.conf.Channel
	for _, r := range module.history {
		// if we don't have a channel, or the channel is stable, check to see if
		// the channel we last resolved with doesn't match the current channel requested.
		//
		// If it doesn't match, we don't know how to resolve the module, so we error.
		if channel != "" && channel != resolver.StableChannel && r.channel != channel {
			return nil, fmt.Errorf("unable to resolve module %s: "+
				"module was previously resolved with channel %s (parent: %s), but now requires channel %s",
				m.conf.Name, r.channel, r.parentModule, channel)
		}

		// use the first history entry that has a channel since we can't have multiple channels
		channel = r.channel
		break //nolint:staticcheck // Why: see above comment
	}

	// If the last version we resolved is mutable, it's impossible for us
	// to compare the two, so we have to use it.
	if ok {
		if module.version != nil && module.version.Mutable {
			// IDEA(jaredallard): We should log this as it's non-deterministic when we
			// have a good interface for doing so.
			return module.version, nil
		}
	}

	v, err := resolver.Resolve(ctx, token, &resolver.Criteria{
		URL:           item.uri,
		Channel:       channel,
		Constraints:   constraints,
		AllowBranches: true,
	})
	if err != nil {
		errorString := ""
		history := module.history
		for i := range history {
			h := &history[i]
			errorString += strings.Repeat(" ", i*2) + "└─ "

			wants := "*"
			if h.constraint != "" {
				wants = h.constraint
			} else if h.channel != "" {
				wants = "(channel) " + h.channel
			}

			errorString += fmt.Sprintln(history[i].parentModule, "wants", wants)
		}
		return nil, errors.Wrapf(err, "failed to resolve module '%s' with constraints\n%s", m.conf.Name, errorString)
	}

	return v, nil
}
