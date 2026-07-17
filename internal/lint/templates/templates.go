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
	"sort"
	"strings"

	"github.com/getoutreach/stencil/internal/codegen"
	"github.com/getoutreach/stencil/internal/lint"
)

// fileBlockCall matches a Go-template call to file.Block anywhere on a line,
// tolerating the {{- trim marker, surrounding whitespace, and any argument
// form (quoted string or variable/expression). Presence is all that matters;
// the argument is intentionally not inspected.
var fileBlockCall = regexp.MustCompile(`\{\{-?\s*.*\bfile\.Block\b`)

// fileBlockNameArg captures the literal name argument of a file.Block call
// when it's written as a simple quoted string, e.g. file.Block "extraContexts".
// A non-literal argument (a variable or expression) doesn't match and is
// simply not attributed to any name.
var fileBlockNameArg = regexp.MustCompile(`\bfile\.Block\b\s*\(?\s*"([^"]*)"`)

// v2StartAny matches a v2 Block START tag with ANY parenthesized name content,
// including a dynamic template expression like ({{ $x }}) that the strict
// codegen.V2BlockPattern rejects. It is a lint-only supplement so a dynamic-name
// block still balances against its plain <</Stencil::Block>> end tag (avoiding a
// false "bare end tag") and still gets the presence-based file.Block check. It
// deliberately does NOT match end tags or EndBlock. The name is display-only.
var v2StartAny = regexp.MustCompile(`^\s*(?://|##|--|<!--)\s?<<Stencil::Block(\([^>]*\))?>>`)

// v2EndAny matches a v2 Block END tag with ANY parenthesized content, including
// a dynamic template expression like ({{ $x }}) that the strict
// codegen.V2BlockPattern rejects. Lint-only supplement (mirrors v2StartAny) so a
// dynamic-name block's close balances against its open instead of dangling. The
// slash after << is required so it never matches an open tag. Note a bare
// <</Stencil::Block>> (no parens) is already handled by V2BlockPattern, so this
// only adds the dynamic/parenthesized-close case.
var v2EndAny = regexp.MustCompile(`^\s*(?://|##|--|<!--)\s?<</Stencil::Block(\([^>]*\))?>>`)

// blockState tracks the currently open block during a scan.
type blockState struct {
	name      string
	startLine int
	sawFile   bool
	legacy    bool
	dynamic   bool
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
	items := f.Items()
	// Findings are appended in detection order, which can place a
	// rule-1/rule-2 finding (emitted at the block's startLine) after a
	// later-line misuse error. Stable-sort by Line so output is monotonic by
	// source line, matching the manifest linter. Same-line findings keep
	// detection order.
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Line < items[j].Line
	})
	return items, nil
}

