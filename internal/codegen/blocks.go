// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file handles parsing of files

package codegen

import (
	"bufio"
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"
)

// blockPattern is the regex used for parsing block commands.
// For unit testing of this regex and explanation, see https://regex101.com/r/nFgOz0/1
var blockPattern = regexp.MustCompile(`^\s*(///|###|<!---)\s*([a-zA-Z ]+)\(([a-zA-Z ]+)\)`)

// parseBlocks reads the blocks from an existing file
func parseBlocks(filePath string) (map[string]string, error) {
	args := make(map[string]string)
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
		matches := blockPattern.FindStringSubmatch(line)
		isCommand := false

		// 1: Comment (###|///)
		// 2: Command
		// 3: Argument to the command
		if len(matches) == 4 {
			cmd := matches[2]
			isCommand = true

			switch cmd {
			case "Block":
				blockName := matches[3]
				if curBlockName != "" {
					return nil, fmt.Errorf("invalid Block when already inside of a block, at %s:%d", filePath, i)
				}
				curBlockName = blockName
			case "EndBlock":
				blockName := matches[3]
				if blockName != curBlockName {
					return nil, fmt.Errorf(
						"invalid EndBlock, found EndBlock with name %q while inside of block with name %q, at %s:%d",
						blockName, curBlockName, filePath, i,
					)
				}

				if curBlockName == "" {
					return nil, fmt.Errorf("invalid EndBlock when not inside of a block, at %s:%d", filePath, i)
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
		curVal, ok := args[curBlockName]
		if ok {
			args[curBlockName] = curVal + "\n" + line
		} else {
			args[curBlockName] = line
		}
	}

	return args, nil
}
