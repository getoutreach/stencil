// Copyright 2023 Outreach Corporation. All Rights Reserved.

// Description: This file contains the code for generating documentation
// for a stencil module.

// Package generate implements a command that generates documentation for
// a stencil module.
package generate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/doc/comment"
	"io"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	// Required for the go:embed directive
	_ "embed"

	"github.com/Masterminds/sprig/v3"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/urfave/cli/v2"
)

// docsTemplate is the template used for generating documentation
//
//go:embed embed/docs.tpl
var docsTemplate string

// Run generates documentation for the current stencil module using
// the given CLI context.
func Run(c *cli.Context) error {
	mf, err := configuration.NewDefaultTemplateRepositoryManifest()
	if err != nil {
		return errors.Wrap(err, "failed to read manifest")
	}

	docsLocation := filepath.Join("docs", "arguments.md")

	fmt.Println("Generating", docsLocation)
	f, err := os.Create(docsLocation)
	if err != nil {
		return errors.Wrap(err, "failed to create arguments.md")
	}
	defer f.Close()

	if err := generateArgumentDocumentation(f, mf); err != nil {
		return errors.Wrap(err, "failed to generate argument documentation")
	}

	fmt.Println("Done!")

	return nil
}

// Argument is an argument to a stencil function
type Argument struct {
	// Name is the name of the argument
	Name string

	// Type is the type of the argument
	Type string

	// Types are the types of the argument if this argument
	// is a union type
	Types []string

	// Default is the default value of the argument
	Default interface{}

	// Options are the options for this value if the type is "enum"
	Options []interface{}

	// Required is whether or not the argument is required
	Required bool

	// Description is the description of the argument. The format of
	// this description should be Markdown.
	Description string
}

// parseJSONSchema grabs information about an argument from a JSON schema
// entry.
func parseJSONSchema(pth string, stencilArg *configuration.Argument, a *Argument) error {
	if stencilArg.Schema == nil {
		return nil
	}

	schemaBuf := new(bytes.Buffer)
	if err := json.NewEncoder(schemaBuf).Encode(stencilArg.Schema); err != nil {
		return errors.Wrap(err, "failed to encode schema into JSON")
	}

	// TODO(jaredallard): Deduplicate this code with stencil_arg.go
	jsc := jsonschema.NewCompiler()
	jsc.Draft = jsonschema.Draft2020

	schemaURL := "manifest.yaml/arguments/" + pth
	if err := jsc.AddResource(schemaURL, schemaBuf); err != nil {
		return errors.Wrapf(err, "failed to add argument '%s' json schema to compiler", pth)
	}

	schema, err := jsc.Compile(schemaURL)
	if err != nil {
		return errors.Wrapf(err, "failed to compile argument '%s' schema", pth)
	}

	if schema.Types != nil {
		if len(schema.Types) == 1 {
			a.Type = schema.Types[0]
		} else {
			a.Types = append(a.Types, schema.Types...)

			// Parse enums, if they exist
			if schema.Enum != nil {
				a.Options = append(a.Options, schema.Enum...)
			} else if schema.Items != nil {
				if subSchema, ok := schema.Items.(*jsonschema.Schema); ok {
					if subSchema.Enum != nil {
						a.Options = append(a.Options, subSchema.Enum...)
					}
				}
			}
		}
	}

	return nil
}

// parseArguments parses the arguments for a given stencil manifest and turns
// them into documentation arguments.
func parseArguments(mf *configuration.TemplateRepositoryManifest) ([]Argument, error) {
	args := make([]Argument, 0, len(mf.Arguments))
	for pth := range mf.Arguments {
		arg := mf.Arguments[pth]

		var p comment.Parser
		doc := p.Parse(arg.Description)

		a := Argument{
			Name:        pth,
			Default:     arg.Default,
			Required:    arg.Required,
			Description: string((&comment.Printer{}).Markdown(doc)),
		}

		if err := parseJSONSchema(pth, &arg, &a); err != nil {
			return nil, err
		}

		args = append(args, a)
	}

	// sort args by name
	sort.Slice(args, func(i, j int) bool {
		return args[i].Name < args[j].Name
	})

	return args, nil
}

// generateArgumentDocumentation generates documentation for the arguments
func generateArgumentDocumentation(w io.Writer, mf *configuration.TemplateRepositoryManifest) error {
	args, err := parseArguments(mf)
	if err != nil {
		return errors.Wrap(err, "failed to parse arguments")
	}

	return template.Must(template.New("docs.tpl").Funcs(sprig.TxtFuncMap()).Parse(docsTemplate)).Execute(w, map[string]interface{}{
		"Arguments": args,
	})
}
