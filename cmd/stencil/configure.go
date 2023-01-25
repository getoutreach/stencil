// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the update command

package main

import (
	"github.com/urfave/cli/v2"
)

// NewCreateCommand returns a new urfave/cli.Command for the
// create command
func NewConfigureCommand() *cli.Command {
	return &cli.Command{
		Name:        "configure",
		Description: "Commands to configure template repositories for native-extension functionality, or stencil powered repositories",
		Subcommands: []*cli.Command{
			NewConfigureModule(),
		},
	}
}
