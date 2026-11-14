// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains the ErrorResult, the error of a failed
// render.

package rendertest

import (
	"errors"
	"strings"
	"testing"
)

// ErrorResult is the error of a failed render. Get it with
// Result.HasError.
type ErrorResult struct {
	// err is the render error
	err error
}

// Error returns the render error.
func (er *ErrorResult) Error() error {
	return er.err
}

// Contains checks that the error message contains substr.
func (er *ErrorResult) Contains(tb testing.TB, substr string) {
	tb.Helper()

	if !strings.Contains(er.err.Error(), substr) {
		tb.Errorf("expected the error to contain %q, got: %v", substr, er.err)
	}
}

// Is checks that the error matches target, as errors.Is does. It fails the
// test with Fatal if it does not.
func (er *ErrorResult) Is(tb testing.TB, target error) {
	tb.Helper()

	if !errors.Is(er.err, target) {
		tb.Fatalf("expected the error to match %v, got: %v", target, er.err)
	}
}

// As finds the first error that matches target, as errors.As does, and sets
// target to it. It fails the test with Fatal if there is none. target must
// be a non-nil pointer.
func (er *ErrorResult) As(tb testing.TB, target any) {
	tb.Helper()

	if !errors.As(er.err, target) {
		tb.Fatalf("expected the error to have an error of type %T, got: %v", target, er.err)
	}
}
