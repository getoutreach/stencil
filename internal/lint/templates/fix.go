// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements a mechanical, comment-preserving auto-fix that
// migrates deprecated legacy Stencil block syntax (###Block(name) /
// ###EndBlock(name), in any of the three legacy comment styles) to the v2
// <<Stencil::Block(name)>> / <</Stencil::Block>> form, one line at a time.

package templates

import (
	"bytes"
	"fmt"

	"github.com/getoutreach/stencil/internal/codegen"
)

// Applied records one fix the fixer made to a template, for logging. Path is
// the template name/path (mirroring lint.Finding.Path); Line is the 1-based
// source line that was rewritten.
type Applied struct {
	Path    string
	Line    int
	Message string
}

// legacyToV2Prefix maps each of codegen.BlockPattern's legacy comment
// prefixes to its v2 (codegen.V2BlockPattern) counterpart. V2BlockPattern
// allows at most one space between the prefix and "<<", so fixLine always
// emits exactly one, regardless of the legacy line's spacing.
//
// A prefix codegen.BlockPattern matches but this map doesn't cover (only
// possible if BlockPattern's prefix alternation is ever extended without
// updating this map) is left unfixed by matchLegacyTag's caller rather than
// guessing, so that case fails safe -- no fix applied -- instead of silently
// emitting a tag with no comment prefix at all.
var legacyToV2Prefix = map[string]string{
	"###":   "##",
	"///":   "//",
	"<!---": "<!--",
}

// legacyTag holds the parts of a line matched as a legacy Block or EndBlock
// tag with a literal name.
type legacyTag struct {
	indent, prefix, command, name, suffix string
}

// matchLegacyTag reports whether line is a legacy Block/EndBlock tag, using
// codegen.BlockPattern directly -- the same shared regex classify() (see
// templates.go) and codegen's runtime parser (blocks.go) use -- rather than a
// second, independently-typed-out copy, so this fixer can never recognize a
// tag the linter doesn't (or vice versa) if BlockPattern is ever changed. ok
// is false when line isn't a recognized Block/EndBlock command, including a
// dynamic-name block (e.g. ###Block({{ $b }})): BlockPattern's name class
// ([a-zA-Z0-9 ]+) cannot match a template expression at all, so classify
// never recognizes these as blocks either (no rule-6 warning is ever emitted
// for them, and in practice this construct exists only in test data).
func matchLegacyTag(line string) (tag legacyTag, ok bool) {
	idx := codegen.BlockPattern.FindStringSubmatchIndex(line)
	if idx == nil {
		return legacyTag{}, false
	}
	command := line[idx[4]:idx[5]]
	if command != codegen.StartStatement && command != codegen.EndStatement {
		// Every other command word is inert to codegen/classify too.
		return legacyTag{}, false
	}
	return legacyTag{
		indent:  line[idx[0]:idx[2]],
		prefix:  line[idx[2]:idx[3]],
		command: command,
		name:    line[idx[6]:idx[7]],
		suffix:  line[idx[1]:],
	}, true
}

// fixLine rewrites a single legacy Block/EndBlock declaration line (with no
// trailing line terminator) to v2 syntax. hasOpen/openName describe the block
// open immediately before this line (as tracked by FixBytes); they are
// consulted only for an EndBlock tag, to decide whether dropping its name is
// safe. message is empty when the line is left unchanged.
//
// Preserves indentation and everything after the tag (e.g. a trailing "-->"
// closing an HTML comment) verbatim; only the recognized tag is rewritten.
//
// It deliberately does not migrate:
//   - a dynamic-name legacy block: see matchLegacyTag.
//   - v2 tag misuse (rule 5, e.g. <<Stencil::EndBlock>>) or anything already
//     in v2 syntax: only the legacy-to-v2 migration is safe to automate.
//   - a legacy prefix with no entry in legacyToV2Prefix: fails safe (see
//     legacyToV2Prefix's doc comment).
//   - an EndBlock(name) whose name does not match the name of the block open
//     at this point (hasOpen/openName). codegen's runtime parser
//     (blocks.go's parseBlocks) rejects exactly this mismatch at render time
//     ("invalid EndBlock, found EndBlock with name ... while inside of block
//     with name ...") -- the only place in the system that ever catches this
//     authoring bug, since neither this linter's own scan() nor v2 syntax
//     itself (whose close tag carries no name) ever compares names. Migrating
//     a mismatched EndBlock to the nameless v2 form would permanently erase
//     that one remaining signal. A bare EndBlock (hasOpen false) has nothing
//     to compare against and is always safe to migrate: parseBlocks' "not
//     inside of a block" check depends only on there being no open block,
//     never on the dropped name.
//
// A legacy tag that is part of an otherwise-broken structure (a bare
// EndBlock, or a Block opened while another is still open) is still
// rewritten when the above safety checks pass: this is a per-line syntax
// migration keyed on pattern match and (for EndBlock) name safety, not on
// full block-pairing correctness. Structural errors are unaffected by the
// rewrite and are still reported (in v2 terms) when the fixed bytes are
// re-linted.
func fixLine(line string, hasOpen bool, openName string) (fixed, message string) {
	tag, ok := matchLegacyTag(line)
	if !ok {
		return line, ""
	}
	v2Prefix, ok := legacyToV2Prefix[tag.prefix]
	if !ok {
		return line, ""
	}

	switch tag.command {
	case codegen.StartStatement:
		return tag.indent + v2Prefix + " <<Stencil::Block(" + tag.name + ")>>" + tag.suffix,
			fmt.Sprintf("migrated deprecated block syntax to <<Stencil::Block(%s)>>", tag.name)
	case codegen.EndStatement:
		if hasOpen && tag.name != openName {
			return line, ""
		}
		return tag.indent + v2Prefix + " <</Stencil::Block>>" + tag.suffix,
			"migrated deprecated block syntax to <</Stencil::Block>>"
	default:
		// Unreachable: matchLegacyTag only returns ok for these two commands.
		return line, ""
	}
}

