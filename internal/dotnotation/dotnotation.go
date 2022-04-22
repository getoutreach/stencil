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
func Get(data interface{}, key string) (interface{}, error) {
	return get(data, key)
}

// getFieldOnMap returns a field on a given map
func getFieldOnMap(data interface{}, key string) (interface{}, error) {
	dataVal := reflect.ValueOf(data)
	dataTyp := dataVal.Type()
	if dataTyp.Kind() != reflect.Map {
		return nil, fmt.Errorf("data is not a map")
	}

	// iterate over the keys of the map
	// converting them to the type of the key, when we find the key
	// we return the value
	iter := dataVal.MapRange()
	for iter.Next() {
		k := iter.Key()
		v := iter.Value()

		if k.Kind() == reflect.Interface {
			// convert interface{} keys into their actual
			// values
			k = reflect.ValueOf(k.Interface())
		}

		// quick hack to convert all types to string :(
		strK := fmt.Sprintf("%v", k.Interface())
		fmt.Printf("%q ===	%q\n", strK, key)
		if strK == key {
			return v.Interface(), nil
		}
	}

	return nil, fmt.Errorf("key %q not found", key)
}

// get is a recursive function to get a field from a map[interface{}]interface{}
// this is done by splitting the key on "." and using the first part of the
// split, if there is anymore parts of the key then get() is called with
// the non processed part
func get(data interface{}, key string) (interface{}, error) {
	spl := strings.Split(key, ".")

	v, err := getFieldOnMap(data, spl[0])
	if err != nil {
		return nil, err
	}

	// check if we have more keys to iterate over
	if len(spl) > 1 {
		// pop the first key, and get the next value as the next key to
		// process
		nextKey := spl[1:][0]
		nextDataTyp := reflect.TypeOf(v)
		if nextDataTyp == nil || nextDataTyp.Kind() != reflect.Map {
			return nil, fmt.Errorf("key %q is not a map, got %v on %q", nextKey, nextDataTyp, reflect.TypeOf(data))
		}

		// pop the first key, and call get() again
		return get(v, strings.Join(spl[1:], "."))
	}

	// otherwise return the data
	return v, nil
}
