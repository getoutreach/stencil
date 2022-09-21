// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the step logger

package log

import (
	"fmt"

	"atomicgo.dev/cursor"
	"golang.org/x/term"
)

// _ ensures that StepLogger implements the Logger interface
var _ Logger = &StepLogger{}

// StepLogger is a logger used for steps. Print functions print
// maxItems lines to the terminal, clearing them as it prints more.
//
// Leveled functions, such as Debug, Warn, and Error, print to the
// parent logger and are displayed, and persisted, in the output.
type StepLogger struct {
	// maxItems is the max lines to show in the step logger
	maxItems int

	// lastPrinted is the last number of lines printed
	lastPrinted int

	// parent is the parent logger
	parent Logger

	// logs contains all of the logs emitted during this step
	logs []Entry

	// termWidth is the width of the terminal
	termWidth int
}

// NewStepLogger create a new step logger
func NewStepLogger(parent Logger) *StepLogger {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 10
	}
	return &StepLogger{parent: parent, logs: make([]Entry, 0), maxItems: 10, termWidth: width}
}

// Clean cleans the step logger output, reset
// resets the number of lines printed if true
func (sl *StepLogger) Clean(reset bool) {
	if sl.lastPrinted != 0 {
		cursor.ClearLinesUp(sl.lastPrinted)
	}
	if reset {
		sl.lastPrinted = 0
	}
}

// Reset resets the step logger, cleaning up ephemeral logs (if present)
// and resetting the log buffer.
func (sl *StepLogger) Reset() {
	sl.Clean(true)
	sl.logs = make([]Entry, 0)
}

// GetLogs returns the logs for the step, note it does not
// clear them.
func (sl *StepLogger) GetLogs() []Entry {
	return sl.logs
}

// record records the line and clears the number of lines printed
func (sl *StepLogger) record(e Entry) {
	sl.logs = append(sl.logs, e)
}

// EphemeralPrint prints output to the ephemeral logger
func (sl *StepLogger) EphemeralPrint(a ...interface{}) {
	sl.print(fmt.Sprint(a...), LevelNone)
}

// EphemeralPrintf prints output to the ephemeral logger
func (sl *StepLogger) EphemeralPrintf(format string, a ...interface{}) {
	sl.print(fmt.Sprintf(format+"\n", a...), LevelNone)
}

// EphemeralPrintln prints output to the ephemeral logger
func (sl *StepLogger) EphemeralPrintln(a ...interface{}) {
	sl.EphemeralPrint(a...)
}

// Print prints to the parent logger
func (sl *StepLogger) Print(a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Print(a...)
}

// Printf prints to the parent logger
func (sl *StepLogger) Printf(format string, a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Printf(format, a...)
}

// Println prints to the parent logger
func (sl *StepLogger) Println(a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Println(a...)
}

// Debug prints to the parent logger
func (sl *StepLogger) Debug(a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Debug(a...)
}

// Debugf prints to the parent logger
func (sl *StepLogger) Debugf(format string, a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Debugf(format, a...)
}

// Warn prints to the parent logger
func (sl *StepLogger) Warn(a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Warn(a...)
}

// Warnf prints to the parent logger
func (sl *StepLogger) Warnf(format string, a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Warnf(format, a...)
}

// Error prints to the parent logger
func (sl *StepLogger) Error(a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Error(a...)
}

// Errorf prints to the parent logger
func (sl *StepLogger) Errorf(format string, a ...interface{}) {
	sl.Clean(true)
	defer sl.flush()

	sl.parent.Errorf(format, a...)
}

// ProgressBar returns the parent progress bar
func (sl *StepLogger) ProgressBar(max int64) *ProgressBar {
	return sl.parent.ProgressBar(max)
}
