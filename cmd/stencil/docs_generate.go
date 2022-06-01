// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the docs generate command

package main

import (
	"github.com/urfave/cli/v2"
)

// NewDocsGenerateCommand returns a new urfave/cli.Command for the
// docs generate command
func NewDocsGenerateCommand() *cli.Command {
	return &cli.Command{
		Name:        "generate",
		Usage:       "Generate documentation",
		Description: "Generates documentation for the current stencil module",
	}
}
