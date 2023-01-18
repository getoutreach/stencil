// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the create command

package main

import (
	"github.com/urfave/cli/v2"
)

// NewCreateCommand returns a new urfave/cli.Command for the
// create command
func NewUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:        "update",
		Description: "Commands to update template repositories, or stencil powered repositories",
		Subcommands: []*cli.Command{
			NewUpdateModule(),
		},
	}
}
