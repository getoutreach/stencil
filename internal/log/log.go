// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This package contains a logger for CLIs.

// Package log contains a CLI-oriented logger that can be used to
// build interactive CLIs.
package log

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

// CLILogger is a level aware logger oriented towards being used in CLIs
type CLILogger struct {
	// output is the output to write to
	output io.Writer

	// started is the time the logger was created
	started time.Time

	// postLogHook is ran after every log line if set
	postLogHook func()

	// preLogHook is ran before every log line if set
	preLogHook func()

	// isTerminal is true if the output is a terminal
	isTerminal bool
}

// Entry is a single log entry
type Entry struct {
	// Level is the log level for this entry
	Level Level

	// Message is the raw string to be printed out
	Message string
}

// Logger is an interface for a logger
type Logger interface {
	Println(...interface{})
	Print(...interface{})
	Printf(string, ...interface{})
	Debug(...interface{})
	Debugf(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	ProgressBar(int64) *ProgressBar
}

// New creates a new logger
func New() *CLILogger {
	return &CLILogger{
		output:     os.Stdout,
		started:    time.Now(),
		isTerminal: term.IsTerminal(0),
	}
}

// Print prints a message to the output
func (l *CLILogger) Print(a ...interface{}) {
	if l.postLogHook != nil {
		defer l.postLogHook()
	}
	if l.preLogHook != nil {
		l.preLogHook()
	}

	fmt.Fprint(l.output, a...)
}

// Println prints a message to the output with a newline
func (l *CLILogger) Println(a ...interface{}) {
	if l.postLogHook != nil {
		defer l.postLogHook()
	}
	if l.preLogHook != nil {
		l.preLogHook()
	}

	fmt.Fprintln(l.output, a...)
}

// Printf prints a formatted message to the output
func (l *CLILogger) Printf(format string, a ...interface{}) {
	if l.postLogHook != nil {
		defer l.postLogHook()
	}
	if l.preLogHook != nil {
		l.preLogHook()
	}

	fmt.Fprintf(l.output, format, a...)
}

// Format formats a log line for printing
func (l *CLILogger) format(e *Entry) string {
	// If there is no level, just return the message
	if e.Level == LevelNone {
		return e.Message
	}

	return fmt.Sprintf("%s %s", logColors[e.Level](e.Level), e.Message)
}

// Debugln prints a debug message
func (l *CLILogger) Debug(a ...interface{}) {
	return
	l.Println(l.format(&Entry{
		Level:   LevelDebug,
		Message: fmt.Sprint(a...),
	}))
}

// Debugf prints a debug message
func (l *CLILogger) Debugf(format string, a ...interface{}) {
	return
	l.Println(l.format(&Entry{
		Level:   LevelDebug,
		Message: fmt.Sprintf(format, a...),
	}))
}

// Warnln prints a warning message
func (l *CLILogger) Warn(a ...interface{}) {
	l.Println(l.format(&Entry{
		Level:   LevelWarn,
		Message: fmt.Sprint(a...),
	}))
}

// Warnf prints a warning message
func (l *CLILogger) Warnf(format string, a ...interface{}) {
	l.Println(l.format(&Entry{
		Level:   LevelWarn,
		Message: fmt.Sprintf(format, a...),
	}))
}

// Errorln prints an error message
func (l *CLILogger) Error(a ...interface{}) {
	l.Println(l.format(&Entry{
		Level:   LevelError,
		Message: fmt.Sprint(a...),
	}))
}

// Errorf prints an error message
func (l *CLILogger) Errorf(format string, a ...interface{}) {
	l.Println(l.format(&Entry{
		Level:   LevelError,
		Message: fmt.Sprintf(format, a...),
	}))
}

// NewOperation creates a new operation for adding/running a set of steps
//
// Note: This is not thread safe, the parent logger will be used.
func (l *CLILogger) NewOperation() *Operation {
	return &Operation{
		logger: l,
	}
}
