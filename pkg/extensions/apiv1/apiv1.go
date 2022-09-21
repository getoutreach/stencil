// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description

// Package apiv1 implements the bridge between a extension and go-plugin
// providing most of the implementation for the extension if it's
// written in Go.
package apiv1

import "encoding/gob"

// init registers known types
func init() { //nolint:gochecknoinits // Why: see comment
	gob.Register([]interface{}{})
	gob.Register(map[string]interface{}{})
	gob.Register(map[interface{}]interface{}{})
}

// This block contains the constants for the go-plugin
// implementation.
const (
	// Version that this extension API implements
	Version = 1

	// Name is the plugin name that is served by go-plugin
	Name = "extension"

	// CookieKey is a basic UX feature for ensuring that
	// we execute a valid stencil plugin. This is exported
	// for ease of consumption by extensions.
	CookieKey = "STENCIL_PLUGIN"

	// CookieValue is the expected value for our CookieKey to
	// return.
	CookieValue = "はじめまして"
)

// TemplateFunction is a request to create a new template function.
type TemplateFunction struct {
	// Name of the template function, will be registered as:
	//  extensions.<extensionLowerName>.<name>
	Name string

	// NumberOfArguments is the number of arguments that the
	// template function takes.
	NumberOfArguments int
}

// TemplateFunctionExec executes a template function
type TemplateFunctionExec struct {
	// Name is the name of the template function to execute.
	Name string

	// Arguments are the arbitrary arguments that were passed to this function
	Arguments []interface{}
}

// Config is configuration returned by an extension
// to the extension host.
type Config struct{}

// Implementation is a plugin implementation
type Implementation interface {
	// GetConfig returns the configuration of this extension.
	GetConfig() (*Config, error)

	// GetTemplateFunctions returns all go-template functions this ext
	// implements, when a function is called, it's transparently passed over to
	// the actual extension and called there instead, its output being
	// returned.
	GetTemplateFunctions() ([]*TemplateFunction, error)

	// ExecuteTemplateFunction executes a provided template function
	// and returns its response.
	ExecuteTemplateFunction(t *TemplateFunctionExec) (interface{}, error)
}
