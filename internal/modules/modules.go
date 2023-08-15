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

	"github.com/Masterminds/semver/v3"
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

	// dontResolve is used to prevent a module from being resolved
	// if it's already been resolved.
	//
	// This is generally only used if a module doesn't make sense to be
	// resolved again. Examples: local modules and in-memory (vfs) modules.
	dontResolve bool

	// version is the version that was resolved for this module
	version *resolver.Version

	//orbVersion is the shared orb version resolved for this module
	orbVersion *resolver.Version

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
}

// GetModulesForService returns a list of modules that have been resolved from the provided
// service manifest, respecting constraints and channels as needed.
//
// nolint:funlen,gocyclo // Why: Will be refactored in the future
func GetModulesForService(ctx context.Context, opts *ModuleResolveOptions) ([]*Module, error) {
	// start resolving the top-level modules
	modulesToResolve := make([]resolveModule, 0)

	// strReplacements are replacements that replace the URL for a module's
	// provided import path.
	strReplacements := make(map[string]string)

	// resolved contains the current modules that have been selected and is used
	// to track previous resolutions/constraints for re-resolving modules.
	resolved := make(map[string]*resolvedModule)

	if opts.ServiceManifest != nil {
		sm := opts.ServiceManifest

		// for each module required by the service manifest
		// add it to the list of module to be resolved
		for i := range sm.Modules {
			modulesToResolve = append(modulesToResolve, resolveModule{
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
		modulesToResolve = append(modulesToResolve, resolveModule{
			conf:   &configuration.TemplateRepository{Name: opts.Module.Name, Version: "vfs"},
			parent: opts.Module.Name + " (top-level)",
		})
	}

	log := opts.Log

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

		if resolv.conf.Version != "" {
			if _, err := semver.NewConstraint(resolv.conf.Version); err != nil {
				// Attempt to resolve as a branch, which essentially is a channel.
				// This is a bit of a hack, ideally we'll consolidate this logic when
				// channels are ripped out of stencil later.
				resolv.conf.Channel = resolv.conf.Version
				resolv.conf.Version = ""
			}
		}
		// Attemp to resolve the OrbVersion by configuration
		if resolv.conf.OrbVersion != "" {
			if _, err := semver.NewConstraint(resolv.conf.OrbVersion); err != nil {
				URI := "https://" + importPath
				orbVersion, err := getLatestModuleForConstraints(ctx, URI, opts.Token, &resolv, resolved)
				if err != nil {
					return nil, err
				}
				rm.orbVersion = orbVersion
			}
		}

		// if the module has already been resolved and is marked as
		// "dontResolve", then re-use it.
		if rm.dontResolve {
			// this module has already been resolved and should always be used
			log.WithFields(logrus.Fields{
				"module": importPath,
			}).Debug("Using in-memory module")
			modulesToResolve = modulesToResolve[1:]
			continue
		}

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

		// use a different url for the module if it's been replaced
		if _, ok := strReplacements[importPath]; ok {
			uri = strReplacements[importPath]
		}

		// if we're not using a local or in-memory replaced module, resolve the version
		// (local modules & in-memory modules should be treated as always satisfying constraints)
		if !uriIsLocal(uri) && opts.Replacements[importPath] == nil {
			var err error
			version, err = getLatestModuleForConstraints(ctx, uri, opts.Token, &resolv, resolved)
			if err != nil {
				return nil, err
			}
		} else {
			// for local + in-memory modules we don't have a version so just stub it
			version = &resolver.Version{
				Mutable: true,
			}

			// if uri is local, represent it as "local" instead of the full path
			if uriIsLocal(uri) {
				version.Tag = "local"
			} else {
				// otherwise assume in-memory
				version.Tag = "in-memory"
			}
			version.Branch = version.Tag

			// don't attempt to resolve this module again
			rm.dontResolve = true
		}

		var orbVersion string
		if rm.orbVersion != nil {
			orbVersion = rm.orbVersion.GitRef()
		}
		// if we have a replacement for the module in-memory, use that instead
		// of creating a new module that's to be resolved
		if _, ok := opts.Replacements[importPath]; ok {
			m = opts.Replacements[importPath]
		} else {
			var err error
			m, err = New(ctx, uri, &configuration.TemplateRepository{
				Name:       importPath,
				Channel:    resolv.conf.Channel,
				Version:    version.GitRef(),
				OrbVersion: orbVersion,
			})
			if err != nil {
				return nil, err
			}
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
		rm.OrbVersion = m.OrbVersion

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
