// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the create templaterepository command

package main

import (
	"os"
	"os/exec"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// NewUpdateModule returns a new urfave/cli.Command for the
// update module command
func NewUpdateModule() *cli.Command {
	return &cli.Command{
		Name:        "module",
		Description: "updates a module with the provided name in the current directory",
		ArgsUsage:   "update module",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "native-extension",
				Usage: "Updates a module to be a native extension",
			},
		},
		Action: func(c *cli.Context) error {
			readAndMergeServiceYaml("service.yaml", c.Bool("native-extension"))
			//nolint:gosec // Why: intentional
			cmd := exec.CommandContext(c.Context, os.Args[0])
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return errors.Wrap(cmd.Run(), "failed to run stancil")
		},
	}
}

func readAndMergeServiceYaml(path string, nativeExtension bool) error {
	if path == "" {
		path = "service.yaml"
	}
	var tm = &configuration.ServiceManifest{}

	if _, err := os.Stat(path); err != nil {
		return errors.Wrap(err, "service.yaml must exist")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(b, tm)
	if err != nil {
		return err
	}

	releaseOpts := map[string]interface{}{
		"enablePrereleases": true,
	}

	if nativeExtension {
		tm.Arguments["plugin"] = true
		releaseOpts["force"] = true
	} else {
		delete(tm.Arguments, "plugin")
	}
	tm.Arguments["releaseOptions"] = releaseOpts

	out, err := yaml.Marshal(tm)
	if err != nil {
		return err
	}

	err = os.WriteFile(path, out, 0o600)
	if err != nil {
		return err
	}
	return nil
}
