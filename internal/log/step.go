// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains the logic for steps

package log

// Step is a single step in an operation
type Step struct {
	name  string
	emoji string
	fn    func(*StepLogger) error
}

// Run runs the step
func (s *Step) Run(log *StepLogger) error {
	return s.fn(log)
}
