// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file generates documentation for all functions
// by reading the source code and outputing markdown.

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	// We're using embed
	_ "embed"

	"github.com/pkg/errors"
)

//go:embed markdown.md.tpl
var functionsTemplate string

type file struct {
	Name     string
	Contents string
}

// generateMarkdown generates the markdown files for all functions.
func generateMarkdown() ([]file, error) {
	files := make([]file, 0)

	// start with stencil --help
	commands := [][]string{{}}
	i := 0 // can't iterate over a slice while modifying it
	for {
		// we've run out of commands
		if i > (len(commands) - 1) {
			break
		}

		// get the current args, and increment the index position
		args := commands[i]
		i++

		// --skip-update <args> --help
		cmdArgs := append([]string{"--skip-update"}, append(args, "--help")...)

		fmt.Println("Generating documentation for command:", strings.Join(cmdArgs, " "))
		b, err := exec.Command("stencil", cmdArgs...).CombinedOutput()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate docs for command %q", strings.Join(args, " "))
		}

		// parse the output and add more commands to the list as we do so
		parsingCommands := false
		parsingDocumentation := false
		documentation := []string{}
		for _, line := range strings.Split(string(b), "\n") {
			if parsingCommands || parsingDocumentation {
				// Stop parsing once we get whitespace
				if strings.TrimSpace(line) == "" {
					parsingCommands = false
					parsingDocumentation = false
					continue
				}
			}

			if parsingDocumentation {
				documentation = append(documentation, strings.TrimSpace(line))
			}

			if parsingCommands {

				//   describe, d -> describe
				command := strings.TrimSpace(strings.Split(line, ",")[0])

				// skip the help command because it results in duplicates
				if command == "help" {
					continue
				}

				// args + new command
				newArgs := append(args, command)
				fmt.Println("Discovered command:", strings.Join(newArgs, " "))
				commands = append(commands, newArgs)
			}

			if strings.Contains(line, "DESCRIPTION:") {
				parsingDocumentation = true
			}

			if strings.Contains(line, "COMMANDS:") {
				parsingCommands = true
			}
		}

		// save the documentation
		t, err := template.New("markdown.md.tpl").Parse(functionsTemplate)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse template")
		}

		buf := bytes.Buffer{}
		args = append([]string{"stencil"}, args...)
		if err := t.Execute(&buf, map[string]string{
			"Command":     strings.Join(args, " "),
			"Description": strings.Join(documentation, " "),
			"Output":      string(b),
		}); err != nil {
			return nil, errors.Wrap(err, "failed to execute template")
		}

		files = append(files, file{
			Name:     strings.Join(args, "_"),
			Contents: buf.String(),
		})
	}

	return files, nil
}

// saveMarkdown writes the markdown files to disk.
func saveMarkdown(files []file) error {
	for _, f := range files {
		if err := ioutil.WriteFile(filepath.Join("content", "en", "commands", f.Name+".md"), []byte(f.Contents), 0644); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	files, err := generateMarkdown()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := saveMarkdown(files); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
