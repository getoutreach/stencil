// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the logic for log levels

package log

import "github.com/gookit/color"

// Level is the level for logging
type Level string

// This block contains the valid log levels
const (
	// LevelNone is used for when a log line does _not_ have a valid level
	// e.g. when the log line is just a formatting message
	LevelNone Level = ""

	// LevelDebug is the debug level
	LevelDebug Level = "debug"

	// LevelInfo is the info level
	LevelInfo Level = "info"

	// LevelWarn is the warn level
	LevelWarn Level = "warn"

	// LevelError is the error level
	LevelError Level = "error"
)

// logColors contains a map of log levels to colors for formatting
var logColors = map[Level]func(a ...interface{}) string{
	// light gray
	LevelDebug: color.New(37).Sprint,
	LevelInfo:  color.Cyan.Sprint,
	LevelWarn:  color.Yellow.Sprint,
	LevelError: color.Red.Sprint,
}
