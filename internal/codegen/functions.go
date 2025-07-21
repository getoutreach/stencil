// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements all functions provided to stencil templates.

// package codegen provides funcutions to stencil templates.
package codegen

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"sigs.k8s.io/yaml"
)

// dereference dereferences a pointer returning the
// referenced data type. If the provided value is not
// a pointer, it is returned.
func dereference(i interface{}) interface{} {
	infType := reflect.TypeOf(i)

	// If not a pointer, noop
	if infType.Kind() != reflect.Ptr {
		return i
	}

	return reflect.ValueOf(i).Elem().Interface()
}

// quotejoinstrings takes a slice of strings and joins
// them with the provided seperator, sep, while quoting all
// values
func quotejoinstrings(elems []string, sep string) string {
	for i := range elems {
		elems[i] = fmt.Sprintf("%q", elems[i])
	}
	return strings.Join(elems, sep)
}

// toYAML is a clone of the helm toYaml function, which takes
// an interface{} and turns it into yaml
//
// Based on:
// https://github.com/helm/helm/blob/a499b4b179307c267bdf3ec49b880e3dbd2a5591/pkg/engine/funcs.go#L83
func toYAML(v interface{}) (string, error) {
	// If no data, return an empty string
	if v == nil {
		return "", nil
	}

	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

// fromYAML converts a YAML document into a interface{}.
//
// Based on: https://github.com/helm/helm/blob/a499b4b179307c267bdf3ec49b880e3dbd2a5591/pkg/engine/funcs.go#L98
func fromYAML(str string) (interface{}, error) {
	var m interface{}

	if err := yaml.Unmarshal([]byte(str), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// toJSON converts a interface{} into a JSON document.
func toJSON(v interface{}) (string, error) {
	// If no data, return an empty string
	if v == nil {
		return "", nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

// fromJSON converts a JSON document into a interface{}.
func fromJSON(str string) (interface{}, error) {
	var m interface{}

	if err := json.Unmarshal([]byte(str), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// toTOML converts any value into a TOML document.
func toTOML(v any) (string, error) {
	// If no data, return an empty string
	if v == nil {
		return "", nil
	}

	data, err := toml.Marshal(v)
	if err != nil {
		return "", err
	}

	return strings.TrimSuffix(string(data), "\n"), nil
}

// fromTOML converts a TOML document into a value.
func fromTOML(str string) (any, error) {
	var m any

	if err := toml.Unmarshal([]byte(str), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// Default are stock template functions that don't impact
// the generation of a file. Anything that does that should be located
// in the scope of the file renderer function instead
var Default = template.FuncMap{
	"Dereference":      dereference,
	"QuoteJoinStrings": quotejoinstrings,
	"toYaml":           toYAML,
	"fromYaml":         fromYAML,
	"toJson":           toJSON,
	"fromJson":         fromJSON,
	"toToml":           toTOML,
	"fromToml":         fromTOML,
}
