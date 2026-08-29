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
// running to the right margin of every line. Nothing here pads.
func RenderMarkdown(body string, width int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if width < 20 {
		width = 20
	}
	textWidth := width - 2 // two columns of indent
	if textWidth < 10 {
		textWidth = 10
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
			level := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
			text := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
			style := lipgloss.NewStyle().Bold(true)
			if level <= 2 {
				style = style.Foreground(ColorPrimary)
			}
			out = append(out, "", "  "+style.Render(text))
			continue
		}
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
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