// splitKeepEnds splits raw into lines, each retaining its original line
// terminator (if any). It mirrors bytes.SplitAfter(raw, []byte("\n")), but
// drops the trailing empty chunk SplitAfter produces when raw ends in a
// newline, so callers see exactly one chunk per source line regardless of
// whether the final line is newline-terminated.
func splitKeepEnds(raw []byte) [][]byte {
	if len(raw) == 0 {
		return nil
	}
	chunks := bytes.SplitAfter(raw, []byte("\n"))
	if last := chunks[len(chunks)-1]; len(last) == 0 {
		chunks = chunks[:len(chunks)-1]
	}
	return chunks
}

// splitTerminator splits a line chunk (as produced by splitKeepEnds) into its
// content and terminator ("\r\n", "\n", or nil for a final line with no
// trailing newline).
func splitTerminator(chunk []byte) (content, term []byte) {
	if bytes.HasSuffix(chunk, []byte("\r\n")) {
		return chunk[:len(chunk)-2], chunk[len(chunk)-2:]
	}
	if bytes.HasSuffix(chunk, []byte("\n")) {
		return chunk[:len(chunk)-1], chunk[len(chunk)-1:]
	}
	return chunk, nil
}

// FixBytes mechanically migrates deprecated legacy block syntax in raw to v2
// syntax (see fixLine) and returns the result along with a log of the changes
// made. When nothing changed, fixed is raw itself (same underlying bytes) so a
// no-op fix never reformats a file that is already clean or already v2. name
// identifies the template in the returned Applied entries' Path field.
//
// Every other line -- and everything on a rewritten line outside the matched
// tag -- is preserved byte for byte, including each line's original
// line-ending style (LF, CRLF, or no trailing terminator on the final line).
//
// While scanning, FixBytes tracks which block (if any) is open using the same
// classify() the linter's own scan() uses (see templates.go), so this view of
// "what's open" -- consulted by fixLine to decide whether an EndBlock's name
// is safe to drop -- never diverges from the linter's. Unlike scan(), this
// tracking does not model illegal nesting (a single open name/flag pair is
// overwritten by a nested start, matching scan()'s "blocks cannot legally
// nest" simplification but not its nested-credit recovery): that additional
// fidelity is not needed here, because a file with illegal nesting is already
// rejected unconditionally by codegen's runtime parser for reasons unrelated
// to any name match, before it would ever reach a name comparison.
func FixBytes(name string, raw []byte) (fixed []byte, applied []Applied) {
	var out bytes.Buffer
	var hasOpen bool
	var openName string
	for i, chunk := range splitKeepEnds(raw) {
		content, term := splitTerminator(chunk)
		text := string(content)

		newContent, message := fixLine(text, hasOpen, openName)

		switch tok := classify(text); {
		case tok.start:
			hasOpen, openName = true, tok.name
		case tok.end, tok.misuse != "":
			hasOpen, openName = false, ""
		}

		if message == "" {
			out.Write(chunk)
			continue
		}
		applied = append(applied, Applied{Path: name, Line: i + 1, Message: message})
		out.WriteString(newContent)
		out.Write(term)
	}
	if len(applied) == 0 {
		return raw, nil
	}
	return out.Bytes(), applied
}
