// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements a simple echo extension for testing
// purposes

package apiv1

import "fmt"

// _ ensures that this package implements the extension interface
var _ Implementation = (*EchoExtension)(nil)

// EchoExtension is a simple extension that echos back the text
// that is passed to it.
type EchoExtension struct{}

func (e *EchoExtension) GetConfig() (*Config, error) {
	return &Config{}, nil
}

func (e *EchoExtension) ExecuteTemplateFunction(t *TemplateFunctionExec) (interface{}, error) {
	if len(t.Arguments) != 1 {
		return nil, fmt.Errorf("echo requires exactly 1 argument")
	}

	return t.Arguments[0], nil
}

func (e *EchoExtension) GetTemplateFunctions() ([]*TemplateFunction, error) {
	return []*TemplateFunction{
		{
			Name:              "echo",
			NumberOfArguments: 1,
		},
	}, nil
}
