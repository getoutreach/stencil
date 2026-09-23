// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: White-box tests for package-internal invariants that an
// external (_test package) test can't reach directly.

package templates

import (
	"testing"

	"github.com/getoutreach/stencil/internal/codegen"
)

// TestLiteralBlockNameMatchesV2BlockPatternArgClass guards against
// literalBlockName (templates.go) drifting from the character class
// codegen.V2BlockPattern (internal/codegen/blocks.go) hand-copies for its own
// args group. classify()'s dynamic/literal split for a name reaching
// v2StartAny/v2EndAny is only correct as long as the two classes agree: every
// character literalBlockName accepts in a name must be one V2BlockPattern's
// own args group also accepts once that name is wrapped in a well-formed
// "##"-prefixed start tag, and vice versa.
func TestLiteralBlockNameMatchesV2BlockPatternArgClass(t *testing.T) {
	for c := byte(0x20); c < 0x7f; c++ {
		name := string(c)
		wantLiteral := literalBlockName.MatchString(name)

		line := "## <<Stencil::Block(" + name + ")>>"
		m := codegen.V2BlockPattern.FindStringSubmatch(line)
		gotLiteral := len(m) == 5

		if gotLiteral != wantLiteral {
			t.Errorf("name %q: literalBlockName accepted=%v, but codegen.V2BlockPattern's own args group accepted=%v (line %q)",
				name, wantLiteral, gotLiteral, line)
		}
	}
}
