// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description

// Package stencil implements the stencil command, which is
// essentially a thing wrapper around the codegen package
// which does most of the heavy lifting.
package stencil

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/blang/semver/v4"
	"github.com/charmbracelet/glamour"
	"github.com/getoutreach/gobox/pkg/cfg"
	"github.com/getoutreach/gobox/pkg/cli/github"
	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/modules"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/getoutreach/stencil/pkg/stencil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	giturls "github.com/whilp/git-urls"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// Command is a thin wrapper around the codegen package that
// implements the "stencil" command.
type Command struct {
	// lock is the current stencil lockfile at command creation time
	lock *stencil.Lockfile

	// manifest is the service manifest that is being used
	// for this template render
	manifest *configuration.ServiceManifest

	// log is the logger used for logging output
	log logrus.FieldLogger

	// dryRun denotes if we should write files to disk or not
	dryRun bool

	// frozenLockfile denotes if we should use versions from the lockfile
	// or not
	frozenLockfile bool

	// allowMajorVersionUpgrade denotes if we should allow major version
	// upgrades without a prompt or not
	allowMajorVersionUpgrades bool

	// token is the github token used for fetching modules
	token cfg.SecretData
}

// NewCommand creates a new stencil command
func NewCommand(log logrus.FieldLogger, s *configuration.ServiceManifest,
	dryRun, frozen, usePrerelease, allowMajorVersionUpgrades bool) *Command {
	l, err := stencil.LoadLockfile("")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.WithError(err).Warn("failed to load lockfile")
	}

	if usePrerelease {
		//nolint:lll // Why: It's a long warning string :'(
		log.Warn("Deprecated: --use-prerelease is deprecated. Set 'rc' as the channel on each module you want to use pre-releases for in the service.yaml instead")
		for i := range s.Modules {
			s.Modules[i].Channel = "rc"
			s.Modules[i].Version = ""
		}
	}

	token, err := github.GetToken()
	if err != nil {
		log.Warn("failed to get github token, using anonymous access")
	}

	return &Command{
		lock:                      l,
		manifest:                  s,
		log:                       log,
		dryRun:                    dryRun,
		frozenLockfile:            frozen,
		allowMajorVersionUpgrades: allowMajorVersionUpgrades,
		token:                     token,
	}
}

// Run fetches dependencies of the root modules and builds the layered filesystem,
// after that GenerateFiles is called to actually walk the filesystem and render
// the templates. This step also does minimal post-processing of the dependencies
// manifests
func (c *Command) Run(ctx context.Context) error {
	if c.frozenLockfile {
		if err := c.useModulesFromLock(); err != nil {
			return errors.Wrap(err, "failed to use lockfile for modules")
		}
	}

	c.log.Info("Fetching dependencies")
	mods, err := modules.GetModulesForService(ctx, c.token, c.manifest, c.log)
	if err != nil {
		return errors.Wrap(err, "failed to process modules list")
	}

	if err := c.checkForMajorVersions(ctx, mods); err != nil {
		return errors.Wrap(err, "failed to handle major version upgrade")
	}

	for _, m := range mods {
		c.log.Infof(" -> %s %s", m.Name, m.Version)
	}

	st := codegen.NewStencil(c.manifest, mods, c.log)
	defer st.Close()

	c.log.Info("Loading native extensions")
	if err := st.RegisterExtensions(ctx); err != nil {
		return err
	}

	c.log.Info("Rendering templates")
	tpls, err := st.Render(ctx, c.log)
	if err != nil {
		return err
	}

	if err := c.writeFiles(st, tpls); err != nil {
		return err
	}

	// Can't dry run post run yet
	if c.dryRun {
		c.log.Info("Skipping post-run commands, dry-run")
		return nil
	}

	return st.PostRun(ctx, c.log)
}

// useModulesFromLock uses the modules from the lockfile instead
// of the latest versions, or manually supplied versions in the
// service manifest.
func (c *Command) useModulesFromLock() error {
	if c.lock == nil {
		return fmt.Errorf("frozen lockfile requires a lockfile to exist")
	}

	for _, m := range c.manifest.Modules {
		// Convert m.URL -> m.Name
		//nolint:staticcheck // Why: We're implementing compat here.
		if m.URL != "" {
			u, err := giturls.Parse(m.URL) //nolint:staticcheck // Why: We're implementing compat here.
			if err != nil {
				//nolint:staticcheck // Why: We're implementing compat here.
				return errors.Wrapf(err, "failed to parse deprecated url module syntax %q as a URL", m.URL)
			}
			m.Name = path.Join(u.Host, u.Path)
		}

		for _, l := range c.lock.Modules {
			if m.Name == l.Name {
				if strings.HasPrefix(l.URL, "file://") {
					return fmt.Errorf("cannot use frozen lockfile for file dependency %q, re-add replacement or run without --frozen-lockfile", l.Name)
				}

				m.Version = l.Version
				break
			}
		}
		if m.Version == "" {
			return fmt.Errorf("frozen lockfile, but no version found for module %q", m.Name)
		}
	}

	return nil
}

