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
			if c.NArg() != 1 {
				return errors.New("expected exactly one argument, path to file")
			}

			return describeFile(c.Args().First())
		},
	}
}

// cleanPath ensures that a path is always relative to the current working directory
// with no .., . or other path elements.
func cleanPath(path string) (string, error) {
	// make absolute so we can handle .. and other weird path things
	// defaults to nothing if already absolute
	path, err := filepath.Abs(path)
	if err != nil {
		return "", errors.Wrap(err, "failed to get absolute path")
	}

	// convert absolute -> relative
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current working directory")
	}
	path = "." + strings.TrimPrefix(path, cwd)
	return filepath.Clean(path), nil
}

// describeFile prints information about a file rendered by a template
func describeFile(filePath string) error {
	l, err := stencil.LoadLockfile("")
	if err != nil {
		return errors.Wrap(err, "failed to load lockfile")
	}

	relativeFilePath, err := cleanPath(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to clean path for searching lockfile")
	}

	for _, f := range l.Files {
		if f.Name == relativeFilePath {
			fmt.Printf("%s was created by module https://%s (template: %s)\n", f.Name, f.Module, f.Template)
			return nil
		}
	}

	return fmt.Errorf("file %q isn't created by stencil", filePath)
}
