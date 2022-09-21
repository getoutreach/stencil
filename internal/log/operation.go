// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file contains operation/step logic for logging.

package log

import "github.com/gookit/color"

// Operation is a container for a list of steps that will be executed
// and logged sequentially.
type Operation struct {
	logger *CLILogger
	steps  []*Step
}

// AddStep adds a step to the operation
func (o *Operation) AddStep(name, emoji string, fn func(*StepLogger) error) {
	s := &Step{
		name:  name,
		emoji: emoji,
		fn:    fn,
	}

	o.steps = append(o.steps, s)
}

// Run runs all the steps in the operation
func (o *Operation) Run() error {
	totalSteps := len(o.steps)
	for i, s := range o.steps {
		stepNumber := i + 1
		o.logger.Printf(color.Gray.Sprintf("[%d/%d]", stepNumber, totalSteps)+" %s  %s \n", s.emoji, s.name)
		sl := NewStepLogger(o.logger)
		sl.Clean(false)
		if err := s.Run(sl); err != nil {
			for _, e := range sl.GetLogs() {
				o.logger.Println(e)
			}
			return err
		}
	}

	return nil
}
