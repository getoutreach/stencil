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

// ErrNoArgs is returned when Call is invoked without any arguments.
var ErrNoArgs = errors.New("expected at least 1 arg")

// ErrFirstArgNotString is returned when the first argument to Call is not a string.
var ErrFirstArgNotString = errors.New("expected first arg to be type string")

// ErrUnknownExtension is returned when the requested extension is not registered.
var ErrUnknownExtension = errors.New("unknown extension")

// ErrFunctionNotProvided is returned when an extension does not provide the requested function.
var ErrFunctionNotProvided = errors.New("extension doesn't provide function")

// ExtensionCaller calls extension functions.
type ExtensionCaller struct {
	funcMap map[string]map[string]generatedTemplateFunc
}

// Call returns a function based on its path, e.g. test.callFunction.
func (ec *ExtensionCaller) Call(args ...any) (any, error) {
	if len(args) == 0 {
		return nil, ErrNoArgs
	}

	extPath, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("%w, got %s", ErrFirstArgNotString, reflect.TypeOf(args[0]))
	}

	keys := strings.Split(extPath, ".")
	extFn := keys[len(keys)-1]                        // last element is the function name
	extName := strings.TrimSuffix(extPath, "."+extFn) // remove the function name from the path

	if _, ok := ec.funcMap[extName]; !ok {
		return nil, fmt.Errorf("%w '%s'", ErrUnknownExtension, extName)
	}

	if _, ok := ec.funcMap[extName][extFn]; !ok {
		return nil, fmt.Errorf("%w: extension '%s' function '%s'", ErrFunctionNotProvided, extName, extFn)
	}

	return ec.funcMap[extName][extFn](args[1:]...)
}
