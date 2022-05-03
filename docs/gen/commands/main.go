// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file generates documentation for all functions
// by reading the source code and outputing markdown.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	// We're using embed
	_ "embed"
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
