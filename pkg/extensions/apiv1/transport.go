// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements the transport for the implementationTransport
// to implement Implementation and vice versa

package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

// _ is a implementation check
var _ Implementation = &implementationTransportToImplementation{}

// _ is a implementation check
var _ implementationTransport = &implementationToImplementationTransport{}

// implementationTransport is the interface used for sending data over the
// wire in go-plugin. Implementation should be implemented by extensions
// instead.
type implementationTransport interface {
	GetConfig() (*Config, error)
	GetTemplateFunctions() ([]*TemplateFunction, error)
	ExecuteTemplateFunction(t *TemplateFunctionExec) ([]byte, error)
}

// newImplementationToImplementationTransport creates a new Implementation backed by a transportImplementation
func newImplementationToImplementationTransport(impl Implementation) *implementationToImplementationTransport {
	return &implementationToImplementationTransport{impl}
}

// implementationToImplementationTransport wraps a Implementation and
// implements the underlyingImplementation plugin interface automatically
// serializing the response values (as needed) into json.
type implementationToImplementationTransport struct {
	impl Implementation
}

// GetConfig returns the config for the extension
func (t *implementationToImplementationTransport) GetConfig() (*Config, error) {
	return t.impl.GetConfig()
}

// GetTemplateFunctions returns the template functions for this extension
func (t *implementationToImplementationTransport) GetTemplateFunctions() ([]*TemplateFunction, error) {
	return t.impl.GetTemplateFunctions()
}

// ExecuteTemplateFunction calls the implementation to execute a template function
// and serializes the response to json to be sent over the wire.
func (t *implementationToImplementationTransport) ExecuteTemplateFunction(exec *TemplateFunctionExec) ([]byte, error) {
	resp, err := t.impl.ExecuteTemplateFunction(exec)
	if err != nil {
		return nil, err
	}

	b := bytes.Buffer{}
	if err := json.NewEncoder(&b).Encode(resp); err != nil {
		return nil, errors.Wrap(err, "failed to encode response")
	}

	fmt.Println(b.String())

	return b.Bytes(), nil
}

// newImplementationTransportToImplementation creates a new Implementation backed by a transportImplementation
func newImplementationTransportToImplementation(impl implementationTransport) *implementationTransportToImplementation {
	return &implementationTransportToImplementation{impl}
}

// implementationTransportToImplementation turns a implementationTransport into
// a Implementation.
type implementationTransportToImplementation struct {
	impl implementationTransport
}

// GetConfig returns the config for the extension
func (t *implementationTransportToImplementation) GetConfig() (*Config, error) {
	return t.impl.GetConfig()
}

// GetTemplateFunctions returns the template functions for this extension
func (t *implementationTransportToImplementation) GetTemplateFunctions() ([]*TemplateFunction, error) {
	return t.impl.GetTemplateFunctions()
}

// ExecuteTemplateFunction calls the implementation to execute a template function
// and serializes the response to json to be sent over the wire.
func (t *implementationTransportToImplementation) ExecuteTemplateFunction(exec *TemplateFunctionExec) (interface{}, error) {
	resp, err := t.impl.ExecuteTemplateFunction(exec)
	if err != nil {
		return nil, err
	}

	var respVal interface{}
	if err := json.NewDecoder(bytes.NewReader(resp)).Decode(&respVal); err != nil {
		return nil, errors.Wrap(err, "failed to encode response")
	}

	return respVal, nil
}
