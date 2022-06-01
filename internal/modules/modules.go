// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements fetching modules for a given
// service manifest.

// Package modules implements all logic needed for interacting
// with stencil modules and their interaction with a service generated
// by stencil.
package modules

import (
	"context"

	"github.com/blang/semver/v4"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
)

// consideredModule is a module that was considered during
// the dependency resolution process.
type consideredModule struct {
	*Module

	// allowedPrereleases is true if we've considered
	// prereleases for this module
	allowedPrereleases bool
}

// GetModulesForService returns a list of modules that a given service manifest
// depends on. They are not returned in the order of their import.
func GetModulesForService(ctx context.Context, m *configuration.ServiceManifest) ([]*Module, error) {
	// create a map of modules, this is used to avoid downloading the same module twice
	// as well as only ever including one version of a module
	modules := make(map[string]*consideredModule)
	if err := getModulesForService(ctx, m, m.Modules, modules); err != nil {
		return nil, errors.Wrap(err, "failed to fetch modules")
	}

	// map[string]*Module -> []*Module
	rtrn := make([]*Module, 0)
	for k := range modules {
		rtrn = append(rtrn, modules[k].Module)
	}
	return rtrn, nil
}

// IDEA(jaredallard): Log when we're skipping a module and why.

// shouldSkipModule returns true if we should skip downloading this module
func shouldSkipModule(deps map[string]*consideredModule, d *configuration.TemplateRepository) bool {
	existingDep, ok := deps[d.Name]
	if !ok {
		// If it's not already been considered, always download it.
		return false
	}

	if existingDep.Version == localModuleVersion {
		// Never skip override a local module
		// IDEA(jaredallard): Warn when overriding local modules.
		return true
	}

	// If there's a version set explicitly, download it if it's newer
	if d.Version != "" {
		if d.Version == localModuleVersion {
			// If it's a local version, always use it over any other version.
			return false
		}

		// Attempt to parse the two versions as semver, if we fail
		// skip downloading the module and use the existing one.
		newV, err := semver.ParseTolerant(d.Version)
		if err != nil {
			return true
		}

		curV, err := semver.ParseTolerant(existingDep.Version)
		if err != nil {
			return true
		}

		// If the new version is less than the existing version, skip downloading
		// the module/
		if newV.LTE(curV) {
			return true
		}
	}

	if !existingDep.allowedPrereleases && d.Prerelease {
		// If we haven't considered pre-releases and are asking
		// for one, re-download with pre-releases enabled
		return false
	}

	// otherwise, don't skip the module
	return false
}

// getModulesForService recursively fetches all modules for a given service manifest
// this is done by iterating over all dependencies and then recursively calling ourself
// to download their dependencies.
func getModulesForService(ctx context.Context, sm *configuration.ServiceManifest,
	deps []*configuration.TemplateRepository, modules map[string]*consideredModule) error {
	for _, d := range deps {
		if shouldSkipModule(modules, d) {
			continue
		}

		// create a module struct for this module, this resolves the latest version if
		// the version wasn't set.
		m, err := New(ctx, sm.Replacements[d.Name], d)
		if err != nil {
			return errors.Wrapf(err, "failed to create dependency %q", d.Name)
		}

		// prefetch the manifest and FS
		mf, err := m.Manifest(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to parse manifest %q", d.Name)
		}
		modules[d.Name] = &consideredModule{m, d.Prerelease}

		// if we have dependencies, download those now
		if len(mf.Modules) != 0 {
			if err := getModulesForService(ctx, sm, mf.Modules, modules); err != nil {
				return errors.Wrapf(err, "failed to process dependency of %q", d.Name)
			}
		}
	}

	return nil
}
