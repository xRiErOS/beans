package commands

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/hmans/beans/pkg/bean"
)

// Layout constants for the TTY-rendered roadmap (variant beta, D13).
// See docs/roadmap-tty-output/DESIGN.md for the authoritative spec.
const (
	roadmapTitleCol = 17 // column where every title starts
	roadmapPrioW    = 8  // priority cell, right-aligned
	roadmapStatusW  = 11 // status cell, left-aligned
	roadmapIDW      = 4  // short ID cell, left-aligned
	roadmapRightW   = roadmapPrioW + 2 + roadmapStatusW + 2 + roadmapIDW // 27

	roadmapMinWidth = 80
	roadmapMaxWidth = 110
)

// roadmapShortID strips the repo prefix and returns the 4-character suffix.
// "beans-tquh" -> "tquh", "lean-stack-ewig" -> "ewig".
func roadmapShortID(id string) string {
	if i := strings.LastIndex(id, "-"); i >= 0 {
		return id[i+1:]
	}
	return id
}

// roadmapRightBlock renders the fixed-width attribute block: priority, status,
// short ID. Priority "normal" is never shown (D10); showPrio is false for
// container rows (milestone, epic).
func roadmapRightBlock(b *bean.Bean, showPrio bool) string {
	prio := ""
	if showPrio && b.Priority != "normal" {
		prio = b.Priority
	}
	return fmt.Sprintf("%*s  %-*s  %-*s",
		roadmapPrioW, prio,
		roadmapStatusW, b.Status,
		roadmapIDW, roadmapShortID(b.ID))
}

// roadmapWrapTitle word-wraps a title to the given cell width. Words longer
// than the width are hard-broken. Never returns an empty slice — an empty
// title yields one empty line. Widths are counted in runes (D16): correct for
// Latin incl. umlauts; CJK/emoji titles wrap early.
func roadmapWrapTitle(title string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(title)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	cur := ""
	flush := func() {
		lines = append(lines, cur)
		cur = ""
	}
	for _, w := range words {
		// Hard-break words that cannot fit on a line of their own.
		for utf8.RuneCountInString(w) > width {
			if cur != "" {
				flush()
			}
			r := []rune(w)
			lines = append(lines, string(r[:width]))
			w = string(r[width:])
		}
		switch {
		case cur == "":
			cur = w
		case utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) <= width:
			cur += " " + w
		default:
			flush()
			cur = w
		}
	}
	if cur != "" {
		flush()
	}
	return lines
}

// roadmapLine renders one bean as one or more physical lines. The first line
// carries prefix, title and the right-hand attribute block; continuation lines
// carry only the wrapped title at the hanging indent (D07). The returned
// string has no trailing newline.
func roadmapLine(prefix string, b *bean.Bean, showPrio bool, width int) string {
	titleW := width - roadmapTitleCol - 2 - roadmapRightW
	if titleW < 1 {
		titleW = 1
	}
	parts := roadmapWrapTitle(b.Title, titleW)

	prefixW := utf8.RuneCountInString(prefix)
	var first string
	if prefixW >= roadmapTitleCol {
		// D17: raster locally broken, keep exactly one separating space.
		first = prefix + " " + parts[0]
	} else {
		first = prefix + strings.Repeat(" ", roadmapTitleCol-prefixW) + parts[0]
	}

	pad := width - roadmapRightW - utf8.RuneCountInString(first)
	if pad < 2 {
		pad = 2
	}

	var sb strings.Builder
	sb.WriteString(first)
	sb.WriteString(strings.Repeat(" ", pad))
	sb.WriteString(roadmapRightBlock(b, showPrio))
	for _, cont := range parts[1:] {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat(" ", roadmapTitleCol))
		sb.WriteString(cont)
	}
	return sb.String()
}