// scan walks the template line by line and appends findings. A single open
// block plus a pendingNested counter (which absorbs the end tags of illegally
// nested starts) is enough because blocks cannot legally nest.
func scan(name string, r io.Reader, f *lint.Findings) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return err
	}

	// Templates sometimes hoist a block's raw contents into a variable via
	// file.Block *before* the block's own <<Stencil::Block(name)>> start tag
	// (e.g. to run it through fromYaml and deduplicate against builtin
	// values), then render that variable inside the tags instead of calling
	// file.Block there directly. Collecting every literally-named file.Block
	// call across the whole file lets rule 1 recognize that pattern instead
	// of only crediting calls that fall between a block's own start/end tags.
	fileBlockNames := collectFileBlockNames(lines)

	var cur *blockState
	pendingNested := 0

	for line := 1; line <= len(lines); line++ {
		text := lines[line-1]

		tok := classify(text)

		switch {
		case tok.misuse != "":
			// v2 tag misuse (rule 5): report with the runtime wording, minus
			// the "line N: " prefix (the line lives in Finding.Line).
			addf(f, name, line, lint.SeverityError, "%s", tok.misuse)
			// A misuse tag is a malformed close attempt, so recover the block
			// state (mirroring the end-tag handling) to avoid a contradictory
			// rule-2 "never closed" cascade. The rule-5 error is the single
			// actionable finding. Recover the innermost pending scope first so a
			// leftover nested credit cannot swallow a later legitimate end tag,
			// and run the rule-1 file.Block check so a block closed by a misuse
			// tag still reports silently-discarded user edits.
			cur, pendingNested = closeBlock(f, name, cur, pendingNested, fileBlockNames)
		case tok.start:
			if cur != nil {
				// rule 4: illegal nesting. Keep the outer block; absorb the
				// inner block's eventual end tag.
				addf(f, name, line, lint.SeverityError,
					"block %q opened inside block %q; blocks cannot be nested. "+
						"Close %q with <</Stencil::Block>> first.", tok.name, cur.name, cur.name)
				pendingNested++
				continue
			}
			cur = &blockState{name: tok.name, startLine: line, legacy: tok.legacy, dynamic: tok.dynamic}
			if tok.legacy {
				// NOTE (DT-4828): intentionally a warning. With warnings-as-errors
				// defaulting true on the aggregate path, a manifest+templates module
				// using deprecated-but-valid legacy blocks will now exit non-zero.
				// This is the deliberate deprecation signal; do not downgrade
				// severity or gate it.
				//
				// This is always mechanically fixable: classify() only sets
				// tok.legacy for a literal (non-dynamic) name, which is exactly what
				// fix.go's FixBytes migrates, so the --fix pointer below never
				// overpromises.
				addf(f, name, line, lint.SeverityWarning,
					"block %q uses deprecated block syntax; please migrate to "+
						"<<Stencil::Block(%s)>> ... <</Stencil::Block>> "+
						"(run 'stencil lint templates --fix' to migrate it automatically).", tok.name, tok.name)
			}
		case tok.end:
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
				cur, pendingNested = closeBlock(f, name, cur, pendingNested, fileBlockNames)
			}
		}
		// A file.Block anywhere inside the open block (including its start line)
		// marks the block as satisfying rule 1. Run once after the switch so the
		// check is not duplicated across the start and default arms.
		if cur != nil && fileBlockCall.MatchString(text) {
			cur.sawFile = true
		}
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

// closeBlock recovers the innermost open scope after a real end tag or a misuse
// tag. It decrements a pending nested credit first (matching the tok.end path);
// only when no nested credit is outstanding does it close cur, emitting the
// rule-1 "no file.Block" finding first so malformed closes still report
// silently-discarded user edits. It returns the updated open block and pending
// nested credit.
// The pendingNested and cur guards below are load-bearing for the misuse caller
// (tok.misuse), which calls closeBlock without pre-checking either; the tok.end
// caller pre-checks both, so from that path the guards are defensive.
// fileBlockNames is the whole-file set of literally-named file.Block calls
// (see collectFileBlockNames), consulted so a call hoisted above the block's
// own start tag still satisfies rule 1.
func closeBlock(f *lint.Findings, name string, cur *blockState, pendingNested int,
	fileBlockNames map[string]bool) (openBlock *blockState, remainingNested int) {
	if pendingNested > 0 {
		return cur, pendingNested - 1
	}
	if cur == nil {
		return nil, pendingNested
	}
	if !cur.sawFile && fileBlockNames[cur.name] {
		cur.sawFile = true
	}
	if !cur.sawFile {
		// rule 1: missing file.Block. For a dynamic name (a template expression
		// like {{ $b }}), omit the "Add {{ file.Block %q }}" suffix since it
		// would render nonsensical nested braces.
		if cur.dynamic {
			addf(f, name, cur.startLine, lint.SeverityError,
				"block %q has no file.Block call; user edits inside this block are "+
					"silently discarded on the next render.", cur.name)
		} else {
			addf(f, name, cur.startLine, lint.SeverityError,
				"block %q has no file.Block call; user edits inside this block are "+
					"silently discarded on the next render. Add {{ file.Block %q }} "+
					"inside the block.", cur.name, cur.name)
		}
	}
	return nil, pendingNested
}