// checkForMajorVersions checks to see if a major version bump has occurred,
// if it is, we report it to the user before progressing.
func (c *Command) checkForMajorVersions(ctx context.Context, mods []*modules.Module) error {
	// skip if no lockfile
	if c.lock == nil {
		return nil
	}

	lastUsedMods := make(map[string]*stencil.LockfileModuleEntry)
	for _, l := range c.lock.Modules {
		lastUsedMods[l.Name] = l
	}

	for _, m := range mods {
		// skip unknown modules
		lastm, ok := lastUsedMods[m.Name]
		if !ok {
			continue
		}

		lastV, err := semver.ParseTolerant(lastm.Version)
		if err != nil {
			continue
		}

		newV, err := semver.ParseTolerant(m.Version)
		if err != nil {
			continue
		}

		// skip major versions that are less or equal to our last version
		if newV.Major <= lastV.Major {
			continue
		}

		if err := c.promptMajorVersion(ctx, m, lastm); err != nil {
			return err
		}
	}

	return nil
}

// promptMajorVersion prompts the user to upgrade their templates
func (c *Command) promptMajorVersion(ctx context.Context, m *modules.Module, lastm *stencil.LockfileModuleEntry) error {
	c.log.Infof("Major version bump detected for %q (%s -> %s)", m.Name, lastm.Version, m.Version)
	if c.allowMajorVersionUpgrades {
		c.log.Info("Continuing with major version upgrade, --allow-major-version-upgrades was set")
		return nil
	}

	// If we're not a terminal, we can't ask for consent
	// so we error out informing the user how to fix this.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("unable to prompt for major version upgrade, stdin is not a terminal, pass --allow-major-version-upgrades to continue")
	}

	gh, err := github.NewClient(github.WithAllowUnauthenticated(), github.WithLogger(c.log))
	if err != nil {
		return errors.Wrap(err, "failed to fetch release notes (create github client)")
	}

	spl := strings.Split(m.Name, "/")
	if len(spl) < 3 {
		return fmt.Errorf("unsupported major version upgrade for module %q", m.Name)
	}

	rel, _, err := gh.Repositories.GetReleaseByTag(ctx, spl[1], spl[2], m.Version)
	if err != nil {
		return errors.Wrap(err, "failed to fetch release notes")
	}

	out := rel.GetBody()
	r, err := glamour.NewTermRenderer(glamour.WithAutoStyle())
	if err == nil {
		out, err = r.Render(rel.GetBody())
		if err != nil {
			c.log.WithError(err).Warn("Failed to render release notes, using raw release notes")
		}
	} else if err != nil {
		c.log.WithError(err).Warn("Failed to create markdown render, using raw release notes")
	}

	fmt.Println(out)

	var proceed bool
	if err := survey.Ask([]*survey.Question{{
		Name: "proceed",
		Prompt: &survey.Confirm{
			Message: fmt.Sprintf("Proceed with upgrade for module %q (%s -> %s)?", m.Name, lastm.Version, m.Version),
			Default: true,
		},
	}}, &proceed); err != nil {
		return err
	}
	if !proceed {
		return fmt.Errorf("Not updating, re-run with --frozen-lockfile to proceed")
	}

	return nil
}

// writeFile writes a codegen.File to disk based on its current state
func (c *Command) writeFile(f *codegen.File) error {
	action := "Created"
	if f.Deleted {
		action = "Deleted"

		if !c.dryRun {
			os.Remove(f.Name())
		}
	} else if f.Skipped {
		action = "Skipped"
	} else if _, err := os.Stat(f.Name()); err == nil {
		action = "Updated"
	}

	if action == "Created" || action == "Updated" {
		if !c.dryRun {
			if err := os.MkdirAll(filepath.Dir(f.Name()), 0o755); err != nil {
				return errors.Wrapf(err, "failed to ensure directory for %q existed", f.Name())
			}

			if err := os.WriteFile(f.Name(), f.Bytes(), f.Mode()); err != nil {
				return errors.Wrapf(err, "failed to create %q", f.Name())
			}
		}
	}

	msg := fmt.Sprintf("  -> %s %s", action, f.Name())
	if c.dryRun {
		msg += " (dry-run)"
	}
	c.log.Info(msg)
	return nil
}

// writeFiles writes the files to disk
func (c *Command) writeFiles(st *codegen.Stencil, tpls []*codegen.Template) error {
	c.log.Infof("Writing template(s) to disk")
	for _, tpl := range tpls {
		c.log.Debugf(" -> %s (%s)", tpl.Module.Name, tpl.Path)
		for i := range tpl.Files {
			if err := c.writeFile(tpl.Files[i]); err != nil {
				return err
			}
		}
	}

	// Don't generate a lockfile in dry-run mode
	if c.dryRun {
		return nil
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
