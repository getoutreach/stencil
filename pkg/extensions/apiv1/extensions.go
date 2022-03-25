// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: See package description

// Package apiv1 implements the extension API for stencil
package apiv1

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

	// ArgumentTypes are the argument types that this function
	// expects. They should be serializable via gob.
	ArgumentTypes []interface{}

	// ReturnType is the return type for this function, note that
	// the signature is always (type, error) and that error is already
	// included in the function signature.
	ReturnType interface{}
}

// TemplateFunctionExec executes a template function
type TemplateFunctionExec struct {
	// Name is the name of this go-template function. It will be prefixed with the
	// following format: extensions.<name>.<templateName>
	Name string

	// Arguments are the arbitrary arguments that were passed to this function
	Arguments []interface{}
}

// Config is configuration returned by an extension
// to the extension host.
type Config struct {
	// Name is the name of this extension, used for template
	// function naming
	Name string
}

// Implementation is the extension api that is implemented
// by all extensions.
type Implementation interface {
	// GetConfig returns the configuration of this extension.
	GetConfig() (*Config, error)

	// GetTemplateFunctions returns all go-template functions this ext
	// implements, when a function is called, it's transparently passed over to
	// the actual extension and called there instead, it's output being
	// returned.
	GetTemplateFunctions() ([]*TemplateFunction, error)

	// ExecuteTemplateFunction executes a provided template function
	// and returns it's response.
	ExecuteTemplateFunction(t *TemplateFunctionExec) (interface{}, error)
}
