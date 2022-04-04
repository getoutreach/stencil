// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the create templaterepository command

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

// NewCreateTemplateRepositoryCommand returns a new urfave/cli.Command for the
// create templaterepository command
func NewCreateTemplateRepositoryCommand() *cli.Command {
	return &cli.Command{
		Name:        "templaterepository",
		Description: "Creates a templaterepository with the provided name in the current directory",
		ArgsUsage:   "create templaterepository <name>",
		Action: func(c *cli.Context) error {
			var manifestFileName = "manifest.yaml"

			// ensure we have a name
			if c.NArg() != 1 {
				return errors.New("must provide a name for the templaterepository")
			}

			allowedFiles := []string{".", "..", ".git"}
			files, err := os.ReadDir(".")
			if err != nil {
				return err
			}

			// ensure we don't have any files in the current directory, except for
			// the allowed files
			for _, file := range files {
				found := false
				for _, af := range allowedFiles {
					if file.Name() == af {
						found = true
						continue
					}
				}
				if !found {
					return errors.New("must be in a directory with no files")
				}
			}

			tm := &configuration.TemplateRepositoryManifest{
				Name: c.Args().Get(0),
			}

			if _, err := os.Stat(manifestFileName); err == nil {
				return fmt.Errorf("%s already exists", manifestFileName)
			}

			f, err := os.Create(manifestFileName)
			if err != nil {
				return err
			}
			defer f.Close()

			enc := yaml.NewEncoder(f)
			defer enc.Close()

			return enc.Encode(tm)
		},
	}
}
