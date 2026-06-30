// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the configuration pac

package configuration_test

import (
	"fmt"
	"testing"

	"go.yaml.in/yaml/v3"
	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/pkg/configuration"
)

func ExampleValidateName() {
	// Normal name
	success := configuration.ValidateName("test")
	fmt.Println("success:", success)

	// Invalid name
	success = configuration.ValidateName("test.1234")
	fmt.Println("success:", success)

	// Output:
	// success: true
	// success: false
}

func ExampleNewServiceManifest() {
	sm, err := configuration.NewServiceManifest("testdata/service.yaml")
	if err != nil {
		// handle the error
		fmt.Println("error:", err)
		return
	}

	fmt.Println(sm.Name)
	fmt.Println(sm.Arguments)

	// Output:
	// testing
	// map[hello:world]
}

func TestArgumentDeprecatedDecode(t *testing.T) {
	tests := []struct {
		name    string
		in      string // YAML for a single Argument
		want    string // expected Deprecated value (when no error)
		wantErr bool
	}{
		{name: "absent", in: "description: hi\n", want: ""},
		{name: "empty string", in: "deprecated: \"\"\n", want: ""},
		{name: "null", in: "deprecated:\n", want: ""},
		{name: "tilde null", in: "deprecated: ~\n", want: ""},
		{name: "message", in: "deprecated: use newArg instead\n", want: "use newArg instead"},
		{name: "quoted message", in: "deprecated: \"Use newArg instead.\"\n", want: "Use newArg instead."},
		{name: "yes is a string not a bool", in: "deprecated: yes\n", want: "yes"},
		{name: "bool true rejected", in: "deprecated: true\n", wantErr: true},
		{name: "bool false rejected", in: "deprecated: false\n", wantErr: true},
		{name: "int rejected", in: "deprecated: 42\n", wantErr: true},
		{name: "float rejected", in: "deprecated: 3.14\n", wantErr: true},
		{name: "list rejected", in: "deprecated: [a, b]\n", wantErr: true},
		{name: "map rejected", in: "deprecated: {x: 1}\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var arg configuration.Argument
			err := yaml.Unmarshal([]byte(tt.in), &arg)
			if tt.wantErr {
				assert.Assert(t, err != nil, "expected decode error, got nil (value=%q)", arg.Deprecated)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.want, string(arg.Deprecated))
			// "deprecated" is defined as a non-empty value.
			assert.Equal(t, tt.want != "", arg.Deprecated != "")
		})
	}
}
