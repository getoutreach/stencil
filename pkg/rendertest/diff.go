// Copyright 2026 Outreach Corporation. All Rights Reserved.

// Description: This file contains a line diff in the unified format, for
// snapshot mismatches.

package rendertest

import (
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines around each change.
const diffContext = 3

// maxDiffCells limits the size of the table that the diff computes. Above it,
// the diff shows all lines as removed and added.
const maxDiffCells = 4_000_000

// diffLine is a line of a diff, with its kind: ' ' (unchanged), '-'
// (removed) or '+' (added).
type diffLine struct {
	kind byte
	text string
}

// unifiedDiff returns the line diff from a to b in the unified format, with
// the file names from and to.
func unifiedDiff(from, to, a, b string) string {
	lines := diffLines(splitLines(a), splitLines(b))

	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", from, to)

	// Each change gets diffContext lines around it, and changes whose
	// context touches or overlaps are in the same hunk.
	aLine, bLine, done := 1, 1, 0
	for first := indexOfChange(lines, 0); first >= 0; first = indexOfChange(lines, done) {
		last := first
		for {
			next := indexOfChange(lines, last+1)
			if next < 0 || next-last > 2*diffContext {
				break
			}
			last = next
		}

		lo, hi := max(first-diffContext, done), min(last+diffContext+1, len(lines))
		aLine, bLine = countLines(lines[done:lo], aLine, bLine)
		aEnd, bEnd := countLines(lines[lo:hi], aLine, bLine)
		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n", aLine, aEnd-aLine, bLine, bEnd-bLine)
		for _, l := range lines[lo:hi] {
			sb.WriteByte(l.kind)
			sb.WriteString(l.text)
			sb.WriteByte('\n')
		}
		aLine, bLine, done = aEnd, bEnd, hi
	}

	return sb.String()
}

// splitLines splits s into lines without their line endings.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// diffLines returns the lines of a and b, marked as unchanged, removed or
// added. The lines that a and b start and end with are unchanged, and
// diffMiddle compares the rest.
func diffLines(a, b []string) []diffLine {
	lines := make([]diffLine, 0, len(a)+len(b))

	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		lines = append(lines, diffLine{' ', a[prefix]})
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	lines = append(lines, diffMiddle(a[prefix:len(a)-suffix], b[prefix:len(b)-suffix])...)
	for _, l := range a[len(a)-suffix:] {
		lines = append(lines, diffLine{' ', l})
	}
	return lines
}

// diffMiddle returns the lines of a and b, marked as unchanged, removed or
// added, using a longest common subsequence.
func diffMiddle(a, b []string) []diffLine {
	lines := make([]diffLine, 0, len(a)+len(b))
	if len(a)*len(b) > maxDiffCells {
		for _, l := range a {
			lines = append(lines, diffLine{'-', l})
		}
		for _, l := range b {
			lines = append(lines, diffLine{'+', l})
		}
		return lines
	}

	// lcs[i*w+j] is the length of the longest common subsequence of a[i:]
	// and b[j:].
	w := len(b) + 1
	lcs := make([]int32, (len(a)+1)*w)
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i*w+j] = lcs[(i+1)*w+j+1] + 1
			} else {
				lcs[i*w+j] = max(lcs[(i+1)*w+j], lcs[i*w+j+1])
			}
		}
	}

	i, j := 0, 0
	for i < len(a) || j < len(b) {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			lines = append(lines, diffLine{' ', a[i]})
			i++
			j++
		case j == len(b) || (i < len(a) && lcs[(i+1)*w+j] >= lcs[i*w+j+1]):
			lines = append(lines, diffLine{'-', a[i]})
			i++
		default:
			lines = append(lines, diffLine{'+', b[j]})
			j++
		}
	}
	return lines
}

// indexOfChange returns the index of the first removed or added line at or
// after start, or -1 if there is none.
func indexOfChange(lines []diffLine, start int) int {
	for i := start; i < len(lines); i++ {
		if lines[i].kind != ' ' {
			return i
		}
	}
	return -1
}

// countLines returns the line numbers of a and b after lines, given those
// before them.
func countLines(lines []diffLine, aLine, bLine int) (aNext, bNext int) {
	for _, l := range lines {
		if l.kind != '+' {
			aLine++
		}
		if l.kind != '-' {
			bLine++
		}
	}
	return aLine, bLine
}
