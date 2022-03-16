// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file is the entrypoint for the stencil CLI
// command for stencil.
// Managed: true

package main

import (
	"context"
	"fmt"
	"os"

	oapp "github.com/getoutreach/gobox/pkg/app"
	gcli "github.com/getoutreach/gobox/pkg/cli"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	// Place any extra imports for your startup code here
	///Block(imports)

	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/stencil"
	"github.com/getoutreach/stencil/pkg/configuration"
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

			if c.Bool("debug") {
				log.SetLevel(logrus.DebugLevel)
				log.Debug("Debug logging enabled")
			}

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to get the current working directory")
			}

			serviceManifest, err := configuration.NewDefaultServiceManifest()
			if err != nil {
				return errors.Wrap(err, "failed to parse service.yaml")
			}

			if !stencil.ValidateName(serviceManifest.Name) {
				return fmt.Errorf("%q is not an acceptable package name", serviceManifest.Name)
			}

			b := codegen.NewBuilder(cwd, log, serviceManifest)
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
			Name:    "dry-run",
			Aliases: []string{"dryrun"},
			Usage:   "Don't write files to disk",
		},
		///EndBlock(flags)
	}
	app.Commands = []*cli.Command{
		///Block(commands)
		NewDescribeCmd(),
		///EndBlock(commands)
	}

	///Block(postApp)
	///EndBlock(postApp)

	// Insert global flags, tracing, updating and start the application.
	gcli.HookInUrfaveCLI(ctx, cancel, &app, log, HoneycombTracingKey, HoneycombDataset)
}
