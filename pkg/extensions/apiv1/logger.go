// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements a simple io.Writer that writes
// to a function with a fmt.Print signature.

package apiv1

import "io"

// _ is a implementation check
var _ io.Writer = &logger{}

// logger implements io.Writer to write to a function with a fmt.Print signature
type logger struct {
	fn func(args ...interface{})
}

// Write writes the data to the logger
func (l *logger) Write(p []byte) (n int, err error) {
	l.fn("[go-plugin] ", string(p))
	return len(p), nil
}
