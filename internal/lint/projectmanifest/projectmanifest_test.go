// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for project manifest (service.yaml) loading and validation.

package projectmanifest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"gotest.tools/v3/assert"

	lint "github.com/getoutreach/stencil/internal/lint"
	projectmanifest "github.com/getoutreach/stencil/internal/lint/projectmanifest"
)

// renderFindings formats findings deterministically for snapshotting.
// Mirrors internal/lint/manifest/manifest_test.go's helper.
func renderFindings(findings []lint.Finding) string {
	if len(findings) == 0 {
		return "(no findings)\n"
	}
	sevWidth, locWidth := 0, 0
	locs := make([]string, len(findings))
	for i := range findings {
		locs[i] = fmt.Sprintf("%s:%d", findings[i].Path, findings[i].Line)
		if l := len(string(findings[i].Severity)); l > sevWidth {
			sevWidth = l
		}
		if l := len(locs[i]); l > locWidth {
			locWidth = l
		}
	}
	var b strings.Builder
	for i := range findings {
		fmt.Fprintf(&b, "%-*s  %-*s  %s\n",
			sevWidth, findings[i].Severity, locWidth, locs[i], findings[i].Message)
	}
	return b.String()
}

func validateString(in string) []lint.Finding {
	res, _ := projectmanifest.Load(strings.NewReader(in))
	return projectmanifest.Validate(res)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"valid minimal", "name: my-service\n"},
		{"empty", "   \n"},
		{"missing name", "modules:\n  - name: github.com/getoutreach/stencil-base\n"},
		{"invalid name uppercase", "name: MyService\n"},
		{"invalid name leading digit", "name: 9lives\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cupaloy.SnapshotT(t, renderFindings(validateString(test.in)))
		})
	}
}

func TestLoadValid(t *testing.T) {
	res, err := projectmanifest.Load(strings.NewReader("name: my-service\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "my-service", res.Manifest.Name)
	assert.Assert(t, res.Root != nil)
	assert.Equal(t, false, res.MultiDoc)
	assert.NilError(t, res.DecodeErr)
}

func TestLoadEmptyInput(t *testing.T) {
	res, err := projectmanifest.Load(strings.NewReader("   \n# just a comment\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest == nil)
	assert.Assert(t, res.DecodeErr != nil) // io.EOF
}

func TestLoadMalformed(t *testing.T) {
	res, err := projectmanifest.Load(strings.NewReader("name: [unterminated\n"))
	assert.NilError(t, err) // read succeeded; decode failure is in DecodeErr
	assert.Assert(t, res.Manifest == nil)
	assert.Assert(t, res.DecodeErr != nil)
}

func TestLoadNonMapping(t *testing.T) {
	// A top-level scalar/sequence is not a mapping: Manifest nil, DecodeErr set.
	res, err := projectmanifest.Load(strings.NewReader("- a\n- b\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest == nil)
	assert.Assert(t, res.DecodeErr != nil)
}

func TestLoadMultiDocument(t *testing.T) {
	res, err := projectmanifest.Load(
		strings.NewReader("name: my-service\n---\nname: second\n"))
	assert.NilError(t, err)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "my-service", res.Manifest.Name) // only doc 1
	assert.Equal(t, true, res.MultiDoc)
}
