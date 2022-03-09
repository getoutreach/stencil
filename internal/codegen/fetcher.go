// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the template repository fetching logic

package codegen

import (
	"context"

	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Fetcher handles all interactions between dependencies
// and fetching of dependencies for stencil. This involes
// creating a vfs, parsing repository manifests, etc.
type Fetcher struct {
	log         logrus.FieldLogger
	m           *configuration.ServiceManifest
	accessToken cfg.SecretData
	extensions  *extensions.Host
}

// NewFetcher creates a new fetcher instance
func NewFetcher(log logrus.FieldLogger, m *configuration.ServiceManifest, accessToken cfg.SecretData,
	extHost *extensions.Host) *Fetcher {
	return &Fetcher{log, m, accessToken, extHost}
}

// ResolveDependencies resolved the dependencies of a given template repository.
// It currently only supports one level dependency resolution and doesn't do any
// smart logic for ordering other than first wins.
func (f *Fetcher) ResolveDependencies(ctx context.Context, filesystems map[string]bool,
	r *configuration.TemplateRepositoryManifest) ([]*modules.Module, error) { //nolint:funlen // Why: will refactor later
	reqModules := make([]*modules.Module, 0)
	args := make(map[string]configuration.Argument)
	for _, d := range r.Modules {
		// If the filesystem already exists, then we can just skip it
		// since something already required it.
		if _, ok := filesystems[d.URL]; ok {
			continue
		}

		m, err := modules.New(d.URL, d.Version)
		if err != nil {
			return nil, err
		}

		mf, err := m.Manifest(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to download repository '%s'", d.URL)
		}
		filesystems[d.URL] = true

		// register an extension if it's found
		if mf.Type == configuration.TemplateRepositoryTypeExt {
			//nolint:govet // Why: We're OK shadowing err
			err := f.extensions.RegisterExtension(ctx, d.URL, mf.Name, d.Version)
			if err != nil {
				return nil, errors.Wrap(err, "failed to download extension")
			}
		}

		for k, v := range mf.Arguments {
			args[k] = v
		}

		subDepModules, err := f.ResolveDependencies(ctx, filesystems, &mf)
		if err != nil {
			return nil, err
		}

		newModules := make([]*modules.Module, 0)

		// only add our filesystem if we're not an extension
		if mf.Type != configuration.TemplateRepositoryTypeExt {
			newModules = append(newModules, m)
		}

		// lookup our dependencies after ours so that we're able to override their files
		// I suspect that this will need to be changed to allow modules to selectively
		// override certain files.
		if len(subDepModules) > 0 {
			newModules = append(newModules, subDepModules...)
		}

		// if we have new filesystems, be sure to append them after our dependencies
		if len(newModules) > 0 {
			reqModules = append(subDepModules, newModules...)
		}
	}

	return reqModules, nil
}
