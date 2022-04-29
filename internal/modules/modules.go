// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements fetching modules for a given
// service manifest.

// Package modules implements all logic needed for interacting
// with stencil modules and their interaction with a service generated
// by stencil.
package modules

import (
	"context"
	"path"

	"github.com/blang/semver/v4"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	giturls "github.com/whilp/git-urls"
)

// IDEA(jaredallard): Remove m.URL support soon.

// GetModulesForService returns a list of modules that a given service manifest
// depends on. They are not returned in the order of their import.
func GetModulesForService(ctx context.Context, m *configuration.ServiceManifest) ([]*Module, error) {
	// create a map of modules, this is used to avoid downloading the same module twice
	// as well as only ever including one version of a module
	modules := make(map[string]*Module)
	if err := getModulesForService(ctx, m, m.Modules, modules); err != nil {
		return nil, errors.Wrap(err, "failed to fetch modules")
	}

	// map[string]*Module -> []*Module
	rtrn := make([]*Module, 0)
	for k := range modules {
		rtrn = append(rtrn, modules[k])
	}
	return rtrn, nil
}

// getModulesForService recursively fetches all modules for a given service manifest
// this is done by iterating over all dependencies and then recursively calling ourself
// to download their dependencies.
func getModulesForService(ctx context.Context, sm *configuration.ServiceManifest,
	deps []*configuration.TemplateRepository, modules map[string]*Module) error {
	for _, d := range deps {
		// Convert d.URL -> d.Name
		//nolint:staticcheck // Why: We're implementing compat here.
		if d.URL != "" {
			u, err := giturls.Parse(d.URL) //nolint:staticcheck // Why: We're implementing compat here.
			if err != nil {
				//nolint:staticcheck // Why: We're implementing compat here.
				return errors.Wrapf(err, "failed to parse deprecated url module syntax %q as a URL", d.URL)
			}
			d.Name = path.Join(u.Host, u.Path)
		}

		// If we already used this dependency once, only fetch it again if
		// this is a newer version to enable modules to request a specific version.
		//
		// IDEA: In the future we should probably warn the user when this happens
		// because it's probably non-deterministic.
		if _, ok := modules[d.Name]; ok {
			newV, err := semver.ParseTolerant(d.Version)
			if err != nil {
				// Handle when we're using a local version
				newV = semver.MustParse("0.0.0")
			}
			curV, err := semver.ParseTolerant(modules[d.Name].Version)
			if err != nil {
				curV = semver.MustParse("0.0.0")
			}

			// The new version is less than the one we already found, so
			// we skip downloading the module here.
			if newV.LTE(curV) {
				continue
			}

			// GC the old module
			modules[d.Name] = nil
		}

		// create a module struct for this module, this resolves the latest version if
		// the version wasn't set.
		m, err := New(ctx, d.Name, sm.Replacements[d.Name], d.Version)
		if err != nil {
			return errors.Wrapf(err, "failed to create dependency %q", d.Name)
		}

		// prefetch the manifest and FS
		mf, err := m.Manifest(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to parse manifest %q", d.Name)
		}
		modules[d.Name] = m

		// if we have dependencies, download those now
		if len(mf.Modules) != 0 {
			if err := getModulesForService(ctx, sm, mf.Modules, modules); err != nil {
				return errors.Wrapf(err, "failed to process dependency of %q", d.Name)
			}
		}
	}

	return nil
}
