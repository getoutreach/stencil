// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements a dotnotation parser for
// accessing a map[string]interface{}

// Package dotnotation implements a dotnotation (hello.world) for
// accessing fields within a map[string]interface{}
package dotnotation

import (
	"fmt"
	"reflect"
	"strings"
)

// Get looks up an entry in data by parsing the "key" into deeply nested keys, traversing it by "dots" in the key name.
func Get(data map[string]interface{}, key string) (interface{}, error) {
	return get(data, key)
}

// get is a recursize function to get a field from a map[string]interface{}
// this is done by splitting the key on "." and using the first part of the
// split, if there is anymore parts of the key then get() is called with
// the non processed part
func get(data map[string]interface{}, key string) (interface{}, error) {
	spl := strings.Split(key, ".")

	partialKey := spl[0]
	subDataInf, ok := data[partialKey]
	if !ok {
		return nil, fmt.Errorf("unknown key %q", partialKey)
	}

	// check if we have more "keys"
	if len(spl) > 1 {
		subData, ok := subDataInf.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("unsupported type %q on key %q", reflect.ValueOf(subDataInf).Type(), partialKey)
		}

		return get(subData, strings.Join(spl[1:], "."))
	}

	// otherwise return the data
	return subDataInf, nil
}
