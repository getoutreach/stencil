// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for stencil manifest loading and validation.

package manifest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"gotest.tools/v3/assert"

	lint "github.com/getoutreach/stencil/internal/lint"
	lintmanifest "github.com/getoutreach/stencil/internal/lint/manifest"
)

func TestLoadValid(t *testing.T) {
	res, readErr := lintmanifest.Load(strings.NewReader("name: testing\n"))
	assert.NilError(t, readErr)
	assert.NilError(t, res.StrictErr)
	assert.Equal(t, false, res.MultiDoc)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "testing", res.Manifest.Name)
	assert.Assert(t, res.Root != nil)
}

func TestLoadUnknownKeyStrictFailsButLenientPopulates(t *testing.T) {
	// 'nme' is an unknown key: strict decode fails, but lenient decode still
	// populates the rest so field checks can run.
	res, readErr := lintmanifest.Load(strings.NewReader("name: testing\nnme: oops\n"))
	assert.NilError(t, readErr)
	assert.Assert(t, res.StrictErr != nil)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "testing", res.Manifest.Name)
}

func TestLoadNestedUnknownKey(t *testing.T) {
	// An unknown key inside an argument must also trip strict decoding.
	in := "name: testing\narguments:\n  foo:\n    scema: {}\n"
	res, readErr := lintmanifest.Load(strings.NewReader(in))
	assert.NilError(t, readErr)
	assert.Assert(t, res.StrictErr != nil)
}

func TestLoadEmptyInput(t *testing.T) {
	res, readErr := lintmanifest.Load(strings.NewReader("   \n# just a comment\n"))
	assert.NilError(t, readErr)
	assert.Assert(t, res.StrictErr != nil) // io.EOF
	assert.Assert(t, res.Manifest == nil)
}

func TestLoadMultiDocument(t *testing.T) {
	res, readErr := lintmanifest.Load(
		strings.NewReader("name: testing\n---\nname: second\n"))
	assert.NilError(t, readErr)
	assert.NilError(t, res.StrictErr)
	assert.Assert(t, res.Manifest != nil)
	assert.Equal(t, "testing", res.Manifest.Name) // only doc 1 is read
	assert.Equal(t, true, res.MultiDoc)
}

// validateString is a convenience that runs Load + Validate over a YAML string.
func validateString(in string) []lint.Finding {
	res, _ := lintmanifest.Load(strings.NewReader(in))
	return lintmanifest.Validate(res)
}

// renderFindings formats findings one per line as aligned columns
// "SEVERITY  PATH:LINE  MESSAGE", or the literal "(no findings)" when empty,
// for stable, readable snapshotting.
//
// It is used only for findings whose messages are stencil-owned and stable.
// Cases whose message is third-party text (JSON-schema compiler output) or a
// Go-internal type name (strict-decode errors) are asserted by severity+path in
// TestValidateExternalErrors instead, so a dependency bump or type rename does
// not churn a golden file.
func renderFindings(findings []lint.Finding) string {
	if len(findings) == 0 {
		return "(no findings)\n"
	}
	// Compute the severity and path:line column widths so the message column
	// aligns regardless of which severities are present.
	sevWidth := 0
	locs := make([]string, len(findings))
	locWidth := 0
	for i, f := range findings {
		if len(f.Severity) > sevWidth {
			sevWidth = len(f.Severity)
		}
		locs[i] = fmt.Sprintf("%s:%d", f.Path, f.Line)
		if len(locs[i]) > locWidth {
			locWidth = len(locs[i])
		}
	}
	var b strings.Builder
	for i, f := range findings {
		fmt.Fprintf(&b, "%-*s  %-*s  %s\n", sevWidth, f.Severity, locWidth, locs[i], f.Message)
	}
	return b.String()
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{
			name: "valid minimal",
			in:   "name: testing\n",
		},
		{
			name: "valid full",
			in: "name: testing\ntype: templates,extension\nstencilVersion: \">=1.0.0\"\n" +
				"arguments:\n  greeting:\n    schema:\n      type: string\n",
		},
		{
			name: "missing name",
			in:   "type: templates\n",
		},
		{
			name: "import path name is valid (not a service name)",
			in:   "name: github.com/getoutreach/stencil-base\n",
		},
		{
			name: "unknown type",
			in:   "name: testing\ntype: templaes\n",
		},
		{
			name: "invalid stencilVersion",
			in:   "name: testing\nstencilVersion: not-a-constraint\n",
		},
		{
			name: "required with default",
			in:   "name: testing\narguments:\n  x:\n    required: true\n    default: hi\n",
		},
		{
			name: "deprecated argument fields",
			in:   "name: testing\narguments:\n  x:\n    type: string\n    values: [a, b]\n",
		},
		{
			name: "deprecated module fields",
			in:   "name: testing\nmodules:\n  - name: github.com/getoutreach/stencil-base\n    url: https://x\n    prerelease: true\n",
		},
		{
			name: "errors and warnings combined",
			in:   "name: testing\ntype: bogus\narguments:\n  x:\n    type: string\n",
		},
		{
			name: "empty input",
			in:   "  \n",
		},
		{
			name: "numeric name coerced to non-empty string is valid",
			in:   "name: 123\n",
		},
		{
			name: "from argument skips field checks",
			in: "name: testing\narguments:\n  x:\n    from: other\n    required: true\n    default: hi\n" +
				"    type: string\n    schema:\n      type: notarealtype\n",
		},
		{
			name: "deprecated argument emits info finding",
			in:   "name: testing\narguments:\n  oldArg:\n    deprecated: use newArg instead\n",
		},
		{
			name: "deprecated bool form is a strict-decode error",
			in:   "name: testing\narguments:\n  oldArg:\n    deprecated: true\n",
		},
		{
			name: "from: arg with deprecated produces no info finding",
			in: "name: testing\nmodules:\n  - name: github.com/getoutreach/stencil-base\n" +
				"arguments:\n  shared:\n    from: github.com/getoutreach/stencil-base\n    deprecated: ignored\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cupaloy.SnapshotT(t, renderFindings(validateString(test.in)))
		})
	}
}

