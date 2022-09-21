// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains logger functions specifically designed
// for terminal/cursor manipulation

package log

import (
	"fmt"

	"atomicgo.dev/cursor"
)

// ClearLine clears the current line in the terminal if we're a terminal
// otherwise it moves the cursor to the beginning of the line and prints a
// newline.
func (l *CLILogger) ClearLine() {
	if !l.isTerminal {
		fmt.Fprint(l.output, "\r\n")
		return
	}

	cursor.ClearLine()
	cursor.StartOfLine()
}
