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
	funcMap map[string]map[string]reflect.Value
}

// Call returns a function based on its path, e.g. test.callFunction
func (ec *ExtensionCaller) Call(args ...reflect.Value) (reflect.Value, error) {
	if len(args) == 0 {
		return reflect.ValueOf(nil), fmt.Errorf("expected at least 1 arg")
	}

	extPath := args[0]
	if extPath.Type().Kind() != reflect.String {
		return reflect.ValueOf(nil), fmt.Errorf("expected first arg to be type string, got %s", extPath.Type().String())
	}
	keys := strings.Split(extPath.Interface().(string), ".")
	if len(keys) != 2 {
		return reflect.ValueOf(nil), fmt.Errorf("invalid extension provided to extension function")
	}

	extName := keys[0]
	extFn := keys[1]

	if _, ok := ec.funcMap[extName]; !ok {
		return reflect.ValueOf(nil), fmt.Errorf("unknown extension '%s'", extName)
	}

	if _, ok := ec.funcMap[extName][extFn]; !ok {
		return reflect.ValueOf(nil), fmt.Errorf("extension '%s' doesn't provide function '%s'", extName, extFn)
	}

	resp := ec.funcMap[extName][extFn].Call(args[1:])
	switch len(resp) {
	case 0:
		return reflect.ValueOf(nil), nil
	case 1:
		return resp[0], nil
	case 2:
		// we need to check that the err pararm returned implements an error interface
		ok := resp[1].Type().Implements(reflect.TypeOf((*error)(nil)).Elem())
		if !ok {
			return reflect.ValueOf(nil), fmt.Errorf("expected extension to return error as second param, got %v", reflect.TypeOf(resp[1]).String())
		}

		// if it's nil, just return the response
		if resp[1].IsNil() {
			return resp[0], nil
		}

		return resp[0], resp[1].Interface().(error)
	}

	return reflect.ValueOf(nil), fmt.Errorf("extension returned wrong number of args")
}
