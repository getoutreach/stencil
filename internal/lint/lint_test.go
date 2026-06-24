// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the lint finding model.

package lint_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/getoutreach/stencil/internal/lint"
)

func TestFindingsCollectAndCount(t *testing.T) {
	var f lint.Findings
	f.Errorf("name", "name is required")
	f.Warnf("arguments.foo.type", "field %q is deprecated", "type")
	f.Add(lint.Finding{Severity: lint.SeverityError, Path: "type", Message: "unknown type"})

	items := f.Items()
	assert.Equal(t, 3, len(items))

	// order is preserved
	assert.Equal(t, lint.SeverityError, items[0].Severity)
	assert.Equal(t, "name", items[0].Path)
	assert.Equal(t, "name is required", items[0].Message)
	assert.Equal(t, lint.SeverityWarning, items[1].Severity)
	assert.Equal(t, "arguments.foo.type", items[1].Path)
	assert.Equal(t, `field "type" is deprecated`, items[1].Message)

	// method tally
	errs, warns := f.Counts()
	assert.Equal(t, 2, errs)
	assert.Equal(t, 1, warns)

	// free-function tally over a slice
	errs2, warns2 := lint.Counts(items)
	assert.Equal(t, 2, errs2)
	assert.Equal(t, 1, warns2)
}

func TestCountsEmpty(t *testing.T) {
	errs, warns := lint.Counts(nil)
	assert.Equal(t, 0, errs)
	assert.Equal(t, 0, warns)
}
