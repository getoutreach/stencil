// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Implements a mechanical, comment-preserving auto-fix that
// migrates deprecated legacy Stencil block syntax (###Block(name) /
// ###EndBlock(name), in any of the three legacy comment styles) to the v2
// <<Stencil::Block(name)>> / <</Stencil::Block>> form, one line at a time.

package templates

import (
	"bytes"
	"fmt"
	"regexp"
)

// Applied records one fix the fixer made to a template, for logging. Path is
// the template name/path (mirroring lint.Finding.Path); Line is the 1-based
// source line that was rewritten.
type Applied struct {
	Path    string
	Line    int
	Message string
}

// legacyLinePattern matches a legacy Block/EndBlock start or end tag with a
// literal name: the same construct classify (see templates.go) recognizes via
// codegen.BlockPattern and flags with the rule-6 deprecation warning. Capture
// groups: 1=leading indent, 2=comment prefix, 3=command (Block or EndBlock),
// 4=name. The command alternation is restricted to these two literals
// (instead of codegen.BlockPattern's broader `[a-zA-Z ]+`) so this regex is
// scoped to exactly what classify treats as a block token; every other
// command word classify silently ignores, so there is nothing to fix there.
var legacyLinePattern = regexp.MustCompile(`^(\s*)(///|###|<!---)\s*(Block|EndBlock)\(([a-zA-Z0-9 ]+)\)`)

// legacyToV2Prefix maps each legacy comment prefix to its v2 counterpart.
// V2BlockPattern allows at most one space between the prefix and "<<", so
// fixLine always emits exactly one, regardless of the legacy line's spacing.
var legacyToV2Prefix = map[string]string{
	"###":   "##",
	"///":   "//",
	"<!---": "<!--",
}

// fixLine rewrites a single legacy Block/EndBlock declaration line (with no
// trailing line terminator) to v2 syntax. message is empty when the line does
// not match (fixed then equals line unchanged). Everything before the tag
// (leading indentation) and after it (e.g. a trailing "-->" closing an HTML
// comment) is preserved verbatim; only the recognized tag itself is rewritten.
//
// It deliberately does not attempt to migrate:
//   - a dynamic-name legacy block (e.g. ###Block({{ $b }})): codegen.BlockPattern
//     cannot match a template expression as a name, so classify never
//     recognizes these as blocks at all (no rule-6 warning is ever emitted for
//     them), and in practice this construct exists only in test data.
//   - v2 tag misuse (rule 5, e.g. <<Stencil::EndBlock>>) or anything already
//     in v2 syntax: only the legacy-to-v2 migration is safe to automate.
//
// A legacy tag that is part of an already-broken structure (a bare EndBlock,
// or a Block opened while another is still open) is still rewritten: this is
// a pure per-line syntax migration keyed only on pattern match, not on
// block-pairing state. The structural error itself is unaffected by the
// rewrite and is still reported (in v2 terms) when the fixed bytes are
// re-linted.
func fixLine(line string) (fixed, message string) {
	idx := legacyLinePattern.FindStringSubmatchIndex(line)
	if idx == nil {
		return line, ""
	}
	indent := line[idx[2]:idx[3]]
	prefix := line[idx[4]:idx[5]]
	command := line[idx[6]:idx[7]]
	name := line[idx[8]:idx[9]]
	suffix := line[idx[1]:]

	v2Prefix := legacyToV2Prefix[prefix]

	switch command {
	case "Block":
		return indent + v2Prefix + " <<Stencil::Block(" + name + ")>>" + suffix,
			fmt.Sprintf("migrated deprecated block syntax to <<Stencil::Block(%s)>>", name)
	case "EndBlock":
		return indent + v2Prefix + " <</Stencil::Block>>" + suffix,
			"migrated deprecated block syntax to <</Stencil::Block>>"
	default:
		// Unreachable: legacyLinePattern only matches these two commands.
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
// Every other line — and everything on a rewritten line outside the matched
// tag — is preserved byte for byte, including each line's original line-ending
// style (LF, CRLF, or no trailing terminator on the final line).
func FixBytes(name string, raw []byte) (fixed []byte, applied []Applied) {
	var out bytes.Buffer
	for i, chunk := range splitKeepEnds(raw) {
		content, term := splitTerminator(chunk)
		newContent, message := fixLine(string(content))
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
