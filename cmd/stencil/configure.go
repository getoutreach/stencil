// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the update command

package main

import (
	"github.com/urfave/cli/v3"
)

// NewConfigureCommand returns a new urfave/cli.Command for the
// configure command.
func NewConfigureCommand() *cli.Command {
	return &cli.Command{
		Name:        "module",
		Description: "Commands to configure template repositories for native-extension functionality, or stencil powered repositories",
		Commands: []*cli.Command{
			NewConfigureModuleCmd(),
		},
	}
}
