// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the legacy-to-v2 block syntax auto-fixer.

package templates_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"
	"gotest.tools/v3/assert"

	linttemplates "github.com/getoutreach/stencil/internal/lint/templates"
)

// renderFix formats a FixBytes result for snapshotting: the fixed bytes
// (fenced so leading/trailing whitespace and line endings are visible), then
// one line per fix in the same "path:line  message" style as renderFindings
// in templates_test.go.
func renderFix(fixed []byte, applied []linttemplates.Applied) string {
	var b strings.Builder
	b.WriteString("--- fixed ---\n")
	b.Write(fixed)
	b.WriteString("--- applied ---\n")
	if len(applied) == 0 {
		b.WriteString("(no fixes applied)\n")
		return b.String()
	}
	for _, a := range applied {
		fmt.Fprintf(&b, "%s:%d  %s\n", a.Path, a.Line, a.Message)
	}
	return b.String()
}

func TestFixBytes(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "no blocks at all is unchanged", in: "just some text\nno blocks here\n"},
		{
			name: "already-v2 block is a byte-identical no-op",
			in:   "## <<Stencil::Block(customMise)>>\n{{ file.Block \"customMise\" }}\n## <</Stencil::Block>>\n",
		},
		{
			name: "legacy hash-comment block is migrated",
			in:   "###Block(x)\n{{ file.Block \"x\" }}\n###EndBlock(x)\n",
		},
		{
			name: "legacy slash-comment block is migrated",
			in:   "///Block(x)\n{{ file.Block \"x\" }}\n///EndBlock(x)\n",
		},
		{
			// Real usage (docs/content/en/getting-started/quick-start.md): the
			// legacy HTML-comment prefix is "<!---" and the line is closed by a
			// trailing "-->", which must survive the rewrite untouched.
			name: "legacy HTML-comment block preserves the trailing comment closer",
			in:   "<!--- Block(overview) -->\n\nhello, world!\n\n<!--- EndBlock(overview) -->\n",
		},
		{
			name: "indentation before the prefix is preserved",
			in:   "    ###Block(x)\n    {{ file.Block \"x\" }}\n    ###EndBlock(x)\n",
		},
		{
			name: "extra whitespace between prefix and command is normalized to one space",
			in:   "###   Block(x)\n{{ file.Block \"x\" }}\n###   EndBlock(x)\n",
		},
		{
			name: "a block name with spaces is preserved verbatim",
			in:   "###Block(my name)\n{{ file.Block \"my name\" }}\n###EndBlock(my name)\n",
		},
		{
			// A legacy block still missing file.Block after the rewrite must
			// still be reportable by the linter (fixing syntax does not fix
			// the missing-file.Block structural error).
			name: "legacy block missing file.Block is migrated but stays incomplete",
			in:   "###Block(x)\n###EndBlock(x)\n",
		},
		{
			name: "mixed legacy and v2 blocks: only the legacy one changes",
			in: "## <<Stencil::Block(a)>>\n{{ file.Block \"a\" }}\n## <</Stencil::Block>>\n" +
				"###Block(b)\n{{ file.Block \"b\" }}\n###EndBlock(b)\n",
		},
		{
			// A dynamic-name legacy block is invisible to codegen.BlockPattern
			// (and so to the linter's rule-6 warning); FixBytes must leave it
			// untouched rather than guessing at a migration for a construct the
			// linter never flagged.
			name: "dynamic-name legacy block is left untouched",
			in:   "###Block({{ $b }})\n{{ file.Block $b }}\n###EndBlock({{ $b }})\n",
		},
		{
			// A bare legacy EndBlock (rule-3 structural error, no matching
			// start) is still migrated at the syntax level; the structural
			// error itself is unaffected and is re-reported (in v2 terms) by
			// LintReader on the fixed bytes.
			name: "bare legacy EndBlock is migrated even though unpaired",
			in:   "###EndBlock(x)\n",
		},
		{
			name: "no trailing newline on the final line is preserved",
			in:   "###Block(x)\n{{ file.Block \"x\" }}\n###EndBlock(x)",
		},
		{
			name: "CRLF line endings are preserved",
			in:   "###Block(x)\r\n{{ file.Block \"x\" }}\r\n###EndBlock(x)\r\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixed, applied := linttemplates.FixBytes("t.tpl", []byte(test.in))
			cupaloy.SnapshotT(t, renderFix(fixed, applied))

			// Idempotency: fixing already-fixed bytes must be a byte-identical
			// no-op (same underlying bytes, zero further fixes applied).
			fixed2, applied2 := linttemplates.FixBytes("t.tpl", fixed)
			assert.Equal(t, string(fixed), string(fixed2))
			assert.Equal(t, 0, len(applied2))
		})
	}
}

// TestFixBytesNoOpReturnsSameBytes pins that an input requiring no fix gets
// back the identical byte slice (not a reallocated copy), matching
// manifest.FixBytes's no-op contract so a CLI caller's writeFixedFile never
// dirties a file that needed no change.
func TestFixBytesNoOpReturnsSameBytes(t *testing.T) {
	in := []byte("## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block>>\n")
	fixed, applied := linttemplates.FixBytes("t.tpl", in)
	assert.Equal(t, 0, len(applied))
	assert.Equal(t, string(in), string(fixed))
}

// TestFixBytesMissingFileBlockStillReportedAfterFix is the CLI-scope
// requirement made explicit at the package level: migrating syntax does not
// silence the rule-1 "no file.Block" error, since that is a structural
// problem the fixer intentionally never touches.
func TestFixBytesMissingFileBlockStillReportedAfterFix(t *testing.T) {
	fixed, applied := linttemplates.FixBytes("t.tpl", []byte("###Block(x)\n###EndBlock(x)\n"))
	assert.Equal(t, 2, len(applied))

	findings, err := linttemplates.LintReader("t.tpl", strings.NewReader(string(fixed)))
	assert.NilError(t, err)
	assert.Equal(t, 1, len(findings))
	assert.Assert(t, strings.Contains(findings[0].Message, "no file.Block call"))
}
