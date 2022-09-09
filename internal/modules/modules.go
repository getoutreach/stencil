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
)

// resolvedModule is used to keep track of a module during the resolution
// stage, keeping track of the constraints that were used to resolve the
// module's version.
type resolvedModule struct {
	*Module

	// constraints is the stack of constraints that were used to resolve
	// this module. This is used to generate a useful error message if a
	// constraint is violated. This is sorted in descending order.
	constraints []constraint
}

type resolveModule struct {
	// conf is the configuration to be used to resolve the module
	conf *configuration.TemplateRepository

	// parent is the name of the module that imported this module
	parent string
}

// constraint is a constraint that can be applied to a module
type constraint struct {
	// str is the string representation of the constraint
	str string

	// parentModule is the name of the module that this constraint originated from
	parentModule string
}

// GetModulesForService returns a list of modules that have been resolved from the provided
// service manifest, respecting constraints and channels as needed.
func GetModulesForService(ctx context.Context, token cfg.SecretData, sm *configuration.ServiceManifest) ([]*Module, error) {
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

		rm := modulesToResolve[0]
		importPath := rm.conf.Name
		if _, ok := resolved[importPath]; !ok {
			resolved[importPath] = &resolvedModule{Module: &Module{}}
		}

		uri := "https://" + importPath
		version := ""

		var m *Module

		// if we're using a replacement update the url of the module
		if sm.Replacements[importPath] != "" {
			uri = sm.Replacements[importPath]
		}

		// if we're not using a local module, resolve the version
		// (local modules should always satisfy constraints)
		if !uriIsLocal(uri) {
			var err error
			version, err = getLatestModuleForConstraints(ctx, uri, token, &rm, resolved)
			if err != nil {
				return nil, err
			}
		}

		// check if the current module already is this version
		// IDEA(jaredallard): In the future we probably want to see if
		// the past version _also_ satisfies the new constraints, and if so
		// skip re-resolving the module.
		if version == resolved[importPath].Version {
			modulesToResolve = modulesToResolve[1:]
			continue
		}

		m, err := New(ctx, uri, &configuration.TemplateRepository{
			Name:    importPath,
			Channel: rm.conf.Channel,
			Version: version,
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
				parent: importPath,
			})
		}

		// set the module on our resolved module
		resolved[importPath].Module = m

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
	m *resolveModule, resolved map[string]*resolvedModule) (string, error) {
	constraints := resolved[m.conf.Name].constraints
	if len(constraints) == 0 {
		constraints = make([]constraint, 0)
	}

	// if we have a constraint, use it to resolve the version
	if m.conf.Version != "" {
		constraints = append(constraints, constraint{
			str:          m.conf.Version,
			parentModule: m.parent,
		})
		resolved[m.conf.Name].constraints = constraints
	}

	constraintsStr := make([]string, len(constraints))
	for i := range constraints {
		constraintsStr[i] = constraints[i].str
	}

	v, err := resolver.Resolve(ctx, token, &resolver.Criteria{
		URL:         uri,
		Channel:     m.conf.Channel,
		Constraints: constraintsStr,
	})
	if err != nil {
		errorString := ""
		for i := range constraints {
			errorString += strings.Repeat(" ", i*2) + "└─ "
			errorString += fmt.Sprintln(constraints[i].parentModule, "wants", constraints[i].str)
		}
		return "", errors.Wrapf(err, "failed to resolve module with constraints\n%s", errorString)
	}

	return v.Tag, nil
}
