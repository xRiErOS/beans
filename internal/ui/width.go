// internal/ui/width.go
package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// DisplayWidth is the width of s in terminal cells.
//
// Every column calculation must go through this rather than len(): the tree
// connectors are multi-byte, and a byte count reports overlong lines that are
// not overlong. Call it on plain text only — ANSI sequences would be counted.
func DisplayWidth(s string) int {
	return runewidth.StringWidth(s)
}

// PadRight fills s with spaces up to width. It never cuts; a string wider than
// width comes back unchanged, so a padding bug cannot silently swallow text.
func PadRight(s string, width int) string {
	if pad := width - DisplayWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// Truncate shortens s to width cells, marking the cut with an ellipsis. The
// ellipsis itself counts toward width, so callers get exactly the column
// budget they asked for, never one cell more.
func Truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if DisplayWidth(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return runewidth.Truncate(s, width, "…")
}

// WrapText breaks s into lines of at most width cells. Words longer than width
// are broken hard, because leaving them whole would overflow the column. The
// result always has at least one element, so callers can index [0] safely.
func WrapText(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	cur := ""
	for _, w := range words {
		for DisplayWidth(w) > width {
			if cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
			head := runewidth.Truncate(w, width, "")
			lines = append(lines, head)
			w = w[len(head):]
		}
		switch {
		case cur == "":
			cur = w
		case DisplayWidth(cur)+1+DisplayWidth(w) <= width:
			cur += " " + w
		default:
			lines = append(lines, cur)
			cur = w
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
