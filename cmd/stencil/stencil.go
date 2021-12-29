// Copyright 2021 Outreach Corporation. All Rights Reserved.

// Description: This file is the entrypoint for the stencil CLI
// command for stencil.
// Managed: true

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	oapp "github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/cfg"
	gcli "github.com/getoutreach/gobox/pkg/cli"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	// Place any extra imports for your startup code here
	///Block(imports)

	"github.com/getoutreach/stencil/internal/stencil"
	"github.com/getoutreach/stencil/pkg/codegen"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	///EndBlock(imports)
)

// HoneycombTracingKey gets set by the Makefile at compile-time which is pulled
// down by devconfig.sh.
var HoneycombTracingKey = "NOTSET" //nolint:gochecknoglobals // Why: We can't compile in things as a const.

///Block(honeycombDataset)
const HoneycombDataset = ""

///EndBlock(honeycombDataset)

///Block(global)
///EndBlock(global)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log := logrus.New()

	///Block(init)
	///EndBlock(init)

	app := cli.App{
		Version: oapp.Version,
		Name:    "stencil",
		///Block(app)
		Action: func(c *cli.Context) error {
			log.Infof("stencil %s", oapp.Version)

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to get the current working directory")
			}

			serviceManifest, err := configuration.NewDefaultServiceManifest()
			if err != nil {
				return errors.Wrap(err, "failed to parse service.yaml")
			}

			if !stencil.ValidateName(serviceManifest.Name) {
				return fmt.Errorf("'%s' is not an acceptable package name", serviceManifest.Name)
			}

			_, err = git.PlainOpen(cwd)
			if err != nil {
				log.Info("creating git repository")
				_, err = git.PlainInit(cwd, false)
				if err != nil {
					return errors.Wrap(err, "failed to initialize git repository")
				}
			}

			b := codegen.NewBuilder(filepath.Base(cwd), cwd, log, serviceManifest,
				c.String("github-ssh-key"), cfg.SecretData(c.String("github-access-token")))

			warnings, err := b.Run(ctx)
			for _, warning := range warnings {
				log.Warn(warning)
			}
			return errors.Wrap(err, "run codegen")
		},
		///EndBlock(app)
	}
	app.Flags = []cli.Flag{
		///Block(flags)
		&cli.BoolFlag{
			Name:  "dev",
			Usage: "Use local manifests instead of remote ones, useful for development",
		},
		&cli.StringFlag{
			Name:    "github-ssh-key",
			Usage:   "SSH Key to use to download templates with, if not set ~/.ssh/config is read and falls back to ssh-agent",
			EnvVars: []string{"GITHUB_SSH_KEY"},
		},
		&cli.StringFlag{
			Name:    "github-access-token",
			Usage:   "Github Access Token (or Personal Access Token) to use for downloading templates",
			EnvVars: []string{"GITHUB_ACCESS_TOKEN"},
		},
		///EndBlock(flags)
	}
	app.Commands = []*cli.Command{
		///Block(commands)
		///EndBlock(commands)
	}

	///Block(postApp)
	///EndBlock(postApp)

	// Insert global flags, tracing, updating and start the application.
	gcli.HookInUrfaveCLI(ctx, cancel, &app, log, HoneycombTracingKey, HoneycombDataset)
}
