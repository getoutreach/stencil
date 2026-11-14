// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains a testing.TB that records failures, to
// test assertions.

package rendertest_test

import (
	"errors"
	"fmt"
	"testing"

	"gotest.tools/v3/assert"
)

// errFatal stops a function that called Fatal on a recordingTB.
var errFatal = errors.New("fatal")

// recordingTB is a testing.TB that records failures instead of failing the
// test. Fatal and FailNow stop the caller with a panic of errFatal, which
// record recovers.
type recordingTB struct {
	*testing.T

	errors []string
	fatal  bool
}

func (r *recordingTB) Error(args ...any) {
	r.errors = append(r.errors, fmt.Sprint(args...))
}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *recordingTB) Fatal(args ...any) {
	r.Error(args...)
	r.FailNow()
}

func (r *recordingTB) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	r.FailNow()
}

func (r *recordingTB) FailNow() {
	panic(errFatal)
}

// record runs fn with a recordingTB and returns it.
func record(t *testing.T, fn func(tb *recordingTB)) *recordingTB {
	t.Helper()

	r := &recordingTB{T: t}
	func() {
		defer func() {
			switch p := recover(); p {
			case nil:
			case errFatal:
				r.fatal = true
			default:
				panic(p)
			}
		}()
		fn(r)
	}()
	return r
}

// assertPasses checks that fn reports no failure.
func assertPasses(t *testing.T, fn func(tb *recordingTB)) {
	t.Helper()

	r := record(t, fn)
	assert.Assert(t, !r.fatal && len(r.errors) == 0, "expected no failure, got %q", r.errors)
}

// assertErrors checks that fn reports a failure with Error, not Fatal.
func assertErrors(t *testing.T, fn func(tb *recordingTB)) {
	t.Helper()

	r := record(t, fn)
	assert.Assert(t, !r.fatal, "expected Error, got Fatal: %q", r.errors)
	assert.Equal(t, len(r.errors), 1, "expected one failure")
}

// assertFatal checks that fn reports a failure with Fatal.
func assertFatal(t *testing.T, fn func(tb *recordingTB)) {
	t.Helper()

	r := record(t, fn)
	assert.Assert(t, r.fatal, "expected Fatal, got %q", r.errors)
}
