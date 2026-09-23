// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for stencil template block linting.

package templates_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bradleyjkemp/cupaloy"

	lint "github.com/getoutreach/stencil/internal/lint"
	linttemplates "github.com/getoutreach/stencil/internal/lint/templates"
)

// lintString runs LintReader over an inline template string named "t.tpl".
func lintString(in string) []lint.Finding {
	findings, _ := linttemplates.LintReader("t.tpl", strings.NewReader(in))
	return findings
}

// renderFindings formats findings one per line as aligned columns
// "SEVERITY  PATH:LINE  MESSAGE", or the literal "(no findings)" when empty,
// for stable, readable snapshotting. Copied from manifest_test.go; every
// template finding message is stencil-owned and stable, so all cases snapshot.
func renderFindings(findings []lint.Finding) string {
	if len(findings) == 0 {
		return "(no findings)\n"
	}
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

func TestLint(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "no blocks at all", in: "just some text\nno blocks here\n"},
		{
			name: "well-formed v2 block",
			in:   "## <<Stencil::Block(customMise)>>\n{{ file.Block \"customMise\" }}\n## <</Stencil::Block>>\n",
		},
		{
			// Regression: a dynamic-name v2 block WITH file.Block must be clean.
			// Previously the dynamic start was unmatched while the plain end tag
			// matched, producing a false "bare end tag" (rule 3).
			name: "dynamic-name v2 block with file.Block",
			in: "      ## <<Stencil::Block({{ $blockName }})>>\n" +
				"      {{ (file.Block $blockName) | trim }}\n" +
				"      ## <</Stencil::Block>>\n",
		},
		{
			// A dynamic-name v2 block still gets the presence check.
			name: "dynamic-name v2 block missing file.Block",
			in:   "## <<Stencil::Block({{ $b }})>>\nno file block here\n## <</Stencil::Block>>\n",
		},
		{
			// Regression: a v2 block opened AND closed with dynamic names must
			// balance (no false "never closed"). Has file.Block -> clean.
			name: "dynamic-name v2 block with dynamic close",
			in: "## <<Stencil::Block({{ $b }})>>\n{{ file.Block $b }}\n" +
				"## <</Stencil::Block({{ $b }})>>\n",
		},
		{
			// Legacy dynamic block: start AND end both unmatched -> balanced, skipped.
			name: "dynamic-name legacy block is not recognized",
			in:   "###Block({{ $b }})\n{{ file.Block $b }}\n###EndBlock({{ $b }})\n",
		},
		{
			name: "file.Block with trim marker",
			in:   "## <<Stencil::Block(x)>>\n{{- file.Block \"x\" }}\n## <</Stencil::Block>>\n",
		},
		{
			// Regression (stencil-circleci's extraContexts block): a template
			// may hoist a block's raw contents into a variable via file.Block
			// *before* the block's own start tag (e.g. to run it through
			// fromYaml and dedupe against builtin values), then render that
			// variable inside the tags instead of calling file.Block there
			// directly. That must not be flagged as missing file.Block.
			name: "file.Block hoisted before block start tag",
			in: "{{- $userContexts := (file.Block \"extraContexts\" | fromYaml) }}\n" +
				"## <<Stencil::Block(extraContexts)>>\n" +
				"{{ $userContexts | toYaml | indent 2 }}\n" +
				"## <</Stencil::Block>>\n",
		},
		{
			// Regression guard: a file.Block call for a DIFFERENT block name
			// must not satisfy this block's rule 1 just by appearing
			// somewhere else in the file.
			name: "file.Block for a different block name does not count",
			in: "{{ file.Block \"other\" }}\n" +
				"## <<Stencil::Block(foo)>>\nhardcoded, no file.Block\n## <</Stencil::Block>>\n",
		},
		{
			name: "block missing file.Block",
			in:   "## <<Stencil::Block(foo)>>\nhardcoded, no file.Block\n## <</Stencil::Block>>\n",
		},
		{
			name: "block never closed (dangling)",
			in:   "## <<Stencil::Block(foo)>>\n{{ file.Block \"foo\" }}\n",
		},
		{
			name: "bare end tag",
			in:   "## <</Stencil::Block>>\n",
		},
		{
			name: "nested blocks",
			in: "## <<Stencil::Block(a)>>\n{{ file.Block \"a\" }}\n" +
				"## <<Stencil::Block(b)>>\n{{ file.Block \"b\" }}\n## <</Stencil::Block>>\n" +
				"## <</Stencil::Block>>\n",
		},
		{
			name: "v2 close tag with args",
			in:   "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::Block(x)>>\n",
		},
		{
			name: "v2 old EndBlock without slash",
			in:   "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <<Stencil::EndBlock>>\n",
		},
		{
			name: "v2 EndBlock with slash",
			in:   "## <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n## <</Stencil::EndBlock>>\n",
		},
		{
			name: "legacy block with file.Block",
			in:   "###Block(x)\n{{ file.Block \"x\" }}\n###EndBlock(x)\n",
		},
		{
			name: "legacy block missing file.Block",
			in:   "###Block(x)\n###EndBlock(x)\n",
		},
		{
			// Rule 1 (missing file.Block, line 1) then rule 3 (bare end, line 4).
			name: "multiple findings in line order",
			in: "## <<Stencil::Block(a)>>\nno file block\n## <</Stencil::Block>>\n" +
				"## <</Stencil::Block>>\n",
		},
		{
			// Doubly-malformed: an illegal nested start then a misuse tag. The
			// misuse consumes the outstanding nested credit (innermost scope
			// first), leaving the outer block "a" open so it later dangles as
			// rule 2 ("block never closed").
			name: "misuse consumes nested credit - outer block still dangles",
			in: "## <<Stencil::Block(a)>>\n{{ file.Block \"a\" }}\n" +
				"## <<Stencil::Block(b)>>\n## <</Stencil::EndBlock>>\n",
		},
		{
			// Regression guard: outer start, illegal nested start, misuse, then two
			// real end tags. The misuse must recover the innermost scope (the
			// nested credit) first; the SECOND end tag must be flagged as a bare
			// end tag (rule 3), proving the leftover credit did not swallow it.
			name: "misuse recovers innermost scope - trailing end tag is bare",
			in: "## <<Stencil::Block(a)>>\n{{ file.Block \"a\" }}\n" +
				"## <<Stencil::Block(b)>>\n## <</Stencil::EndBlock>>\n" +
				"## <</Stencil::Block>>\n## <</Stencil::Block>>\n",
		},
		{
			// Regression guard: a block missing file.Block, closed by a misuse
			// tag, must still emit rule 1 (silently-discarded edits) plus rule 5.
			name: "rule-1 finding survives a misuse close",
			in: "## <<Stencil::Block(a)>>\nno file block\n" +
				"## <</Stencil::EndBlock>>\n",
		},
		{
			// Rule 6: a single "#" instead of "##" before the start tag. The
			// tag still balances against its (correctly prefixed) end tag and
			// has a file.Block call, so rule 6 is the only finding.
			name: "single hash instead of double hash before block start",
			in:   "# <<Stencil::Block(foo)>>\n{{ file.Block \"foo\" }}\n## <</Stencil::Block>>\n",
		},
		{
			// Rule 6 and rule 1 both fire: the single-"#" start tag is still
			// recognized as a real start (so it balances), but it also has no
			// file.Block call.
			name: "single hash block also missing file.Block",
			in:   "# <<Stencil::Block(foo)>>\nno file block here\n## <</Stencil::Block>>\n",
		},
		{
			// Rule 6 end-tag mirror: a single "#" instead of "##" before the
			// END tag. The tag still closes the (correctly prefixed) start
			// tag's block, so rule 6 is the only finding.
			name: "single hash instead of double hash before block end",
			in:   "## <<Stencil::Block(foo)>>\n{{ file.Block \"foo\" }}\n# <</Stencil::Block>>\n",
		},
		{
			// Regression: mirrors a real-world report where a block's single-
			// "#" start AND single-"#" end tag were both invisible to the
			// pre-rule-6 linter, leaving the block "never closed" and
			// cascading into false "illegal nesting" errors for every block
			// that followed. With rule 6 recognizing both tags as real (in
			// addition to reporting the bad prefix on each), the block closes
			// normally and the next, correctly-prefixed block is clean.
			name: "single hash start and end tags close cleanly, no nesting cascade",
			in: "# <<Stencil::Block(a)>>\n{{ file.Block \"a\" }}\n# <</Stencil::Block>>\n" +
				"## <<Stencil::Block(b)>>\n{{ file.Block \"b\" }}\n## <</Stencil::Block>>\n",
		},
		{
			// Regression: the exact stencil-base shape -- a normal block,
			// then a single-"#" block with no file.Block call, then another
			// normal block. Rule 6 fires on the middle block's tags and
			// rule 1 fires for its missing file.Block, but it still closes
			// normally, so neither the preceding nor the following
			// correctly-prefixed block is swept into a nesting/dangling
			// cascade.
			name: "good block, then single-hash block, then good block - no cascade",
			in: "## <<Stencil::Block(before)>>\n{{ file.Block \"before\" }}\n## <</Stencil::Block>>\n" +
				"# <<Stencil::Block(bad)>>\n# <</Stencil::Block>>\n" +
				"## <<Stencil::Block(after)>>\n{{ file.Block \"after\" }}\n## <</Stencil::Block>>\n",
		},
		{
			// Same shape, but the single-"#" block DOES have a file.Block
			// call, so rule 6 (on both its tags) is the only finding: rule 1
			// doesn't spuriously fire, and neither neighbor is touched.
			name: "good block, then single-hash block with file.Block, then good block - no cascade",
			in: "## <<Stencil::Block(before)>>\n{{ file.Block \"before\" }}\n## <</Stencil::Block>>\n" +
				"# <<Stencil::Block(bad)>>\n{{ file.Block \"bad\" }}\n# <</Stencil::Block>>\n" +
				"## <<Stencil::Block(after)>>\n{{ file.Block \"after\" }}\n## <</Stencil::Block>>\n",
		},
		{
			// Regression: leading indentation before the tag (as in a nested
			// template block) must not defeat rule 6 -- mirrors how the
			// dynamic-name tests above exercise indentation for
			// v2StartAny/v2EndAny.
			name: "single hash tags are still detected when indented",
			in:   "      # <<Stencil::Block(foo)>>\n      {{ file.Block \"foo\" }}\n      # <</Stencil::Block>>\n",
		},
		{
			// Negative case: a "#" appearing mid-line -- not at the line's
			// own comment-prefix position -- must never be mistaken for a
			// single-hash START tag, even when literal text after it happens
			// to look like one (e.g. a line documenting the syntax rather
			// than using it). Detection must anchor to the start of the
			// line's prefix, like every other tag regex in this file.
			name: "a stray # mid-line is never mistaken for a single-hash start tag",
			in:   "Example of the WRONG syntax: # <<Stencil::Block(name)>>\n",
		},
		{
			// Negative case: same guard for the END-tag mirror.
			name: "a stray # mid-line is never mistaken for a single-hash end tag",
			in:   "Docs: to close a block, use # <</Stencil::Block>>\n",
		},
		{
			// Review finding (PR #530): with a correct "##" prefix, a closing
			// tag carrying literal args reaches codegen.V2BlockPattern and
			// reports rule 5 (MsgClosingTagArgs). With a single "#", the
			// strict pattern misses (wrong prefix) and v2EndAny accepted the
			// line as a plain end tag, silently losing that misuse report.
			// This must fire it too, alongside rule 6.
			name: "single hash close tag with literal args also reports closing-tag-args misuse",
			in:   "# <<Stencil::Block(x)>>\n{{ file.Block \"x\" }}\n# <</Stencil::Block(x)>>\n",
		},
		{
			// Regression guard: a single-"#" close tag carrying a DYNAMIC
			// name (mirroring its dynamic-name open, as the correctly-
			// prefixed "dynamic-name v2 block with dynamic close" case above
			// already allows) must NOT be flagged as closing-tag-args misuse
			// -- only rule 6 fires, same as if the prefix had been correct.
			name: "single hash close tag with dynamic args is not misuse",
			in: "# <<Stencil::Block({{ $b }})>>\n{{ file.Block $b }}\n" +
				"# <</Stencil::Block({{ $b }})>>\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cupaloy.SnapshotT(t, renderFindings(lintString(test.in)))
		})
	}
}
