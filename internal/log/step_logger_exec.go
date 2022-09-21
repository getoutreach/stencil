// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the functions for a step logger related
// to the io.Writer interface.

package log

// Write writes the given bytes to the step logger
// This is used to implement the io.Writer interface
func (sl *StepLogger) Write(b []byte) (int, error) {
	if b != nil {
		sl.EphemeralPrint(string(b))
	}
	return len(b), nil
}
