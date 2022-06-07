// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the lint command

package main

import (
	"github.com/getoutreach/stencil/internal/cmd/stencil/lint"
	"github.com/urfave/cli/v2"
)

// NewLintCommand returns a new urfave/cli.Command for the
// lint command
func NewLintCommand() *cli.Command {
	return &cli.Command{
		Name:  "lint",
		Usage: "Lint a stencil modules",
		Action: func(c *cli.Context) error {
			return lint.Run()
		},
	}
}
