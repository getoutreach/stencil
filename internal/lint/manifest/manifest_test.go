// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for stencil manifest loading and validation.

package manifest_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	lint "github.com/getoutreach/stencil/internal/lint"
	lintmanifest "github.com/getoutreach/stencil/internal/lint/manifest"
)

func TestLoadValid(t *testing.T) {
	mf, strictErr, multiDoc, readErr := lintmanifest.Load(strings.NewReader("name: testing\n"))
	assert.NilError(t, readErr)
	assert.NilError(t, strictErr)
	assert.Equal(t, false, multiDoc)
	assert.Assert(t, mf != nil)
	assert.Equal(t, "testing", mf.Name)
}

func TestLoadUnknownKeyStrictFailsButLenientPopulates(t *testing.T) {
	// 'nme' is an unknown key: strict decode fails, but lenient decode still
	// populates the rest so field checks can run.
	mf, strictErr, _, readErr := lintmanifest.Load(strings.NewReader("name: testing\nnme: oops\n"))
	assert.NilError(t, readErr)
	assert.Assert(t, strictErr != nil)
	assert.Assert(t, mf != nil)
	assert.Equal(t, "testing", mf.Name)
}

func TestLoadNestedUnknownKey(t *testing.T) {
	// An unknown key inside an argument must also trip strict decoding.
	in := "name: testing\narguments:\n  foo:\n    scema: {}\n"
	_, strictErr, _, readErr := lintmanifest.Load(strings.NewReader(in))
	assert.NilError(t, readErr)
	assert.Assert(t, strictErr != nil)
}

func TestLoadEmptyInput(t *testing.T) {
	mf, strictErr, _, readErr := lintmanifest.Load(strings.NewReader("   \n# just a comment\n"))
	assert.NilError(t, readErr)
	assert.Assert(t, strictErr != nil) // io.EOF
	assert.Assert(t, mf == nil)
}

func TestLoadMultiDocument(t *testing.T) {
	mf, strictErr, multiDoc, readErr := lintmanifest.Load(
		strings.NewReader("name: testing\n---\nname: second\n"))
	assert.NilError(t, readErr)
	assert.NilError(t, strictErr)
	assert.Assert(t, mf != nil)
	assert.Equal(t, "testing", mf.Name) // only doc 1 is read
	assert.Equal(t, true, multiDoc)
}

// validateString is a convenience that runs Load + Validate over a YAML string.
func validateString(in string) []lint.Finding {
	mf, strictErr, _, _ := lintmanifest.Load(strings.NewReader(in))
	return lintmanifest.Validate(mf, strictErr)
}

