// Copyright 2026 Outreach Corporation. Licensed under the Apache License 2.0.

// Description: Tests for the project manifest (service.yaml) auto-fixer.

package projectmanifest_test

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	projectmanifest "github.com/getoutreach/stencil/internal/lint/projectmanifest"
)

func TestFixBytesPrereleaseTrue(t *testing.T) {
	in := "name: s\nmodules:\n  - name: github.com/x/a\n    prerelease: true\n"
	fixed, applied, ok := projectmanifest.FixBytes([]byte(in))
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(applied))
	s := string(fixed)
	assert.Assert(t, strings.Contains(s, "channel: rc"))
	assert.Assert(t, !strings.Contains(s, "prerelease"))
}

func TestFixBytesPrereleaseFalseRemoved(t *testing.T) {
	in := "name: s\nmodules:\n  - name: github.com/x/a\n    prerelease: false\n"
	fixed, applied, ok := projectmanifest.FixBytes([]byte(in))
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(applied))
	assert.Assert(t, !strings.Contains(string(fixed), "prerelease"))
}

func TestFixBytesKeepsExistingChannel(t *testing.T) {
	in := "name: s\nmodules:\n  - name: github.com/x/a\n    channel: stable\n    prerelease: true\n"
	fixed, applied, ok := projectmanifest.FixBytes([]byte(in))
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, len(applied))
	assert.Assert(t, strings.Contains(string(fixed), "channel: stable"))
	assert.Assert(t, !strings.Contains(string(fixed), "prerelease"))
}

func TestFixBytesNoModulesNoOp(t *testing.T) {
	in := "name: s\narguments:\n  foo: bar\n"
	fixed, applied, ok := projectmanifest.FixBytes([]byte(in))
	assert.Equal(t, true, ok)
	assert.Equal(t, 0, len(applied))
	assert.Equal(t, in, string(fixed)) // no-op returns raw verbatim
}

func TestFixBytesMalformed(t *testing.T) {
	_, _, ok := projectmanifest.FixBytes([]byte("name: [unterminated\n"))
	assert.Equal(t, false, ok)
}

func TestFixBytesPreservesSurroundingComments(t *testing.T) {
	// A head comment on the module and an unrelated key survive the migration.
	in := "name: s\nmodules:\n  # our core dep\n  - name: github.com/x/a\n    channel: rc\n    prerelease: false\n"
	fixed, _, ok := projectmanifest.FixBytes([]byte(in))
	assert.Equal(t, true, ok)
	s := string(fixed)
	assert.Assert(t, strings.Contains(s, "# our core dep"))
	assert.Assert(t, strings.Contains(s, "channel: rc"))
	assert.Assert(t, !strings.Contains(s, "prerelease"))
}
