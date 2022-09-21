// Copyright 2022 Outreach Corporation. All Rights Reserved.

// Description: This file implements a progress bar

package log

import (
	"fmt"
	"os"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressBar is a progress bar for the terminal
type ProgressBar struct {
	logger *CLILogger
	pb     *progressbar.ProgressBar
	max    int64
}

// ProgressBar creates a new progress bar
func (l *CLILogger) ProgressBar(max int64) *ProgressBar {
	return &ProgressBar{
		logger: l,
		max:    max,
	}
}

func (p *ProgressBar) hooks() {
	p.logger.preLogHook = func() {
		p.logger.ClearLine()
	}
	p.logger.postLogHook = func() {
		p.Redraw()
	}
}

// Start starts the progress bar
func (p *ProgressBar) Start() {
	p.hooks()

	p.pb = progressbar.NewOptions64(p.max,
		progressbar.OptionSetDescription(""),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "#",
			SaucerHead:    "",
			SaucerPadding: "-",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	p.pb.RenderBlank() //nolint:errcheck // Why: best effort
}

// Redraw redraws the progress bar
func (p *ProgressBar) Redraw() {
	if p.pb == nil {
		return
	}
	// Note: This renders even if not 0%, despite the name
	p.pb.RenderBlank() //nolint:errcheck // Why: best effort
}

// UpdateMax changes the max value of the progress bar
func (p *ProgressBar) UpdateMax(max int64) {
	p.max = max

	if p.pb == nil {
		return
	}
	p.pb.ChangeMax64(max)
}

// Set sets the current value of the progress bar
func (p *ProgressBar) Set(n int64) {
	if p.pb == nil {
		return
	}
	p.pb.Set64(n) //nolint:errcheck // Why: best effort
}

// Inc increments the progress bar by 1
func (p *ProgressBar) Inc() {
	if p.pb == nil {
		return
	}
	p.pb.Add(1) //nolint:errcheck // Why: best effort
}

// Close finishes the progress bar and then closes it
func (p *ProgressBar) Close() {
	if p.pb == nil {
		return
	}
	p.pb.Close() //nolint:errcheck // Why: best effort

	p.logger.ClearLine()
	// undo the hooks
	p.logger.preLogHook = nil
	p.logger.postLogHook = nil
}