// hasFinding reports whether findings contains one with the given severity and path
// whose message contains substr.
func hasFinding(findings []lint.Finding, sev lint.Severity, path, substr string) bool {
	for _, f := range findings {
		if f.Severity == sev && f.Path == path && strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []lint.Finding // exact set (severity+path), message checked via wantMsg
		wantMsg map[string]string
		none    bool // expect zero findings
	}{
		{
			name: "valid minimal",
			in:   "name: testing\n",
			none: true,
		},
		{
			name: "valid full",
			in: "name: testing\ntype: templates,extension\nstencilVersion: \">=1.0.0\"\n" +
				"arguments:\n  greeting:\n    schema:\n      type: string\n",
			none: true,
		},
		{
			name: "unknown top-level key",
			in:   "name: testing\nnme: oops\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "manifest.yaml"}},
		},
		{
			name: "missing name",
			in:   "type: templates\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "name"}},
			wantMsg: map[string]string{
				"name": "name is required",
			},
		},
		{
			name: "import path name is valid (not a service name)",
			in:   "name: github.com/getoutreach/stencil-base\n",
			none: true,
		},
		{
			name: "unknown type",
			in:   "name: testing\ntype: templaes\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "type"}},
			wantMsg: map[string]string{
				"type": "unknown type",
			},
		},
		{
			name: "invalid schema",
			in:   "name: testing\narguments:\n  bad:\n    schema:\n      type: notarealtype\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "arguments.bad.schema"}},
		},
		{
			name: "https $ref schema reports finding (no network)",
			in:   "name: testing\narguments:\n  bad:\n    schema:\n      $ref: https://example.com/schema.json\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "arguments.bad.schema"}},
		},
		{
			name: "file $ref schema reports finding (no filesystem read)",
			in:   "name: testing\narguments:\n  bad:\n    schema:\n      $ref: file:///etc/hostname\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "arguments.bad.schema"}},
		},
		{
			name: "invalid stencilVersion",
			in:   "name: testing\nstencilVersion: not-a-constraint\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "stencilVersion"}},
		},
		{
			name: "required with default",
			in:   "name: testing\narguments:\n  x:\n    required: true\n    default: hi\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "arguments.x"}},
			wantMsg: map[string]string{
				"arguments.x": "required argument must not set a default",
			},
		},
		{
			name: "deprecated argument fields",
			in:   "name: testing\narguments:\n  x:\n    type: string\n    values: [a, b]\n",
			want: []lint.Finding{
				{Severity: lint.SeverityWarning, Path: "arguments.x.type"},
				{Severity: lint.SeverityWarning, Path: "arguments.x.values"},
			},
		},
		{
			name: "deprecated module fields",
			in:   "name: testing\nmodules:\n  - name: github.com/getoutreach/stencil-base\n    url: https://x\n    prerelease: true\n",
			want: []lint.Finding{
				{Severity: lint.SeverityWarning, Path: "modules.github.com/getoutreach/stencil-base.url"},
				{Severity: lint.SeverityWarning, Path: "modules.github.com/getoutreach/stencil-base.prerelease"},
			},
		},
		{
			name: "errors and warnings combined",
			in:   "name: testing\ntype: bogus\narguments:\n  x:\n    type: string\n",
			want: []lint.Finding{
				{Severity: lint.SeverityError, Path: "type"},
				{Severity: lint.SeverityWarning, Path: "arguments.x.type"},
			},
		},
		{
			name: "strict failure still yields field findings",
			in:   "name: testing\ntype: templaes\nnme: oops\n",
			want: []lint.Finding{
				{Severity: lint.SeverityError, Path: "manifest.yaml"},
				{Severity: lint.SeverityError, Path: "type"},
			},
		},
		{
			name: "empty input",
			in:   "  \n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "manifest.yaml"}},
			wantMsg: map[string]string{
				"manifest.yaml": "manifest is empty",
			},
		},
		{
			name: "numeric name coerced to non-empty string is valid",
			in:   "name: 123\n",
			none: true,
		},
		{
			name: "from argument skips field checks",
			in: "name: testing\narguments:\n  x:\n    from: other\n    required: true\n    default: hi\n" +
				"    type: string\n    schema:\n      type: notarealtype\n",
			none: true,
		},
		{
			name: "deprecated argument emits info finding",
			in:   "name: testing\narguments:\n  oldArg:\n    deprecated: use newArg instead\n",
			want: []lint.Finding{
				{Severity: lint.SeverityInfo, Path: "arguments.oldArg"},
			},
			wantMsg: map[string]string{
				"arguments.oldArg": "argument \"oldArg\" is deprecated: use newArg instead",
			},
		},
		{
			name: "deprecated bool form is a strict-decode error",
			in:   "name: testing\narguments:\n  oldArg:\n    deprecated: true\n",
			want: []lint.Finding{
				{Severity: lint.SeverityError, Path: "manifest.yaml"},
			},
		},
		{
			name: "from: arg with deprecated produces no info finding",
			in: "name: testing\nmodules:\n  - name: github.com/getoutreach/stencil-base\n" +
				"arguments:\n  shared:\n    from: github.com/getoutreach/stencil-base\n    deprecated: ignored\n",
			none: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := validateString(test.in)
			if test.none {
				assert.Equal(t, 0, len(got), "expected no findings, got %v", got)
				return
			}
			// every expected (severity,path) must be present
			for _, w := range test.want {
				assert.Assert(t, hasFinding(got, w.Severity, w.Path, ""),
					"missing finding %s at %q in %v", w.Severity, w.Path, got)
			}
			// no error findings beyond those expected (warnings may include extras only if expected)
			assert.Equal(t, len(test.want), len(got),
				"finding count mismatch: want %d, got %v", len(test.want), got)
			for path, substr := range test.wantMsg {
				assert.Assert(t, hasFindingMsg(got, path, substr),
					"missing message %q at %q in %v", substr, path, got)
			}
		})
	}
}

func TestValidateDeterministicOrder(t *testing.T) {
	in := "name: testing\narguments:\n  zeta:\n    type: string\n  alpha:\n    type: string\n"
	got := validateString(in)
	assert.Equal(t, 2, len(got))
	// sorted by argument key: alpha before zeta
	assert.Equal(t, "arguments.alpha.type", got[0].Path)
	assert.Equal(t, "arguments.zeta.type", got[1].Path)
}

func TestValidateMultiDocWarningIsCallerConcern(t *testing.T) {
	// Validate itself does not see multiDoc; this asserts doc-1 findings only.
	in := "name: testing\n---\nname: second\n"
	got := validateString(in)
	assert.Equal(t, 0, len(got))
}

func TestValidateFromCarveOut(t *testing.T) {
	// A from: argument that also sets schema/required+default/type yields nothing.
	in := "name: testing\narguments:\n  x:\n    from: other\n    schema:\n      type: notarealtype\n" +
		"    required: true\n    default: hi\n    type: string\n"
	got := validateString(in)
	assert.Equal(t, 0, len(got))
}

// hasFindingMsg reports whether findings has one at path whose message contains substr.
func hasFindingMsg(findings []lint.Finding, path, substr string) bool {
	for _, f := range findings {
		if f.Path == path && strings.Contains(f.Message, substr) {
			return true
		}
	}
	return false
}
