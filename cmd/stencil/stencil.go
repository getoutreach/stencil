// Copyright 2024 Outreach Corporation. All Rights Reserved.

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
	"github.com/getoutreach/gobox/pkg/cfg"
	gcli "github.com/getoutreach/gobox/pkg/cli"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	// Place any extra imports for your startup code here
	// <<Stencil::Block(imports)>>
	"github.com/getoutreach/stencil/internal/cmd/stencil"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	// <</Stencil::Block>>
)

// HoneycombTracingKey gets set by the Makefile at compile-time which is pulled
// down by devconfig.sh.
var HoneycombTracingKey = "NOTSET" //nolint:gochecknoglobals // Why: We can't compile in things as a const.

// TeleforkAPIKey gets set by the Makefile at compile-time which is pulled
// down by devconfig.sh.
var TeleforkAPIKey = "NOTSET" //nolint:gochecknoglobals // Why: We can't compile in things as a const.

// <<Stencil::Block(honeycombDataset)>>

// HoneycombDataset is the dataset to use when talking to Honeycomb
const HoneycombDataset = "dev-tooling-team"

// <</Stencil::Block>>

// <<Stencil::Block(global)>>

// <</Stencil::Block>>

// main is the entrypoint for the stencil CLI.
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	log := logrus.New()

	// <<Stencil::Block(init)>>

	// <</Stencil::Block>>

	app := cli.App{
		Version: oapp.Version,
		Name:    "stencil",
		// <<Stencil::Block(app)>>
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

			cmd := stencil.NewCommand(
				log,
				serviceManifest,
				c.Bool("dry-run"),
				c.Bool("frozen-lockfile"),
				c.Bool("use-prerelease"),
				c.Bool("allow-major-version-upgrades"),
				c.Int("concurrent-resolvers"),
			)
			return errors.Wrap(cmd.Run(ctx), "run codegen")
		},
		// <</Stencil::Block>>
	}
	app.Flags = []cli.Flag{
		// <<Stencil::Block(flags)>>
		&cli.StringFlag{
			Name:        "concurrent-resolvers",
			Aliases:     []string{"c"},
			DefaultText: "5",
			Usage:       "Number of concurrent resolvers to use when resolving modules",
		},
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
		&cli.BoolFlag{
			Name:  "allow-major-version-upgrades",
			Usage: "Allow major version upgrades without confirmation",
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enables debug logging for version resolution, template render, and other useful information",
			Aliases: []string{"d"},
		},
		// <</Stencil::Block>>
	}
	app.Commands = []*cli.Command{
		// <<Stencil::Block(commands)>>
		NewDescribeCmd(),
		NewCreateCommand(),
		NewDocsCommand(),
		NewConfigureCommand(),
		// <</Stencil::Block>>
	}

	// <<Stencil::Block(postApp)>>

	// <</Stencil::Block>>

	// Insert global flags, tracing, updating and start the application.
	gcli.Run(ctx, cancel, &app, &gcli.Config{
		Logger: log,
		Telemetry: gcli.TelemetryConfig{
			Otel: gcli.TelemetryOtelConfig{
				Dataset:         HoneycombDataset,
				HoneycombAPIKey: cfg.SecretData(HoneycombTracingKey),
			},
		},
	})
}
