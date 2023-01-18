// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the create templaterepository command

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/AlecAivazis/survey/v2"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// NewCreateModule returns a new urfave/cli.Command for the
// create module command
func NewCreateModule() *cli.Command {
	return &cli.Command{
		Name:        "module",
		Description: "Creates a module with the provided name in the current directory",
		ArgsUsage:   "create module <name>",
		Aliases:     []string{"templaterepository"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "native-extension",
				Usage: "Generates a native extension",
			},
		},
		Action: func(c *cli.Context) error {
			var manifestFileName = "service.yaml"

			// ensure we have a name
			if c.NArg() != 1 {
				return errors.New("must provide a name for the module")
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

			var reportingTeam string
			if err := survey.AskOne(&survey.Input{
				Message: "What is the reporting team for this module in the form of a GitHub slug (used in CODEOWNERS)?",
			}, &reportingTeam); err != nil {
				return errors.Wrap(err, "ask for reporting team")
			}

			var description string
			if err := survey.AskOne(&survey.Input{
				Message: "Enter a description for the module.",
			}, &description); err != nil {
				return errors.Wrap(err, "ask for description")
			}

			releaseOpts := map[string]interface{}{
				"enablePrereleases": true,
			}

			tm := &configuration.ServiceManifest{
				Name: path.Base(c.Args().Get(0)),
				Modules: []*configuration.TemplateRepository{{
					Name: "github.com/getoutreach/stencil-template-base",
				}},
				Arguments: map[string]interface{}{
					"reportingTeam": reportingTeam,
					"description":   description,
				},
			}

			if c.Bool("native-extension") {
				tm.Arguments["plugin"] = true
				releaseOpts["force"] = true
			}
			tm.Arguments["releaseOptions"] = releaseOpts

			if _, err := os.Stat(manifestFileName); err == nil {
				return fmt.Errorf("%s already exists", manifestFileName)
			}

			f, err := os.Create(manifestFileName)
			if err != nil {
				return err
			}
			defer f.Close()

			enc := yaml.NewEncoder(f)
			if err := enc.Encode(tm); err != nil {
				return err
			}
			if err := enc.Close(); err != nil {
				return err
			}

			//nolint:gosec // Why: intentional
			cmd := exec.CommandContext(c.Context, os.Args[0])
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return errors.Wrap(cmd.Run(), "failed to run stencil")
		},
	}
}
