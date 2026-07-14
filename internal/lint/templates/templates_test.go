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
			// Legacy dynamic block: start AND end both unmatched -> balanced, skipped.
			name: "dynamic-name legacy block is not recognized",
			in:   "###Block({{ $b }})\n{{ file.Block $b }}\n###EndBlock({{ $b }})\n",
		},
		{
			name: "file.Block with trim marker",
			in:   "## <<Stencil::Block(x)>>\n{{- file.Block \"x\" }}\n## <</Stencil::Block>>\n",
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cupaloy.SnapshotT(t, renderFindings(lintString(test.in)))
		})
	}
}
