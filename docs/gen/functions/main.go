// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file generates documentation for all functions
// by reading the source code and outputing markdown.

package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	// We're using embed
	_ "embed"

	"github.com/pkg/errors"
	"github.com/princjef/gomarkdoc"
	"github.com/princjef/gomarkdoc/lang"
	"github.com/princjef/gomarkdoc/logger"
)

//go:embed functions.md.tpl
var functionsTemplate string

type file struct {
	Name     string
	Contents string
}

// generateMarkdown generates the markdown files for all functions.
func generateMarkdown() ([]file, error) {
	// Create a renderer to output data
	out, err := gomarkdoc.NewRenderer(
		gomarkdoc.WithTemplateOverride("func", functionsTemplate),
	)
	if err != nil {
		return nil, err
	}

	buildPkg, err := build.ImportDir("../internal/codegen", build.ImportComment)
	if err != nil {
		return nil, err
	}

	// Create a documentation package from the build representation of our
	// package.
	log := logger.New(logger.DebugLevel)
	pkg, err := lang.NewPackageFromBuild(log, buildPkg)
	if err != nil {
		return nil, err
	}

	files := make([]file, 0)
	for _, typ := range pkg.Types() {
		if !strings.HasPrefix(typ.Name(), "Tpl") {
			continue
		}

		for _, f := range typ.Methods() {
			txt, err := out.Func(f)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to generate documentation for %s", f.Name())
			}

			files = append(files, file{
				Name:     strings.ToLower(strings.TrimPrefix(typ.Name(), "Tpl")) + "." + f.Name(),
				Contents: txt,
			})
		}
	}

	return files, nil
}

// saveMarkdown writes the markdown files to disk.
func saveMarkdown(files []file) error {
	for _, f := range files {
		if err := ioutil.WriteFile(filepath.Join("content", "en", "functions", f.Name+".md"), []byte(f.Contents), 0644); err != nil {
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
