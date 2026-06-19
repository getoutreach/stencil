// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the update module command

package main

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// NewConfigureModuleCmd returns a new urfave/cli.Command for the
// update module command
func NewConfigureModuleCmd() *cli.Command {
	return &cli.Command{
		Name:        "configure",
		Description: "updates a module with the provided name in the current directory",
		ArgsUsage:   "configure module",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "remove-native-extension",
				Usage: "Removes native extension configuration for the provided module",
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			// Call readAndMergeServiceYaml to update the service yaml to add or remove the native-extension fields.
			if err := readAndMergeServiceYaml("service.yaml", c.Bool("remove-native-extension"), "foo"); err != nil {
				if err.Error() == "no action" {
					return nil
				}
				return err
			}
			//nolint:gosec // Why: intentional
			cmd := exec.CommandContext(ctx, os.Args[0])
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			return errors.Wrap(cmd.Run(), "failed to run stencil")
		},
	}
}

// readAndMergeServiceYaml takes a path and a bool and updates the service.yaml to create/remove fields
// associated with native-extensions.
func readAndMergeServiceYaml(path string, removeNativeExtension bool, input string) error {
	log := logrus.New()
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

	if err := yaml.Unmarshal(b, tm); err != nil {
		return err
	}

	releaseOpts := map[string]interface{}{
		"enablePrereleases": true,
	}

	if !removeNativeExtension {
		// Note that for any additions to the configure command this line may need to be updated/removed.
		// This section avoids removing any comments in the service.yaml file by overwriting it if there's no action to be taken.
		if tm.Arguments["plugin"] == true && releaseOpts["force"] == true {
			log.Info("The module is already a native extension, no action taken.")
			return errors.New("no action")
		}
		tm.Arguments["plugin"] = true
		releaseOpts["force"] = true
	} else {
		// Note that for any additions to the configure command this line may need to be updated/removed.
		// This section avoids removing any comments in the service.yaml file by overwriting it if there's no action to be taken.
		_, ok := tm.Arguments["plugin"]
		if !ok {
			log.Info("The module is already not a native extension, no action taken.")
			return errors.New("no action")
		}
		delete(tm.Arguments, "plugin")
	}
	tm.Arguments["releaseOptions"] = releaseOpts

	if input != "" {
		reader := bufio.NewReader(os.Stdin)
		log.Info("This will remove any comments in your service.yaml, are you sure you want to proceed? y/N")
		text, err := reader.ReadString('\n')
		if err != nil {
			return errors.Wrap(err, "failed to get user input for overwriting service.yaml")
		}
		if strings.TrimSpace(text) != "y" {
			log.Info("No action taken")
			return errors.New("no action")
		}
	}

	out, err := yaml.Marshal(tm)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, out, 0o600); err != nil {
		return err
	}
	return nil
}
