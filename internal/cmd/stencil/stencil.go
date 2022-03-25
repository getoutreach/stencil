// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description

// Package stencil implements the stencil command, which is
// essentially a thing wrapper around the codegen package
// which does most of the heavy lifting.
package stencil

import (
	"context"
	"os"
	"path/filepath"

	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Command is a thin wrapper around the codegen package that
// implements the "stencil" command.
type Command struct {
	// manifest is the service manifest that is being used
	// for this template render
	manifest *configuration.ServiceManifest

	// log is the logger used for logging output
	log logrus.FieldLogger

	// dryRun denotes if we should write files to disk or not
	dryRun bool
}

// NewCommand creates a new stencil command
func NewCommand(log logrus.FieldLogger, s *configuration.ServiceManifest, dryRun bool) *Command {
	_, err := stencil.LoadLockfile("")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.WithError(err).Warn("failed to load lockfile")
	}

	return &Command{
		manifest: s,
		log:      log,
		dryRun:   dryRun,
	}
}

// Run fetches dependencies of the root modules and builds the layered filesystem,
// after that GenerateFiles is called to actually walk the filesystem and render
// the templates. This step also does minimal post-processing of the dependencies
// manifests
func (c *Command) Run(ctx context.Context) error {
	mods, err := modules.GetModulesForService(ctx, c.manifest)
	if err != nil {
		return errors.Wrap(err, "failed to process modules list")
	}
	st := codegen.NewStencil(c.manifest, mods)

	c.log.Info("Loading native extensions")
	if err := st.RegisterExtensions(ctx); err != nil {
		return err
	}

	c.log.Info("Rendering templates")
	tpls, err := st.Render(ctx, c.log)
	if err != nil {
		return err
	}

	// Below options mutate, so we shallow return
	if c.dryRun {
		return nil
	}

	if err := c.writeFiles(st, tpls); err != nil {
		return err
	}

	return st.PostRun(ctx, c.log)
}

// writeFile writes a codegen.File to disk based on its current state
func (c *Command) writeFile(f *codegen.File) error {
	action := "Created"
	if f.Deleted {
		action = "Deleted"
		os.Remove(f.Name())
	} else if f.Skipped {
		action = "Skipped"
	} else if _, err := os.Stat(f.Name()); err == nil {
		action = "Updated"
	}

	if action == "Created" || action == "Updated" {
		if err := os.MkdirAll(filepath.Dir(f.Name()), 0o755); err != nil {
			return errors.Wrapf(err, "failed to ensure directory for %q existed", f.Name())
		}

		if err := os.WriteFile(f.Name(), f.Bytes(), f.Mode()); err != nil {
			return errors.Wrapf(err, "failed to create %q", f.Name())
		}
	}

	c.log.Infof("  -> %s %s", action, f.Name())
	return nil
}

// writeFiles writes the files to disk
func (c *Command) writeFiles(st *codegen.Stencil, tpls []*codegen.Template) error {
	c.log.Infof("Writing template(s) to disk")
	for _, tpl := range tpls {
		c.log.Debugf(" -> %s (%s)", tpl.Module.Name, tpl.Path)
		for _, f := range tpl.Files {
			if err := c.writeFile(f); err != nil {
				return err
			}
		}
	}

	l := st.GenerateLockfile(tpls)
	f, err := os.Create(stencil.LockfileName)
	if err != nil {
		return errors.Wrap(err, "failed to create lockfile")
	}
	defer f.Close()

	return errors.Wrap(yaml.NewEncoder(f).Encode(l),
		"failed to encode lockfile into yaml")
}
