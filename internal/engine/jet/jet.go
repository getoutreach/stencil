// Copyright 2023 Outreach Corporation. All Rights Reserved.

// Description: This file implements an engine for the jet templating language

// Package jet implements a jet templating language engine
package jet

import (
	"fmt"
	"io"
	"reflect"

	"github.com/CloudyKit/jet/v6"
)

// NewInstance returns a new instance of the jet engine
func NewInstance(moduleName string) (*Instance, error) {
	c := &cache{}

	return &Instance{
		set:   jet.NewSet(jet.NewInMemLoader(), jet.WithCache(c)),
		cache: c,
	}, nil
}

// Instance is an instance of a jet engine
type Instance struct {
	// set is the underlying template set for jet
	set *jet.Set

	// cache is the underlying jet cache
	cache jet.Cache
}

// funcsToJetFunc converts a map[string]any into set.AddGlobalFunc
func (i *Instance) funcsToJetFunc(fns map[string]any) {
	for name, fn := range fns {
		i.set.AddGlobalFunc(name, func(a jet.Arguments) reflect.Value {
			rv := reflect.ValueOf(fn)
			if rv.Kind() != reflect.Func {
				panic(fmt.Errorf("global function %q is not a function", name))
			}

			// convert all arguments into a []reflect.Value
			args := make([]reflect.Value, a.NumOfArguments())
			for i := 0; i < a.NumOfArguments(); i++ {
				args[i] = a.Get(i)
			}

			rtrn := rv.Call(args)
			if len(rtrn) > 2 {
				panic(fmt.Errorf("global function %q returned more than two arguments, halp!?", name))
			}

			if len(rtrn) == 2 {
				// if the second argument is an error, panic
				if rtrn[1].Interface() != nil {
					a.Panicf("global function %q returned an error: %v", name, rtrn[1].Interface())
				}
			}

			// otherwise, return the first return value
			return rtrn[0]
		})
	}
}

// Parse parses a template and adds it to the current template instance
func (i *Instance) Parse(name string, r io.Reader, fns map[string]any) error {
	contents, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Ensure the global functions are available
	i.funcsToJetFunc(fns)

	// TODO(jaredallard): This probably doesn't handle their extend/inherit stuff
	// correctly. We probably would need to load all templates into memory first
	// to do that.
	t, err := i.set.Parse(name, string(contents))
	if err != nil {
		return err
	}
	i.cache.Put("/"+name, t)

	return err
}

// Render renders a template into the provide writer
func (i *Instance) Render(name string, out io.Writer, fns map[string]any, args any) error {
	t, err := i.set.GetTemplate(name)
	if err != nil {
		return err
	}

	// Update the global functions
	i.funcsToJetFunc(fns)

	return t.Execute(out, nil, args)
}
