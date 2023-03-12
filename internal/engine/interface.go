// Copyright 2023 Outreach Corporation. All Rights Reserved.

// Description: This file contains the interface(s) for engines
// to implement.

package engine

import "io"

// NewInstance is a function that returns a new instance of an engine.
// All engines must implement this.
type NewInstance func(moduleName string) (Instance, error)

// Instance is an instance of an engine. An instance uses a global backing
// engine that enables cross-template rendering an function calls, when
// applicable to the underlying engine (e.g., go-templates)
type Instance interface {
	// Parse parses a template and adds it to the current engine. Depending
	// on the underlying implementation it may be executed.
	Parse(name string, r io.Reader, fns map[string]any) error

	// Render renders the template to the given writer
	Render(name string, out io.Writer, fns map[string]any, args any) error
}
