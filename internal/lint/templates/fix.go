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
// prefixes to its v2 (codegen.V2BlockPattern) counterpart. A prefix with no
// entry here (only reachable if BlockPattern's alternation is ever extended
// without updating this map) is left unfixed rather than guessed at, so that
// case fails safe instead of emitting a tag with no comment prefix.
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

// matchLegacyTag reports whether line is a legacy Block/EndBlock tag. It uses
// codegen.BlockPattern directly, the same regex classify() (templates.go) and
// codegen's runtime parser use, so this fixer can't drift from what the
// linter recognizes. ok is false for a dynamic-name block (e.g.
// ###Block({{ $b }})): BlockPattern's name class can't match a template
// expression, so classify doesn't recognize these as blocks either.
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

// fixLine rewrites a single legacy Block/EndBlock line (no trailing
// terminator) to v2 syntax, preserving indentation and everything after the
// tag (e.g. a trailing "-->") verbatim. message is empty when left unchanged:
// a dynamic-name block, v2 tag misuse, an unmapped legacy prefix (see
// legacyToV2Prefix), or a legacy EndBlock whose name doesn't match hasOpen/
// openName (the block FixBytes reports as currently open).
//
// The name-mismatch check exists because codegen's runtime parser rejects a
// mismatched Block/EndBlock pair at render time, and that's the only place in
// the system that ever catches it -- migrating the EndBlock to v2's nameless
// close tag would erase that signal for good. A bare EndBlock (hasOpen false)
// has nothing to compare against, so it's always safe to migrate.
//
// Otherwise a legacy tag is rewritten even inside an already-broken structure
// (nested/dangling blocks): this is a per-line syntax migration, not a
// structural fix, and structural errors are still reported after re-linting.
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
// syntax (see fixLine) and returns the result plus a log of the changes made.
// When nothing changed, fixed is raw itself, so a no-op fix never reformats an
// already-clean or already-v2 file. name identifies the template in the
// returned Applied entries' Path field. Every other line, and everything on a
// rewritten line outside the matched tag, is preserved byte for byte,
// including each line's original line-ending style.
//
// It tracks which block is open using the same classify() the linter's scan()
// uses (templates.go), so fixLine's name-mismatch check never diverges from
// the linter's view of the file. Unlike scan(), it doesn't model illegal
// nesting (a nested start just overwrites the tracked name): a file with
// illegal nesting is already rejected by codegen's runtime parser for reasons
// unrelated to any name match, so that extra fidelity isn't needed here.
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
