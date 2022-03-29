// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the create command

package main

import (
	"github.com/urfave/cli/v2"
)

// NewCreateCommand returns a new urfave/cli.Command for the
// create command
func NewCreateCommand() *cli.Command {
	return &cli.Command{
		Name:        "create",
		Description: "Commands to create template repositories, or stencil powered repositories",
		Subcommands: []*cli.Command{
			NewCreateTemplateRepositoryCommand(),
		},
	}
}
