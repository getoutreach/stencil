// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the extension caller interface for interacting
// with extensions.

package extensions

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// This block contains errors returned by this file.
var (
	// ErrExpectedAtLeastOneArg is returned when Call is invoked without any arguments.
	ErrExpectedAtLeastOneArg = errors.New("expected at least 1 arg")

	// ErrExpectedStringArg is returned when the first argument to Call isn't a string.
	ErrExpectedStringArg = errors.New("expected first arg to be type string")

	// ErrUnknownExtension is returned when the requested extension isn't registered.
	ErrUnknownExtension = errors.New("unknown extension")

	// ErrExtensionMissingFunction is returned when an extension doesn't provide the requested function.
	ErrExtensionMissingFunction = errors.New("extension doesn't provide function")
)

// ExtensionCaller calls extension functions.
type ExtensionCaller struct {
	funcMap map[string]map[string]generatedTemplateFunc
}

// Call returns a function based on its path, e.g. test.callFunction.
func (ec *ExtensionCaller) Call(args ...any) (any, error) {
	if len(args) == 0 {
		return nil, ErrExpectedAtLeastOneArg
	}

	extPath, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("%w, got %s", ErrExpectedStringArg, reflect.TypeOf(args[0]))
	}

	keys := strings.Split(extPath, ".")
	extFn := keys[len(keys)-1]                        // last element is the function name
	extName := strings.TrimSuffix(extPath, "."+extFn) // remove the function name from the path

	if _, ok := ec.funcMap[extName]; !ok {
		return nil, fmt.Errorf("%w '%s'", ErrUnknownExtension, extName)
	}

	if _, ok := ec.funcMap[extName][extFn]; !ok {
		return nil, fmt.Errorf("%w: extension '%s', function '%s'", ErrExtensionMissingFunction, extName, extFn)
	}

	return ec.funcMap[extName][extFn](args[1:]...)
}
