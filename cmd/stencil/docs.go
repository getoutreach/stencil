// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the docs command

package main

import (
	"github.com/urfave/cli/v3"
)

// NewDocsCommand returns a new urfave/cli.Command for the
// docs command
func NewDocsCommand() *cli.Command {
	return &cli.Command{
		Name:        "docs",
		Description: "Commands for generating documentation",
		Commands: []*cli.Command{
			NewDocsGenerateCommand(),
		},
	}
}
