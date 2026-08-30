// internal/ui/render.go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// cellStyle is the colour and weight a row carries. Type word, id and title
// all take it, so a row reads as one unit rather than three fragments.
type cellStyle struct {
	tint     lipgloss.Color
	bold     bool
	hasColor bool
}

// styleFor resolves a bean's type into a colour and a weight. A type with no
// configured colour — task, by default — keeps the terminal's own text colour;
// it is the commonest row and earns no ink.
func styleFor(b *bean.Bean, cfg *config.Config) cellStyle {
	tc := cfg.GetType(b.Type)
	if tc == nil || tc.Color == "" {
		emphasis := tc != nil && tc.Emphasis
		return cellStyle{bold: emphasis}
	}
	return cellStyle{tint: ResolveColor(tc.Color), bold: tc.Emphasis, hasColor: true}
}

func (s cellStyle) render(text string) string {
	if text == "" {
		return text
	}
	st := lipgloss.NewStyle().Bold(s.bold)
	if s.hasColor {
		st = st.Foreground(s.tint)
	}
	return st.Render(text)
}

// statusCell renders the status in the decided form. Archive statuses are not
// bold: they are done, and done recedes.
func statusCell(b *bean.Bean, c Columns, cfg *config.Config) string {
	text := PadRight(c.StatusText(b), c.Status)
	colour := "overlay1"
	archive := true
	if sc := cfg.GetStatus(b.Status); sc != nil {
		colour = sc.Color
		archive = sc.Archive
	}
	return lipgloss.NewStyle().Foreground(ResolveColor(colour)).Bold(!archive).Render(text)
}

// prioCell renders the priority, or blank cells for normal.
func prioCell(b *bean.Bean, c Columns, cfg *config.Config) string {
	text := c.PrioText(b)
	if text == "" {
		return strings.Repeat(" ", c.Prio)
	}
	colour := ""
	if pc := cfg.GetPriority(b.Priority); pc != nil {
		colour = pc.Color
	}
	bold := b.Priority == "critical" || b.Priority == "high"
	return lipgloss.NewStyle().Foreground(ResolveColor(colour)).Bold(bold).
		Render(PadRight(text, c.Prio))
}

// progressCell renders a milestone's descendant completion.
func progressCell(p *Progress, c Columns) string {
	if p == nil || c.ProgressWidth == 0 {
		return strings.Repeat(" ", c.ProgressWidth)
	}
	filled := 0
	if p.Total > 0 {
		filled = (progressBarWidth*p.Done + p.Total/2) / p.Total
	}
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	bar := lipgloss.NewStyle().Foreground(ResolveColor("green")).
		Render(strings.Repeat("█", filled)) +
		lipgloss.NewStyle().Foreground(ResolveColor("surface2")).
			Render(strings.Repeat("░", progressBarWidth-filled))
	return bar + strings.Repeat(" ", c.Gap) + Muted.Render(PadRight(fmt.Sprintf("%d/%d", p.Done, p.Total), c.Counter))
}

// tagCell renders tags on exactly one line.
//
// Tags never hard-break: a word wrap once split #slug-tailwind-upgrade across
// three lines mid-word. The title is what must never be cut; tags are metadata
// and may be. The first tag is always shown, elided if it must be, because a
// column holding nothing but "+3" tells the reader nothing.
func tagCell(tags []string, width int) string {
	if width <= 0 || len(tags) == 0 {
		return ""
	}

	cell := func(tag string, room int) string {
		c := "#" + tag
		if DisplayWidth(c) <= room {
			return c
		}
		return Truncate(c, room)
	}

	marker := ""
	if len(tags) > 1 {
		marker = fmt.Sprintf("+%d", len(tags)-1)
	}
	room := width
	if marker != "" {
		room = width - DisplayWidth(marker) - 1
	}
	if room < 1 {
		room = 1
	}

	shown := []string{cell(tags[0], room)}
	used := DisplayWidth(shown[0])

	for i := 1; i < len(tags); i++ {
		remaining := len(tags) - i
		mark := fmt.Sprintf("+%d", remaining)
		reserve := DisplayWidth(mark) + 1
		next := cell(tags[i], width)
		if used+1+DisplayWidth(next) > width-reserve {
			// Nothing more fits. The marker must itself fit within what is
			// left after the tags already shown — if it does not, the
			// marker is dropped rather than let the cell overflow its
			// width, since staying within budget outranks reporting the
			// overflow count.
			plain := strings.Join(shown, " ")
			if used+1+DisplayWidth(mark) <= width {
				plain += " " + mark
			}
			return Muted.Render(plain)
		}
		shown = append(shown, next)
		used += 1 + DisplayWidth(next)
	}
	return Muted.Render(strings.Join(shown, " "))
}
