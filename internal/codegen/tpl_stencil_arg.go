// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the code for the stencil.Arg
// template function.

package codegen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/getoutreach/stencil/internal/dotnotation"
	"github.com/getoutreach/stencil/pkg/configuration"
	"github.com/pkg/errors"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Arg returns the value of an argument in the service's manifest
//
//	{{- stencil.Arg "name" }}
//
// Note: Using `stencil.Arg` with no path returns all arguments
// and is equivalent to `stencil.Args`. However, that is DEPRECATED
// along with `stencil.Args` as it doesn't provide default types, or
// check the JSON schema, or track which module calls what argument.
func (s *TplStencil) Arg(pth string) (interface{}, error) {
	if pth == "" {
		return s.Args(), nil
	}

	// This is a TODO because I don't know if template functions
	// can even get a context passed to them
	ctx := context.TODO()

	mf, err := s.t.Module.Manifest(ctx)
	if err != nil {
		// In theory this should never happen because we've
		// already parsed the manifest. But, just in case
		// we handle this here.
		return nil, err
	}

	if _, ok := mf.Arguments[pth]; !ok {
		return "", fmt.Errorf("module %q doesn't list argument %q as an argument in its manifest", s.t.Module.Name, pth)
	}
	arg := mf.Arguments[pth]

	// If there's a "from" we should handle that now before anything else,
	// so that its definition is used.
	if arg.From != "" {
		fromArg, err := s.resolveFrom(ctx, pth, &arg)
		if err != nil {
			return "", err
		}
		// Guaranteed to not be nil
		arg = *fromArg
	}

	mapInf := make(map[interface{}]interface{})
	for k, v := range s.s.m.Arguments {
		mapInf[k] = v
	}

	// if not set then we return a default value based on the denoted type
	v, err := dotnotation.Get(mapInf, pth)
	if err != nil {
		v, err = s.resolveDefault(pth, &arg)
		if err != nil {
			return "", err
		}
	}

	// validate the data
	if arg.Schema != nil {
		if err := s.validateArg(pth, &arg, v); err != nil {
			return nil, err
		}
	}

	return v, nil
}

// resolveDefault resolves the default value of an argument from the manifest
func (s *TplStencil) resolveDefault(pth string, arg *configuration.Argument) (interface{}, error) {
	if arg.Default != nil {
		return arg.Default, nil
	}

	if arg.Required {
		return nil, fmt.Errorf("module %q requires argument %q but is not set", s.t.Module.Name, pth)
	}

	// json schema convention is to define "type" as the top level key.
	typ, ok := arg.Schema["type"]
	if !ok {
		// fallback to the deprecated arg.Type
		typ = arg.Type //nolint:staticcheck // Why: Compat

		// If arg.Type isn't set then we have no type information
		// so return nothing. This is likely problematic so a linter
		// should warn on this.
		if arg.Type == "" { //nolint:staticcheck // Why: Compat
			return nil, nil
		}
	}
	typs, ok := typ.(string)
	if !ok {
		return nil, fmt.Errorf("module %q argument %q has invalid type: %v", s.t.Module.Name, pth, typ)
	}

	var v interface{}
	switch typs {
	case "map", "object":
		v = make(map[interface{}]interface{})
	case "list", "array":
		v = []interface{}{}
	case "boolean", "bool":
		v = false
	case "integer", "int", "number":
		v = 0
	case "string":
		v = ""
	default:
		return "", fmt.Errorf("module %q argument %q has invalid type %q", s.t.Module.Name, pth, typs)
	}

	return v, nil
}

// resolveFrom resoles the "from" field of an argument
func (s *TplStencil) resolveFrom(ctx context.Context, pth string, arg *configuration.Argument) (*configuration.Argument, error) {
	foundModuleInDeps := false
	ourMf, err := s.t.Module.Manifest(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure that the module imports the referenced module
	for _, m := range ourMf.Modules {
		if m.Name == arg.From {
			foundModuleInDeps = true
		}
	}
	if !foundModuleInDeps {
		return nil, fmt.Errorf(
			"module %q argument %q references an argument in module %q, but doesn't list it as a dependency",
			s.t.Module.Name, pth, arg.From,
		)
	}

	// Get the manifest for the referenced module
	var fromMf *configuration.TemplateRepositoryManifest
	for _, m := range s.s.modules {
		if m.Name == arg.From {
			mf, err := m.Manifest(ctx)
			if err != nil {
				return nil, err
			}
			fromMf = &mf

			// Found the module, break
			break
		}
	}
	if fromMf == nil {
		return nil, fmt.Errorf(
			"module %q argument %q references an argument in module %q, but wasn't imported by stencil (this is a bug)",
			s.t.Module.Name, pth, arg.From,
		)
	}

	// Ensure that the module imported exposes that argument
	fromArg, ok := fromMf.Arguments[pth]
	if !ok {
		return nil, fmt.Errorf(
			"module %q argument %q references an argument in module %q, but the module does not expose that argument",
			s.t.Module.Name, pth, arg.From,
		)
	}
	return &fromArg, nil
}

// validateArg validates an argument against the schema
func (s *TplStencil) validateArg(pth string, arg *configuration.Argument, v interface{}) error {
	schemaBuf := new(bytes.Buffer)
	if err := json.NewEncoder(schemaBuf).Encode(arg.Schema); err != nil {
		return errors.Wrap(err, "failed to encode schema into JSON")
	}

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

	if err := schema.Validate(v); err != nil {
		var validationError *jsonschema.ValidationError
		if errors.As(err, &validationError) {
			return fmt.Errorf("module %q argument %q validation failed: %#v",
				s.t.Module.Name, pth, validationError.DetailedOutput())
		}

		return errors.Wrapf(err, "module %q argument %q validation failed", s.t.Module.Name, pth)
	}

	return nil
}
