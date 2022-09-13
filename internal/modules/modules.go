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

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/updater/resolver"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// resolvedModule is used to keep track of a module during the resolution
// stage, keeping track of the constraints that were used to resolve the
// module's version.
type resolvedModule struct {
	*Module

	// version is the version that was resolved for this module
	version *resolver.Version

	// history is the stack of history that were used to resolve
	// this module. This is used to generate a useful error message if a
	// constraint, or other import condition,
	//  is violated. This is sorted in descending order.
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

// GetModulesForService returns a list of modules that have been resolved from the provided
// service manifest, respecting constraints and channels as needed.
func GetModulesForService(ctx context.Context, token cfg.SecretData,
	sm *configuration.ServiceManifest, log logrus.FieldLogger) ([]*Module, error) {
	// start resolving the top-level modules
	modulesToResolve := make([]resolveModule, len(sm.Modules))
	for i := range sm.Modules {
		modulesToResolve[i] = resolveModule{
			conf:   sm.Modules[i],
			parent: sm.Name + " (top-level)",
		}
	}

	// resolved contains the current modules that have been selected and is used
	// to track previous resolutions/constraints for re-resolving modules.
	resolved := make(map[string]*resolvedModule)

	// resolve all versions, adding more to the stack as we go
	for {
		// done resolving the modules
		if len(modulesToResolve) == 0 {
			break
		}

		resolv := modulesToResolve[0]
		importPath := resolv.conf.Name
		if _, ok := resolved[importPath]; !ok {
			resolved[importPath] = &resolvedModule{Module: &Module{}}
		}
		rm := resolved[importPath]

		// log the resolution attempt
		rm.history = append(rm.history, resolution{
			constraint:   resolv.conf.Version,
			channel:      resolv.conf.Channel,
			parentModule: resolv.parent,
		})
		log.WithFields(logrus.Fields{
			"module": importPath,
			"parent": resolv.parent,
		}).Debug("resolving module")

		uri := "https://" + importPath
		var version *resolver.Version

		var m *Module

		// if we're using a replacement update the url of the module
		if _, ok := sm.Replacements[importPath]; ok {
			uri = sm.Replacements[importPath]
		}

		// if we're not using a local module, resolve the version
		// (local modules should always satisfy constraints)
		if !uriIsLocal(uri) {
			var err error
			version, err = getLatestModuleForConstraints(ctx, uri, token, &resolv, resolved)
			if err != nil {
				return nil, err
			}
		} else {
			// for local modules we don't have a version and the module New() handles this,
			// so just create a stub mutable version
			version = &resolver.Version{
				Mutable: true,
			}
		}

		m, err := New(ctx, uri, &configuration.TemplateRepository{
			Name:    importPath,
			Channel: resolv.conf.Channel,
			Version: version.GitRef(),
		})
		if err != nil {
			return nil, err
		}

		mf, err := m.Manifest(ctx)
		if err != nil {
			return nil, err
		}

		// add the dependencies of this module to the stack to be resolved
		for i := range mf.Modules {
			modulesToResolve = append(modulesToResolve, resolveModule{
				conf:   mf.Modules[i],
				parent: importPath + "@" + version.String(),
			})
		}

		// set the module on our resolved module
		rm.Module = m
		rm.version = version

		log.WithFields(logrus.Fields{
			"module":  importPath,
			"version": version.GitRef(),
		}).Debug("resolved module")

		// resolve the next module
		modulesToResolve = modulesToResolve[1:]
	}

	// convert the resolved modules to a list of modules
	modules := make([]*Module, 0, len(resolved))
	for _, m := range resolved {
		modules = append(modules, m.Module)
	}
	return modules, nil
}

// getLatestModuleForConstraints returns the latest module that satisfies the provided constraints
func getLatestModuleForConstraints(ctx context.Context, uri string, token cfg.SecretData,
	m *resolveModule, resolved map[string]*resolvedModule) (*resolver.Version, error) {
	constraints := make([]string, 0)
	for _, r := range resolved[m.conf.Name].history {
		if r.constraint != "" {
			constraints = append(constraints, r.constraint)
		}
	}

	channel := m.conf.Channel
	for _, r := range resolved[m.conf.Name].history {
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
	if rm, ok := resolved[m.conf.Name]; ok {
		if rm.version != nil && rm.version.Mutable {
			// IDEA(jaredallard): We should log this as it's non-deterministic when we
			// have a good interface for doing so.
			return rm.version, nil
		}
	}

	v, err := resolver.Resolve(ctx, token, &resolver.Criteria{
		URL:           uri,
		Channel:       channel,
		Constraints:   constraints,
		AllowBranches: true,
	})
	if err != nil {
		errorString := ""
		history := resolved[m.conf.Name].history
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
