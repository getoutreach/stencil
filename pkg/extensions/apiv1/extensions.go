// Package apiv1 implements the extension API for stencil
package apiv1

import (
	"github.com/getoutreach/stencil/pkg/functions"
)

const (
	// Version that this extension API implements
	Version = 1

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

	// NumArguments are the number of arguments, arbitrarily, that this
	// function expects.
	NumArguments int
}

// TemplateFunctionExec executes a template function
type TemplateFunctionExec struct {
	// Name is the name of this go-template function. It will be prefixed with the
	// following format: extensions.<name>.<templateName>
	Name string

	// File is the file metadata provided by stencil
	// See: https://github.com/getoutreach/stencil/blob/df20471857be9f8cc60bc538fa54e227c449fe8c/pkg/functions/rendered_template.go#L8
	File *functions.RenderedTemplate

	// Stencil is the stencil object provided by stencil
	// See: https://github.com/getoutreach/stencil/blob/df20471857be9f8cc60bc538fa54e227c449fe8c/pkg/functions/stencil.go#L20
	Stencil *functions.Stencil

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
	ExecuteTemplateFunction(t *TemplateFunctionExec) ([]interface{}, error)
}