// TestValidateExternalErrors covers cases whose finding messages are not
// stencil-owned: JSON-schema compiler output and strict-decode errors carry
// third-party wording, an internal schema pointer, or the Go type name
// configuration.TemplateRepositoryManifest. Snapshotting those would couple the
// goldens to a dependency's exact output, so these assert only severity+path —
// the contract stencil actually owns — matching the pre-snapshot test.
func TestValidateExternalErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []lint.Finding // severity+path only; message is third-party text
	}{
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
			name: "unknown top-level key",
			in:   "name: testing\nnme: oops\n",
			want: []lint.Finding{{Severity: lint.SeverityError, Path: "manifest.yaml"}},
		},
		{
			name: "strict failure still yields field findings",
			in:   "name: testing\ntype: templaes\nnme: oops\n",
			want: []lint.Finding{
				{Severity: lint.SeverityError, Path: "manifest.yaml"},
				{Severity: lint.SeverityError, Path: "type"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := validateString(test.in)
			assert.Equal(t, len(test.want), len(got),
				"finding count mismatch; got %v", got)
			// Findings come out in a guaranteed order (checks run in a fixed
			// sequence and arguments are sorted; see TestValidateDeterministicOrder),
			// so the positional match against want is intentional, not incidental.
			for i, w := range test.want {
				assert.Equal(t, w.Severity, got[i].Severity)
				assert.Equal(t, w.Path, got[i].Path)
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

func TestValidateAnnotatesLines(t *testing.T) {
	// 1 name: testing
	// 2 arguments:
	// 3   x:
	// 4     type: string
	// 5     values: [a, b]
	const in = `name: testing
arguments:
  x:
    type: string
    values: [a, b]
`
	got := validateString(in)

	// Both deprecation warnings carry the line of their key.
	assert.Equal(t, 4, findingLine(t, got, "arguments.x.type"))
	assert.Equal(t, 5, findingLine(t, got, "arguments.x.values"))
}

func TestValidateAnnotatesSchemaErrorLine(t *testing.T) {
	// 1 name: testing
	// 2 arguments:
	// 3   bad:
	// 4     schema:
	// 5       type: notarealtype
	const in = `name: testing
arguments:
  bad:
    schema:
      type: notarealtype
`
	got := validateString(in)
	assert.Equal(t, 4, findingLine(t, got, "arguments.bad.schema"))
}

func TestValidateAnnotatesRequiredDefaultLine(t *testing.T) {
	// 1 name: testing
	// 2 arguments:
	// 3   x:
	// 4     required: true
	// 5     default: hi
	const in = `name: testing
arguments:
  x:
    required: true
    default: hi
`
	got := validateString(in)
	// The required+default finding is anchored on the argument block key (x:).
	assert.Equal(t, 3, findingLine(t, got, "arguments.x"))
}

func TestValidateWholeDocumentFindingHasNoLine(t *testing.T) {
	got := validateString("  \n") // empty manifest → check-1 finding
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "manifest.yaml", got[0].Path)
	assert.Equal(t, 0, got[0].Line) // whole-document: no resolvable line
}

// findingLine returns the Line of the first finding at path, failing the test
// if no such finding exists.
func findingLine(t *testing.T, findings []lint.Finding, path string) int {
	t.Helper()
	for _, f := range findings {
		if f.Path == path {
			return f.Line
		}
	}
	t.Fatalf("no finding at path %q in %v", path, findings)
	return 0
}
