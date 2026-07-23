// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file handles parsing of files

package codegen

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// This block contains sentinel errors returned by block parsing.
var (
	// ErrEndBlockClosingTag is returned when a legacy EndBlock command is
	// written using a closing tag instead of a closing Block tag.
	ErrEndBlockClosingTag = errors.New("invalid EndBlock closing tag")

	// ErrClosingTagArgs is returned when a closing Block tag is given
	// arguments, which is not allowed.
	ErrClosingTagArgs = errors.New("invalid closing tag arguments")

	// ErrAlreadyInsideBlock is returned when a Block command is found
	// while already inside of another block.
	ErrAlreadyInsideBlock = errors.New("invalid nested block")

	// ErrNotInsideBlock is returned when an EndBlock command is found
	// while not inside of a block.
	ErrNotInsideBlock = errors.New("invalid EndBlock")

	// ErrMismatchedEndBlockName is returned when an EndBlock command's
	// name doesn't match the name of the block it's closing.
	ErrMismatchedEndBlockName = errors.New("mismatched EndBlock name")

	// ErrDanglingBlock is returned when a file ends while still inside
	// of an unclosed block.
	ErrDanglingBlock = errors.New("dangling block")
)

// StartStatement is a constant for the start of a statement.
const StartStatement = "Block"

// EndStatement is a constant for the end of a statement.
const EndStatement = "EndBlock"

// Block-misuse messages, shared with the templates linter so runtime and lint
// wording never diverge. Callers that report a line prefix it with "line N: ".
const (
	// MsgEndBlockClosingTag is emitted for <</Stencil::EndBlock>>.
	MsgEndBlockClosingTag = "Stencil::EndBlock with a <</, should use <</Stencil::Block>> instead"
	// MsgClosingTagArgs is emitted for a closing tag carrying arguments.
	MsgClosingTagArgs = "expected no arguments to <</Stencil::Block>>"
	// MsgEndBlockOpenTag is emitted for <<Stencil::EndBlock>>.
	MsgEndBlockOpenTag = "<<Stencil::EndBlock>> should be <</Stencil::Block>>"
)

// BlockPattern is the regex used for parsing block commands.
// For unit testing of this regex and explanation, see https://regex101.com/r/nFgOz0/1
// Capture groups: 1=comment prefix, 2=command, 3=name.
var BlockPattern = regexp.MustCompile(`^\s*(///|###|<!---)\s*([a-zA-Z ]+)\(([a-zA-Z0-9 ]+)\)`)

// V2BlockPattern is the new regex for parsing blocks
// For unit testing of this regex and explanation, see https://regex101.com/r/eJZ7R2/1
// Capture groups: 1=comment prefix, 2="/" if closing, 3=command, 4="(args)" or "".
// internal/lint/templates.classify depends on these indices.
var V2BlockPattern = regexp.MustCompile(`^\s*(//|##|--|<!--)\s{0,1}<<(/?)Stencil::([a-zA-Z ]+)(\([a-zA-Z0-9 _]+\))?>>`)

// parseBlocks reads the blocks from an existing file.
func parseBlocks(filePath string) (map[string]string, error) {
	blocks := make(map[string]string)
	f, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]string), nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "failed to read blocks from file %q", filePath)
	}
	defer f.Close()

	var curBlockName string
	scanner := bufio.NewScanner(f)
	for i := 0; scanner.Scan(); i++ {
		line := scanner.Text()
		matches := BlockPattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			// 0: full match
			// 1: comment prefix
			// 2: / if end of block
			// 3: block name
			// 4: block args, if present
			v2Matches := V2BlockPattern.FindStringSubmatch(line)
			if len(v2Matches) == 5 {
				cmd := v2Matches[3]
				if v2Matches[2] == "/" {
					if cmd == EndStatement {
						return nil, fmt.Errorf("%w: line %d: %s", ErrEndBlockClosingTag, i+1, MsgEndBlockClosingTag)
					}

					// If there is a /, it's a closing tag and we should
					// translate it to a closing block command
					cmd = EndStatement
					if v2Matches[4] != "" {
						return nil, fmt.Errorf("%w: line %d: %s", ErrClosingTagArgs, i+1, MsgClosingTagArgs)
					}

					v2Matches[4] = fmt.Sprintf("(%s)", curBlockName)
				} else if cmd == EndStatement {
					// If it's not a closing tag, but the command is EndBlock,
					// we should error. This is because we don't want to
					// allow users to use the old EndBlock command
					// without a closing tag
					return nil, errors.Errorf("line %d: %s", i+1, MsgEndBlockOpenTag)
				}

				// fake the old matches format so we can reuse the same code
				matches = []string{
					v2Matches[0],
					v2Matches[1],
					cmd,
					strings.TrimPrefix(strings.TrimSuffix(v2Matches[4], ")"), "("),
				}
			}
		}
		isCommand := false

		// 1: Comment (###|///)
		// 2: Command
		// 3: Argument to the command
		if len(matches) == 4 {
			cmd := matches[2]
			isCommand = true

			switch cmd {
			case StartStatement:
				blockName := matches[3]
				if curBlockName != "" {
					return nil, fmt.Errorf("%w: at %s:%d", ErrAlreadyInsideBlock, filePath, i+1)
				}
				curBlockName = blockName
			case EndStatement:
				blockName := matches[3]

				if curBlockName == "" {
					return nil, fmt.Errorf("%w: at %s:%d", ErrNotInsideBlock, filePath, i+1)
				}

				if blockName != curBlockName {
					return nil, fmt.Errorf(
						"%w: found %q, expected %q, at %s:%d",
						ErrMismatchedEndBlockName, blockName, curBlockName, filePath, i+1,
					)
				}

				curBlockName = ""
			default:
				isCommand = false
			}
		}

		// we skip lines that had a recognized command in them, or that
		// aren't in a block
		if isCommand || curBlockName == "" {
			continue
		}

		// add the line we processed to the current block we're in
		// and account for having an existing curVal or not. If we
		// don't then we assign curVal to start with the line we
		// just found.
		curVal, ok := blocks[curBlockName]
		if ok {
			blocks[curBlockName] = curVal + "\n" + line
		} else {
			blocks[curBlockName] = line
		}
	}

	if curBlockName != "" {
		return nil, fmt.Errorf("%w: %q, at %s", ErrDanglingBlock, curBlockName, filePath)
	}

	return blocks, nil
}
