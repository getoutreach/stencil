// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains code for the lint command
package lint

import (
	"fmt"
	"os"

	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"golang.org/x/exp/event/severity"
	"gopkg.in/yaml.v3"
)

// issue represents a linting issue
type issue struct {
	// severity is the severity of the issue
	severity severity.Level

	// message is the message of the issue
	message string

	// path is the path to the file that contains the issue
	path string

	// line is the line number that contains the issue
	line int //nolint:structcheck,unused // Why: Will be used in the future

	// column is the column number that contains the issue
	column int //nolint:structcheck,unused // Why: Will be used in the future
}

// Run is the entrypoint for the lint command
func Run() error {
	f, err := os.Open("manifest.yaml")
	if err != nil {
		return errors.Wrap(err, "failed to open manifest.yaml")
	}
	defer f.Close()

	mf := &configuration.TemplateRepositoryManifest{}
	if err := yaml.NewDecoder(f).Decode(mf); err != nil {
		return errors.Wrap(err, "failed to decode manifest.yaml")
	}

	issues := []issue{}

	for k, v := range mf.Arguments {
		if v.Description == "" {
			issues = append(issues, issue{
				severity: severity.Warning,
				message:  fmt.Sprintf("argument %q is missing a description", k),
				path:     "manifest.yaml",
			})
		}

		if v.Schema == nil {
			issues = append(issues, issue{
				severity: severity.Warning,
				message:  fmt.Sprintf("argument %q is missing a schema", k),
				path:     "manifest.yaml",
			})
		} else {
			if _, ok := v.Schema["type"]; !ok {
				issues = append(issues, issue{
					severity: severity.Warning,
					message:  fmt.Sprintf("argument %q schema is missing a top-level type, this will result in nil values being returned when not set", k),
					path:     "manifest.yaml",
				})
			}
		}
	}

	for _, iss := range issues {
		fmt.Printf("%s: %s (%s)\n", iss.severity, iss.message, iss.path)
	}

	return nil
}
