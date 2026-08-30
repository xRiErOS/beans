// internal/ui/markdown.go
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderMarkdown lays out a bean body for the terminal: headings, bullets,
// fenced code, everything else as wrapped prose.
//
// It replaces glamour, whose padding leaks into the output as coloured spaces
// running to the right margin of every line. Nothing here pads, and nothing
// here paints a background — both were glamour's headline defects, not
// incidental ones.
func RenderMarkdown(body string, width int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	// textWidth is what WrapText actually gets: width minus the two columns
	// of indent every rendered line carries. It is not floored at some
	// larger "readable" minimum the way minRenderableTitle floors a title
	// column — RenderMarkdown(body, 12) must emit lines that fit a 12-cell
	// terminal, not silently claim 20. WrapText already applies the one
	// genuine floor this layout needs (width < 1 becomes 1, see width.go)
	// and documents the narrow case where even that still overflows by a
	// cell or two; textWidth only guards against going negative into it.
	textWidth := width - 2
	if textWidth < 1 {
		textWidth = 1
	}

	var out []string
	inCode := false

	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimRight(raw, " \t")
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			out = append(out, "  "+Muted.Render(line))
			continue
		}
		if trimmed == "" {
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			// len() here counts a run of literal '#' bytes, which are
			// single-byte ASCII in any encoding — byte count and cell
			// count agree. That stops being true the moment this measures
			// the heading's rendered text, which is why the text itself is
			// wrapped through WrapText/DisplayWidth below, never len().
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			text := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			style := lipgloss.NewStyle().Bold(true)
			if level <= 2 {
				style = style.Foreground(ColorPrimary)
			}
			// A blank line separates a heading from what precedes it, but
			// only when something precedes it — a heading opening the body
			// should not start with a blank line above it.
			if len(out) > 0 {
				out = append(out, "")
			}
			for _, p := range WrapText(text, textWidth) {
				out = append(out, "  "+style.Render(p))
			}
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			// len() here counts leading space bytes, again single-byte
			// ASCII, so byte count and cell count agree for this measurement.
			indent := len(line) - len(strings.TrimLeft(line, " "))
			parts := WrapText(trimmed[2:], textWidth-indent-2)
			out = append(out, strings.Repeat(" ", 2+indent)+TreeLine.Render("•")+" "+parts[0])
			for _, p := range parts[1:] {
				out = append(out, strings.Repeat(" ", 4+indent)+p)
			}
			continue
		}
		for _, p := range WrapText(trimmed, textWidth) {
			out = append(out, "  "+p)
		}
	}
	return strings.Join(out, "\n")
}
