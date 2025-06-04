// Copyright 2024 Outreach Corporation. All Rights Reserved.

// Description: This file provides a threadsafe work list for resolving
// modules. All functions with a workList receiver are threadsafe

package modules

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

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

// newWorkList constructs a list of modules to resolve and a map of string replacements
func newWorkList(opts *ModuleResolveOptions) workList {
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

	rm.mu.Lock()
	// log the resolution attempt
	rm.history = append(rm.history, resolution{
		constraint:   resolv.conf.Version,
		channel:      resolv.conf.Channel,
		parentModule: resolv.parent,
	})
	rm.mu.Unlock()

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

// getLatestModuleForConstraints returns the latest module that satisfies the provided constraints
func (list *workList) getLatestModuleForConstraints(ctx context.Context, item *workItem, token cfg.SecretData) (*resolver.Version, error) {
	constraints := make([]string, 0)
	history := []resolution{}

	m := item.spec
	module := item.inProgressResolution

	module.mu.Lock()
	history = append(history, module.history...)
	module.mu.Unlock()
	defer func() {
		module.mu.Lock()
		module.history = history
		module.mu.Unlock()
	}()

	for _, r := range history {
		if r.constraint != "" {
			constraints = append(constraints, r.constraint)
		}
	}

	channel := m.conf.Channel
	for _, r := range history {
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
	if module.version != nil && module.version.Mutable {
		// IDEA(jaredallard): We should log this as it's non-deterministic when we
		// have a good interface for doing so.
		return module.version, nil
	}

	cacheFile := filepath.Join(StencilCacheDir(), "module_version",
		ModuleCacheDirectory(item.uri, item.spec.conf.Channel), "version.json")

	if useModuleCache(cacheFile) {
		data, err := os.ReadFile(cacheFile)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read cache resolved version"+
				" from cache for mddule %s:%s", item.uri, item.spec.conf.Channel)
		}

		var cached *resolver.Version
		err = json.Unmarshal(data, cached)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to deserialize cached version for module %s:%s", item.uri, item.spec.conf.Channel)
		}

		return cached, nil
	}

	v, err := resolver.Resolve(ctx, token, &resolver.Criteria{
		URL:           item.uri,
		Channel:       channel,
		Constraints:   constraints,
		AllowBranches: true,
	})
	if err != nil {
		errorString := ""

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

	data, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize version for module %s:%s", item.uri, item.spec.conf.Channel)
	}

	err = os.WriteFile(cacheFile, data, 0o600)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write resolved version"+
			" to cache for module %s:%s", item.uri, item.spec.conf.Channel)
	}

	return v, nil
}
