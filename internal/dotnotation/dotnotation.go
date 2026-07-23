// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements a dotnotation parser for
// accessing a map[string]interface{}

// Package dotnotation implements a dotnotation (hello.world) for
// accessing fields within a map[string]interface{}
package dotnotation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// This block contains sentinel errors returned by this package.
var (
	// ErrDataNotMap is returned when the data being traversed is not a map.
	ErrDataNotMap = errors.New("data is not a map")
	// ErrKeyNotFound is returned when a key is not found in the map.
	ErrKeyNotFound = errors.New("key not found")
	// ErrKeyNotMap is returned when a nested key's value is not a map.
	ErrKeyNotMap = errors.New("key is not a map")
)

// Get looks up an entry in data by parsing the "key" into deeply nested keys, traversing it by "dots" in the key name.
func Get(data any, key string) (any, error) {
	return get(data, key)
}

// getFieldOnMap returns a field on a given map.
func getFieldOnMap(data any, key string) (any, error) {
	dataVal := reflect.ValueOf(data)
	dataTyp := dataVal.Type()
	if dataTyp.Kind() != reflect.Map {
		return nil, ErrDataNotMap
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
		if strK == key {
			return v.Interface(), nil
		}
	}

	return nil, fmt.Errorf("%w: %q", ErrKeyNotFound, key)
}

// get is a recursive function to get a field from a map[interface{}]interface{}
// this is done by splitting the key on "." and using the first part of the
// split, if there is anymore parts of the key then get() is called with
// the non processed part.
func get(data any, key string) (any, error) {
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
			return nil, fmt.Errorf("%w: %q (got %v on %q)", ErrKeyNotMap, nextKey, nextDataTyp, reflect.TypeOf(data))
		}

		// pop the first key, and call get() again
		return get(v, strings.Join(spl[1:], "."))
	}

	// otherwise return the data
	return v, nil
}
