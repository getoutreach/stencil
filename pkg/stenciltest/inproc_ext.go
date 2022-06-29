package stenciltest

import (
	"encoding/json"
	"fmt"

	"github.com/getoutreach/stencil/pkg/extensions/apiv1"
	"github.com/pkg/errors"
)

// inprocExt wraps test-provided implementation and 'simulates' the transport layer between
// the template and the target extension. This is done to ensure that ALL responses from the
// extension are undergoing same JSON-based encoding and decoding logic.
//
// Reason: assume the test uses real extension implementation in order to test in-repo extension
// and this extension returns some 'local struct' as a response to ExecTemplateFunction call. When
// this plugin is invoked by stencil, the GRPC transport layers of the stencil and the plugin
// convert the result struct to JSON and then back as a generic interface{} that consists of
// either primitives or map[string]interface{} or []interface[], losing the original type info.
//
// If we do not wrap here, type-specific 'plugin structs' are fed directly into the Go template,
// making their fields and methods available to the template. Since the JSON layer is missing,
// things may work 'nicely' in a unit test, but completely break when invoked with the transport
// from multiple reasons: 'methods' are lost, JSON marshalling can fail, JSON field names can be
// different if json tags are set on the struct, and many more.
type inprocExt struct {
	ext apiv1.Implementation
}

// GetConfig delegates the call as is
func (e inprocExt) GetConfig() (*apiv1.Config, error) {
	return e.ext.GetConfig()
}

// GetTemplateFunctions delegates the calls as is
func (e inprocExt) GetTemplateFunctions() ([]*apiv1.TemplateFunction, error) {
	return e.ext.GetTemplateFunctions()
}

// ExecuteTemplateFunction executes a provided template function on the target, JSONifies
// its response and then decodes the result JSON bytes as a plain interface{}, losing the
// source type. See docs on inprocExt for a reason.
func (e inprocExt) ExecuteTemplateFunction(t *apiv1.TemplateFunctionExec) (interface{}, error) {
	resp, err := e.ext.ExecuteTemplateFunction(t)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to encode extension response: %w", err)
	}

	var respVal interface{}
	if err := json.Unmarshal(b, &respVal); err != nil {
		return nil, errors.Wrap(err, "failed to decode etension response back into a generic interface")
	}

	return respVal, nil
}
