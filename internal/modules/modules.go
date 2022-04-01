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
	"path"
	"strings"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/pkg/errors"
	giturls "github.com/whilp/git-urls"
)

// IDEA(jaredallard): Remove m.URL support soon.

// GetModulesForService returns a list of modules that a given service manifest
// depends on. They are not returned in the order of their import.
func GetModulesForService(ctx context.Context, m *configuration.ServiceManifest,
	frozen bool, l *stencil.Lockfile) ([]*Module, error) {
	// If frozen, we iterate over the lockfile to set the versions
	if frozen {
		if l == nil {
			return nil, fmt.Errorf("frozen lockfile requires a lockfile to exist")
		}

		for _, m := range m.Modules {
			// Convert m.URL -> m.Name
			//nolint:staticcheck // Why: We're implementing compat here.
			if m.URL != "" {
				u, err := giturls.Parse(m.URL) //nolint:staticcheck // Why: We're implementing compat here.
				if err != nil {
					//nolint:staticcheck // Why: We're implementing compat here.
					return nil, errors.Wrapf(err, "failed to parse deprecated url module syntax %q as a URL", m.URL)
				}
				m.Name = path.Join(u.Host, u.Path)
			}

			for _, l := range l.Modules {
				if m.Name == l.Name {
					if strings.HasPrefix(l.URL, "file://") {
						return nil,
							fmt.Errorf("cannot use frozen lockfile for file dependency %q, re-add replacement or run without --frozen-lockfile", l.Name)
					}

					m.Version = l.Version
					break
				}
			}
			if m.Version == "" {
				return nil, fmt.Errorf("frozen lockfile, but no version found for module %q", m.Name)
			}
		}
	}

	// create a map of modules, this is used to avoid downloading the same module twice
	// as well as only ever including one version of a module
	modules := make(map[string]*Module)
	if err := getModulesForService(ctx, m, m.Modules, modules, l); err != nil {
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
	deps []*configuration.TemplateRepository, modules map[string]*Module, l *stencil.Lockfile) error {
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

		// If we already used this dependency once, don't fetch it again.
		if _, ok := modules[d.Name]; ok {
			continue
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
			if err := getModulesForService(ctx, sm, mf.Modules, modules, l); err != nil {
				return errors.Wrapf(err, "failed to process dependency of %q", d.Name)
			}
		}
	}

	return nil
}
