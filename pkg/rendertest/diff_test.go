// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains tests for the unified diff.

package rendertest

import (
	"fmt"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestUnifiedDiff(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want string
	}{
		{
			name: "changed line",
			a:    "a\nb\nc\n",
			b:    "a\nB\nc\n",
			want: "@@ -1,3 +1,3 @@\n a\n-b\n+B\n c\n",
		},
		{
			name: "added and removed lines",
			a:    "a\nb\n",
			b:    "b\nc\n",
			want: "@@ -1,2 +1,2 @@\n-a\n b\n+c\n",
		},
		{
			name: "empty side",
			a:    "",
			b:    "a\n",
			want: "@@ -1,0 +1,1 @@\n+a\n",
		},
		{
			name: "changes far apart are separate hunks",
			a:    "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n",
			b:    "x\n2\n3\n4\n5\n6\n7\n8\n9\ny\n",
			want: "@@ -1,4 +1,4 @@\n-1\n+x\n 2\n 3\n 4\n@@ -7,4 +7,4 @@\n 7\n 8\n 9\n-10\n+y\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, unifiedDiff("from", "to", tt.a, tt.b), "--- from\n+++ to\n"+tt.want)
		})
	}
}

func TestUnifiedDiffOfLargeInputs(t *testing.T) {
	var a, b strings.Builder
	for i := range 2100 {
		fmt.Fprintf(&a, "%d\n", i)
		fmt.Fprintf(&b, "%d\n", i+1)
	}

	diff := unifiedDiff("from", "to", a.String(), b.String())
	assert.Assert(t, strings.HasPrefix(diff, "--- from\n+++ to\n@@ -1,2100 +1,2100 @@\n-0\n"), diff[:50])
}
