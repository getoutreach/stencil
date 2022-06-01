// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file is the entrypoint for the stencil CLI
// command for stencil.
// Managed: true

package main

import (
	"context"
	"os"
	"path/filepath"

	oapp "github.com/getoutreach/gobox/pkg/app"
	"github.com/getoutreach/gobox/pkg/box"
	gcli "github.com/getoutreach/gobox/pkg/cli"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	// Place any extra imports for your startup code here
	///Block(imports)
	"github.com/getoutreach/stencil/internal/cmd/stencil"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	///EndBlock(imports)
)

// HoneycombTracingKey gets set by the Makefile at compile-time which is pulled
// down by devconfig.sh.
var HoneycombTracingKey = "NOTSET" //nolint:gochecknoglobals // Why: We can't compile in things as a const.

// TeleforkAPIKey gets set by the Makefile at compile-time which is pulled
// down by devconfig.sh.
var TeleforkAPIKey = "NOTSET" //nolint:gochecknoglobals // Why: We can't compile in things as a const.

///Block(honeycombDataset)

// HoneycombDataset is the dataset to use when talking to Honeycomb
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
		Description: "a smart templating engine for service development",
		Action: func(c *cli.Context) error {
			log.Infof("stencil %s", oapp.Version)

			if c.Bool("debug") {
				log.SetLevel(logrus.DebugLevel)
				log.Debug("Debug logging enabled")
			}

			serviceManifest, err := configuration.NewDefaultServiceManifest()
			if err != nil {
				return errors.Wrap(err, "failed to parse service.yaml")
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return errors.Wrap(err, "failed to get user's home directory")
			}

			// If we have a box config, ensure it's up to date. In the future this may
			// become a requirement to run stencil.
			if _, err := os.Stat(filepath.Join(homeDir, box.BoxConfigPath, box.BoxConfigFile)); err == nil {
				if _, err := box.EnsureBoxWithOptions(ctx, box.WithLogger(log)); err != nil {
					return errors.Wrap(err, "failed to load box config")
				}
			}

			cmd := stencil.NewCommand(log, serviceManifest, c.Bool("dry-run"),
				c.Bool("frozen-lockfile"), c.Bool("use-prerelease"))
			return errors.Wrap(cmd.Run(ctx), "run codegen")
		},
		///EndBlock(app)
	}
	app.Flags = []cli.Flag{
		///Block(flags)
		&cli.BoolFlag{
			Name:    "dry-run",
			Aliases: []string{"dryrun"},
			Usage:   "Don't write files to disk",
		},
		&cli.BoolFlag{
			Name:  "frozen-lockfile",
			Usage: "Use versions from the lockfile instead of the latest",
		},
		&cli.BoolFlag{
			Name:  "use-prerelease",
			Usage: "Use prerelease versions of stencil modules",
		},
		///EndBlock(flags)
	}
	app.Commands = []*cli.Command{
		///Block(commands)
		NewDescribeCmd(),
		NewCreateCommand(),
		NewDocsCommand(),
		NewLintCommand(),
		///EndBlock(commands)
	}

	///Block(postApp)

	///EndBlock(postApp)

	// Insert global flags, tracing, updating and start the application.
	gcli.HookInUrfaveCLI(ctx, cancel, &app, log, HoneycombTracingKey, HoneycombDataset, TeleforkAPIKey)
}
