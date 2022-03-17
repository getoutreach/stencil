// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements all functions provided to stencil templates.

// package codegen provides funcutions to stencil templates.
package codegen

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"
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

// Default are stock template functions that don't impact
// the generation of a file. Anything that does that should be located
// in the scope of the file renderer function instead
var Default = template.FuncMap{
	"Dereference":      dereference,
	"QuoteJoinStrings": quotejoinstrings,
}
