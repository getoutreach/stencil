// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the describe command

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getoutreach/stencil/pkg/stencil"
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
			l, err := stencil.LoadLockfile("")
			if err != nil {
				return errors.Wrap(err, "failed to load lockfile")
			}

			// make absolute so we can handle .. and other weird path things
			// defaults to nothing if already absolute
			filePath, err := filepath.Abs(c.Args().First())
			if err != nil {
				return errors.Wrap(err, "failed to get absolute path")
			}

			// convert absolute -> relative
			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to get current working directory")
			}
			filePath = "." + strings.TrimPrefix(filePath, cwd)

			// ensure that we don't have any weird path elements
			relativeFilePath := filepath.Clean(filePath)
			for _, f := range l.Files {
				if f.Name == relativeFilePath {
					fmt.Printf("%s was created by module https://%s (template: %s)\n", f.Name, f.Module, f.Template)
					return nil
				}
			}

			return fmt.Errorf("file %q isn't created by stencil", filePath)
		},
	}
}
