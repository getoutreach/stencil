// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the describe command

package main

import (
	"fmt"

	pubstencil "github.com/getoutreach/stencil/pkg/stencil"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

// NewDescribeCmd returns a new urfave/cli.Command for the
// describe command
func NewDescribeCmd() *cli.Command {
	return &cli.Command{
		Name:        "describe",
		Description: "Print information about a known file rendered by a template",
		Action: func(c *cli.Context) error {
			l, err := pubstencil.LoadLockfile("")
			if err != nil {
				return errors.Wrap(err, "failed to load lockfile")
			}

			fileName := c.Args().First()
			for _, f := range l.Files {
				if f.Name == fileName {
					fmt.Printf("%s was created by module https://%s (template: %s)\n", f.Name, f.Module, f.Template)
					return nil
				}
			}

			return fmt.Errorf("file %q isn't created by stencil", fileName)
		},
	}
}
