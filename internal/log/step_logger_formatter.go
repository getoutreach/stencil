// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the formatting code for the
// step logger.

package log

import (
	"strings"

	"atomicgo.dev/cursor"
	"github.com/gookit/color"
)

// chunks splits a string into multiple string slices based
// on the chunkSize provided, adapted from StackOverflow:
// https://stackoverflow.com/questions/25686109/split-string-by-length-in-golang
func chunks(s string, chunkSize int) []string {
	if s == "" {
		return nil
	}
	if chunkSize >= len(s) {
		return []string{s}
	}
	var chunks = make([]string, 0, (len(s)-1)/chunkSize+1)
	currentLen := 0
	currentStart := 0
	for i := range s {
		if currentLen == chunkSize {
			chunks = append(chunks, s[currentStart:i])
			currentLen = 0
			currentStart = i
		}
		currentLen++
	}
	chunks = append(chunks, s[currentStart:])
	return chunks
}

// print prints to the terminal, keeping sl.maxItems on the actual
// terminal, if we're a terminal. Print splits newlines into multiple
// recorded lines and accounts for them
func (sl *StepLogger) print(msg string, lvl Level) {
	prefix := " => "

	// \r -> \n cause go away windows
	formattedLine := strings.ReplaceAll(msg, "\r", "\n")

	for _, nline := range strings.Split(formattedLine, "\n") {
		// minus 4 comes from the format length
		maxLineLength := sl.termWidth - len([]rune(prefix))

		nline = strings.ReplaceAll(nline, "\n", "")

		// tabs are unknown how long they will be
		// so we just convert them to spaces.
		// see:
		// https://unix.stackexchange.com/questions/389255/determine-how-long-tabs-t-are-on-a-line
		nline = strings.ReplaceAll(nline, "\t", " ")

		for _, line := range chunks(nline, maxLineLength) {
			sl.record(Entry{Level: lvl, Message: prefix + line})
		}
	}

	sl.flush()
}

// flush prints the logs to the terminal
func (sl *StepLogger) flush() {
	cursor.Hide()
	defer cursor.Show()

	sl.Clean(false)

	// reverse iterate the logs, offsetting sl.maxItems, if the len is greater
	// than the max items, otherwise start at increasing order.
	printedLines := 0
	for i := range sl.logs {
		// default to increasing order
		pos := i

		// if the length is greater than the max items, reverse the order
		// and offset the position by the max items
		if len(sl.logs) > sl.maxItems {
			pos = ((len(sl.logs) - 1) - sl.maxItems) + i
		} else if i > (sl.maxItems - 1) {
			break
		}

		// if the position is greater than the length, we have reached the end
		if pos > len(sl.logs)-1 {
			break
		}

		color.Grayln(sl.logs[pos].Message)
		printedLines++
	}

	sl.lastPrinted = printedLines
}
