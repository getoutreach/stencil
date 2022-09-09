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

	// module is the name of the module that imported this module
	module string
}

// constraint is a constraint that can be applied to a module
type constraint struct {
	// str is the string representation of the constraint
	str string

	// module is the name of the module that this constraint originated from
	module string
}

// GetModulesForService returns a list of modules that have been resolved from the provided
// service manifest, respecting constraints and channels as needed.
func GetModulesForService(ctx context.Context, token cfg.SecretData, sm *configuration.ServiceManifest) ([]*Module, error) {
	// start resolving the top-level modules
	modulesToResolve := make([]resolveModule, len(sm.Modules))
	for i := range sm.Modules {
		modulesToResolve[i] = resolveModule{
			conf:   sm.Modules[i],
			module: sm.Name + " (top-level)",
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
		if _, ok := resolved[rm.conf.Name]; !ok {
			resolved[rm.conf.Name] = &resolvedModule{}
		}

		uri := "https://" + rm.conf.Name
		version := ""

		var m *Module

		// if we're using a replacement, update the URI and do not resolve the version
		// if using a file URL.
		if sm.Replacements[rm.conf.Name] != "" {
			uri = sm.Replacements[rm.conf.Name]
		}

		// if we're not using a local module, resolve the version
		if !uriIsLocal(uri) {
			var err error
			version, err = getLatestModuleForConstraints(ctx, token, &rm, resolved)
			if err != nil {
				return nil, err
			}
		}

		m, err := New(ctx, uri, &configuration.TemplateRepository{
			Name:    rm.conf.Name,
			Version: version,
		})
		if err != nil {
			return nil, err
		}

		mf, err := m.Manifest(ctx)
		if err != nil {
			return nil, err
		}
		for i := range mf.Modules {
			modulesToResolve = append(modulesToResolve, resolveModule{
				conf:   mf.Modules[i],
				module: rm.conf.Name,
			})
		}

		// set the module on our resolved module
		resolved[rm.conf.Name].Module = m

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
func getLatestModuleForConstraints(ctx context.Context, token cfg.SecretData,
	m *resolveModule, resolved map[string]*resolvedModule) (string, error) {
	constraints := resolved[m.conf.Name].constraints
	if len(constraints) == 0 {
		constraints = make([]constraint, 0)
	}

	// if we have a constraint, use it to resolve the version
	if m.conf.Version != "" {
		constraints = append(constraints, constraint{
			str:    m.conf.Version,
			module: m.module,
		})
		resolved[m.conf.Name].constraints = constraints
	}

	constraintsStr := make([]string, len(constraints))
	for i := range constraints {
		constraintsStr[i] = constraints[i].str
	}

	v, err := resolver.Resolve(ctx, token, &resolver.Criteria{
		URL:         "https://" + m.conf.Name,
		Channel:     m.conf.Channel,
		Constraints: constraintsStr,
	})
	if err != nil {
		errorString := ""
		for i := range constraints {
			errorString += strings.Repeat(" ", i*2) + "└─ "
			errorString += fmt.Sprintln(constraints[i].module, "wants", constraints[i].str)
		}
		return "", errors.Wrapf(err, "failed to resolve module with constraints\n%s", errorString)
	}

	return v.Tag, nil
}
