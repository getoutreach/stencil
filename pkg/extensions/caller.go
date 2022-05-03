// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: Implements the extension caller interface for interacting
// with extensions.

package extensions

import (
	"fmt"
	"reflect"
	"strings"
)

// ExtensionCaller calls extension functions
type ExtensionCaller struct {
	funcMap map[string]map[string]generatedTemplateFunc
}

// Call returns a function based on its path, e.g. test.callFunction
func (ec *ExtensionCaller) Call(args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("expected at least 1 arg")
	}

	extPath, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("expected first arg to be type string, got %s", reflect.TypeOf(args[0]))
	}

	keys := strings.Split(extPath, ".")
	extFn := keys[len(keys)-1]                        // last element is the function name
	extName := strings.TrimSuffix(extPath, "."+extFn) // remove the function name from the path

	if _, ok := ec.funcMap[extName]; !ok {
		return nil, fmt.Errorf("unknown extension '%s'", extName)
	}

	if _, ok := ec.funcMap[extName][extFn]; !ok {
		return nil, fmt.Errorf("extension '%s' doesn't provide function '%s'", extName, extFn)
	}

	return ec.funcMap[extName][extFn](args[1:]...)
}
