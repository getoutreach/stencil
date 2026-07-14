// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements static block-correctness validation of Stencil
// templates (*.tpl) without rendering or resolving dependencies.

// Package templates implements static block-correctness validation of Stencil
// template files without rendering or resolving dependencies.
package templates

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/lint"
)

// fileBlockCall matches a Go-template call to file.Block anywhere on a line,
// tolerating the {{- trim marker, surrounding whitespace, and any argument
// form (quoted string or variable/expression). Presence is all that matters;
// the argument is intentionally not inspected.
var fileBlockCall = regexp.MustCompile(`\{\{-?\s*.*\bfile\.Block\b`)

// blockState tracks the currently open block during a scan.
type blockState struct {
	name      string
	startLine int
	sawFile   bool
	legacy    bool
}

// addf appends a finding at the given source line. Findings carry their line in
// lint.Finding.Line (mirroring the manifest linter); Path is the template name.
func addf(f *lint.Findings, name string, line int, sev lint.Severity, format string, a ...any) {
	f.Add(lint.Finding{
		Severity: sev,
		Path:     name,
		Line:     line,
		Message:  fmt.Sprintf(format, a...),
	})
}

// LintReader lints a single template stream named name (e.g. a file path or
// "<stdin>") and returns every finding. It never returns an error for lint
// problems; a non-nil error is reserved for an I/O failure reading r.
func LintReader(name string, r io.Reader) ([]lint.Finding, error) {
	var f lint.Findings
	if err := scan(name, r, &f); err != nil {
		return nil, err
	}
	return f.Items(), nil
}

// scan walks the template line by line and appends findings. A single open
// block plus a pendingNested counter (which absorbs the end tags of illegally
// nested starts) is enough because blocks cannot legally nest.
func scan(name string, r io.Reader, f *lint.Findings) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var cur *blockState
	pendingNested := 0

	for line := 1; sc.Scan(); line++ {
		text := sc.Text()

		start, end, name2, hasArgs, legacy, misuse := classify(text)

		switch {
		case misuse != "":
			// v2 tag misuse (rule 5): report with the runtime wording, minus
			// the "line N: " prefix (the line lives in Finding.Line).
			addf(f, name, line, lint.SeverityError, "%s", misuse)
		case start:
			if cur != nil {
				// rule 4: illegal nesting. Keep the outer block; absorb the
				// inner block's eventual end tag.
				addf(f, name, line, lint.SeverityError,
					"block %q opened inside block %q; blocks cannot be nested. "+
						"Close %q with <</Stencil::Block>> first.", name2, cur.name, cur.name)
				pendingNested++
				continue
			}
			cur = &blockState{name: name2, startLine: line, legacy: legacy}
			if legacy {
				addf(f, name, line, lint.SeverityWarning,
					"block %q uses deprecated block syntax; please migrate to "+
						"<<Stencil::Block(%s)>> ... <</Stencil::Block>>.", name2, name2)
			}
			// A file.Block on the start line itself still counts.
			if fileBlockCall.MatchString(text) {
				cur.sawFile = true
			}
		case end:
			_ = hasArgs // hasArgs only meaningful for misuse, handled above
			switch {
			case pendingNested > 0:
				pendingNested--
			case cur == nil:
				// rule 3: bare end tag.
				addf(f, name, line, lint.SeverityError,
					"found a block end tag with no matching start. Stencil blocks must be "+
						"balanced like XML tags: a <</Stencil::Block>> (or EndBlock) must be "+
						"preceded by its <<Stencil::Block(name)>> (or Block(name)) start.")
			default:
				if !cur.sawFile {
					// rule 1: missing file.Block.
					addf(f, name, cur.startLine, lint.SeverityError,
						"block %q has no file.Block call; user edits inside this block are "+
							"silently discarded on the next render. Add {{ file.Block %q }} "+
							"inside the block.", cur.name, cur.name)
				}
				cur = nil
			}
		default:
			if cur != nil && fileBlockCall.MatchString(text) {
				cur.sawFile = true
			}
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	if cur != nil {
		// rule 2: dangling block.
		addf(f, name, cur.startLine, lint.SeverityError,
			"block %q is never closed. Stencil blocks must be balanced like XML tags: "+
				"every <<Stencil::Block(%s)>> needs a matching <</Stencil::Block>>.",
			cur.name, cur.name)
	}
	return nil
}

// classify inspects a single line and reports what block token it is, if any.
//
//   - start:   a block start tag (v2 <<Stencil::Block(name)>> or legacy Block(name))
//   - end:     a block end tag (v2 <</Stencil::Block>> or legacy EndBlock(name))
//   - name2:   the block name (start tags; always a literal, since the regexes
//     only match [a-zA-Z0-9 _] — dynamic-name blocks are not matched at all)
//   - hasArgs: whether an end tag carried an argument (v2 misuse signal)
//   - legacy:  whether the tag used the deprecated ###Block/### syntax
//   - misuse:  a non-empty rule-5 message (with no "line N:" prefix) when the
//     line is a malformed v2 tag; empty otherwise
//
// v2 is matched before legacy, mirroring codegen.parseBlocks.
func classify(text string) (start, end bool, name2 string, hasArgs, legacy bool, misuse string) {
	if m := codegen.V2BlockPattern.FindStringSubmatch(text); len(m) == 5 {
		// m[2]="/" for closing tag; m[3]=command; m[4]="(args)" or "".
		closing := m[2] == "/"
		cmd := m[3]
		args := m[4]
		switch {
		case closing && cmd == codegen.EndStatement:
			// <</Stencil::EndBlock>>
			return false, false, "", false, false,
				"Stencil::EndBlock with a <</, should use <</Stencil::Block>> instead"
		case closing:
			if args != "" {
				// <</Stencil::Block(name)>>
				return false, false, "", true, false,
					"expected no arguments to <</Stencil::Block>>"
			}
			return false, true, "", false, false, ""
		case cmd == codegen.EndStatement:
			// <<Stencil::EndBlock>>
			return false, false, "", false, false,
				"<<Stencil::EndBlock>> should be <</Stencil::Block>>"
		default:
			// <<Stencil::Block(name)>>
			return true, false, trimArgs(args), false, false, ""
		}
	}

	if m := codegen.BlockPattern.FindStringSubmatch(text); len(m) == 4 {
		// m[2]=command; m[3]=name.
		switch m[2] {
		case "Block":
			return true, false, m[3], false, true, ""
		case codegen.EndStatement:
			return false, true, "", false, true, ""
		}
	}
	return false, false, "", false, false, ""
}

// trimArgs turns a captured "(name)" into "name". An empty capture yields "".
func trimArgs(args string) string {
	return strings.TrimPrefix(strings.TrimSuffix(args, ")"), "(")
}