// collectFileBlockNames scans every line of the template -- regardless of
// whether a block is currently open -- for file.Block calls with a literal
// string argument, and returns the set of block names referenced.
func collectFileBlockNames(lines []string) map[string]bool {
	names := make(map[string]bool)
	for _, text := range lines {
		if m := fileBlockNameArg.FindStringSubmatch(text); m != nil {
			names[m[1]] = true
		}
	}
	return names
}

// token describes what block construct a single line is, if any.
//
//   - start:  a block start tag (v2 <<Stencil::Block(name)>> or legacy Block(name))
//   - end:    a block end tag (v2 <</Stencil::Block>> or legacy EndBlock(name))
//   - name:   the block name for start tags. For a strict-pattern match it is a
//     literal (the regexes only capture [a-zA-Z0-9 _]); for a dynamic-name v2
//     block matched via v2StartAny it holds the raw template expression (e.g.
//     "{{ $x }}"). It is display-only and never compared.
//   - legacy: whether the tag used the deprecated ###Block/### syntax
//   - misuse: a non-empty rule-5 message (with no "line N:" prefix) when the
//     line is a malformed v2 tag; empty otherwise
type token struct {
	start   bool
	end     bool
	name    string
	legacy  bool
	misuse  string
	dynamic bool
}

// classify inspects a single line and reports what block token it is, if any.
// v2 is matched before legacy, mirroring codegen.parseBlocks.
func classify(text string) token {
	// len == 5: codegen.V2BlockPattern has 4 capture groups (m[0] is the full
	// match). If codegen adds/removes a group this guard silently falls through
	// to the fallbacks below — keep in sync with codegen.V2BlockPattern.
	if m := codegen.V2BlockPattern.FindStringSubmatch(text); len(m) == 5 {
		// m[2]="/" for closing tag; m[3]=command; m[4]="(args)" or "".
		closing := m[2] == "/"
		cmd := m[3]
		args := m[4]
		switch {
		case closing && cmd == codegen.EndStatement:
			// <</Stencil::EndBlock>>
			return token{misuse: codegen.MsgEndBlockClosingTag}
		case closing:
			if args != "" {
				// <</Stencil::Block(name)>>
				return token{misuse: codegen.MsgClosingTagArgs}
			}
			return token{end: true}
		case cmd == codegen.EndStatement:
			// <<Stencil::EndBlock>>
			return token{misuse: codegen.MsgEndBlockOpenTag}
		default:
			// <<Stencil::Block(name)>>
			return token{start: true, name: trimArgs(args)}
		}
	}

	// len == 4: codegen.BlockPattern has 3 capture groups. Keep in sync with
	// codegen.BlockPattern.
	if m := codegen.BlockPattern.FindStringSubmatch(text); len(m) == 4 {
		// m[2]=command; m[3]=name.
		switch m[2] {
		case codegen.StartStatement:
			return token{start: true, name: m[3], legacy: true}
		case codegen.EndStatement:
			return token{end: true, legacy: true}
		}
	}

	if m := v2StartAny.FindStringSubmatch(text); m != nil {
		// A dynamic-name v2 Block start (strict V2BlockPattern rejected the
		// {{...}} name). Recognize it as a start with a display-only name so it
		// balances and gets the file.Block presence check; the name is never
		// compared. m[1] is "(expr)" or "" — trim to the raw expression text.
		return token{start: true, name: trimArgs(m[1]), dynamic: true}
	}
	if v2EndAny.MatchString(text) {
		// A dynamic-name v2 Block end (strict V2BlockPattern rejected the
		// {{...}} name). Recognize it as an end so a dynamic-name block's close
		// balances against its open instead of dangling. Mirrors v2StartAny.
		return token{end: true}
	}
	return token{}
}

// trimArgs turns a captured "(name)" into "name". An empty capture yields "".
func trimArgs(args string) string {
	return strings.TrimPrefix(strings.TrimSuffix(args, ")"), "(")
}
