// internal/commands/showtable.go
package commands

import (
	"context"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/config"
)

// The label column is sized from the label vocabulary below rather than from
// the bean at hand: two beans rendered by two separate calls must produce the
// same geometry, or reading down a column -- the entire point of this view --
// stops working. An extra front matter key longer than the widest fixed label
// is the one thing that can still widen the column, so it is measured too,
// per bean, and that is a deliberate trade: an unusually long key is visible
// and rare, whereas a silently uneven grid is neither.
var tableLabelVocabulary = []string{
	"title:", "id:", "type:", "status:", "priority:", "tags:",
	"parent:", "blocked by:", "blocking:", "created:", "updated:", "order:",
}

// tableRow is one line of the grid before layout: a label and its value, or
// several label/value pairs that share the line. A pair's label is padded to
// its own fixed width so that status and priority start at the same cell in
// every bean, which a plain strings.Join would not guarantee.
type tableRow struct {
	label string
	value string
	pairs []tablePair
}

type tablePair struct {
	label string
	value string
	// width is the cell budget for the value, taken from the config
	// vocabulary (the longest type/status/priority name) so the pair
	// columns line up across beans.
	width int
}

// renderBeanTable lays out one bean's whole front matter as a label/value
// grid, capped at width cells including the borders.
//
// It is an arrangement of the same fields renderBeanHeader prints, not a
// selection of them: dropping one here would reintroduce exactly the defect
// beans-p1d0 fixed, and TestTableCarriesEveryFrontMatterField pins that.
func renderBeanTable(b *bean.Bean, cfg *config.Config, width int) string {
	rows := beanTableRows(b, cfg)

	labelWidth := 0
	for _, l := range tableLabelVocabulary {
		if w := ui.DisplayWidth(l); w > labelWidth {
			labelWidth = w
		}
	}
	for _, r := range rows {
		if w := ui.DisplayWidth(r.label); w > labelWidth {
			labelWidth = w
		}
	}

	// A row is "│ " + label + " │ " + value + " │": seven cells of border
	// and padding on top of the two text columns.
	valueWidth := width - labelWidth - 7
	if valueWidth < 1 {
		valueWidth = 1
	}

	var sb strings.Builder
	border := func(left, mid, right string) {
		sb.WriteString(ui.TreeLine.Render(
			left+strings.Repeat("─", labelWidth+2)+mid+strings.Repeat("─", valueWidth+2)+right) + "\n")
	}
	bar := ui.TreeLine.Render("│")

	border("┌", "┬", "┐")
	for _, r := range rows {
		for i, line := range tableRowLines(r, valueWidth) {
			label := ""
			if i == 0 {
				label = r.label
			}
			sb.WriteString(bar + " " + ui.Muted.Render(ui.PadRight(label, labelWidth)) +
				" " + bar + " " + padVisible(line, valueWidth) + " " + bar + "\n")
		}
	}
	border("└", "┴", "┘")

	return sb.String()
}

// tableRowLines renders a row's value into the lines of its cell: a plain
// value wraps, a paired row stays on one line because its cells are already
// budgeted.
func tableRowLines(r tableRow, valueWidth int) []string {
	if len(r.pairs) > 0 {
		return []string{padPairs(r.pairs, valueWidth)}
	}
	if r.value == "" {
		return []string{""}
	}
	return ui.WrapText(r.value, valueWidth)
}

// padPairs joins the pairs of one row, padding each value to its budget so
// the following label starts at a fixed cell. The last pair is not padded --
// trailing spaces before the border are the caller's job.
func padPairs(pairs []tablePair, valueWidth int) string {
	var parts []string
	for i, p := range pairs {
		value := p.value
		if i < len(pairs)-1 {
			value = padVisible(value, p.width)
		}
		// The first pair of a row carries no label of its own -- the row's
		// left column already names it ("type:", "created:"). Rendering an
		// empty label with its separating space would indent that value by
		// one cell and break the raster against every other row.
		cell := value
		if p.label != "" {
			cell = ui.Muted.Render(p.label) + " " + value
		}
		parts = append(parts, cell)
	}
	line := strings.Join(parts, "  ")
	if visibleWidth(line) > valueWidth {
		// A narrow --max-width cannot fit the pairs; cutting is better
		// than breaking the border, and the cut is visible.
		return ui.Truncate(stripANSI(line), valueWidth)
	}
	return line
}

// padVisible pads s to width counting only visible cells, so a styled value
// keeps the grid aligned. ui.PadRight would count the ANSI sequences.
func padVisible(s string, width int) string {
	if pad := width - visibleWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

func visibleWidth(s string) int {
	return ui.DisplayWidth(stripANSI(s))
}

// stripANSI removes the SGR sequences lipgloss emits. Width arithmetic must
// see the text a terminal shows, not the bytes it consumes.
func stripANSI(s string) string {
	var sb strings.Builder
	for i := 0; i < len(s); {
		if s[i] == 0x1b {
			j := i
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				i = j + 1
				continue
			}
			break
		}
		sb.WriteByte(s[i])
		i++
	}
	return sb.String()
}

// beanTableRows turns one bean into the sketch's row sequence. The order is
// the PO's: what a thing is, then how it stands, then how it relates, then
// the free text, then the managed stamps.
func beanTableRows(b *bean.Bean, cfg *config.Config) []tableRow {
	tint := lipgloss.NewStyle()
	if tc := cfg.GetType(b.Type); tc != nil {
		tint = tint.Bold(tc.Emphasis)
		if tc.Color != "" {
			tint = tint.Foreground(ui.ResolveColor(tc.Color))
		}
	}

	rows := []tableRow{
		{label: "title:", value: b.Title},
		{label: "id:", value: tint.Render(b.ID)},
	}

	statusCell := b.Status
	statusColor, statusBold := "gray", true
	if sc := cfg.GetStatus(b.Status); sc != nil {
		statusColor, statusBold = sc.Color, !sc.Archive
	}
	statusCell = lipgloss.NewStyle().Foreground(ui.ResolveColor(statusColor)).
		Bold(statusBold).Render(b.Status)
	if implicit, from := core.ImplicitStatus(b.ID); implicit != "" {
		statusCell += " " + ui.Muted.Render("↑"+implicit+" ("+from+")")
	}

	priorityCell := ""
	if b.Priority != "" {
		priorityColor := "gray"
		if pc := cfg.GetPriority(b.Priority); pc != nil {
			priorityColor = pc.Color
		}
		priorityCell = lipgloss.NewStyle().Foreground(ui.ResolveColor(priorityColor)).
			Render(b.Priority)
	}

	rows = append(rows, tableRow{label: "type:", pairs: []tablePair{
		{label: "", value: tint.Render(b.Type), width: vocabularyWidth(cfg.TypeNames())},
		{label: "status:", value: statusCell, width: vocabularyWidth(cfg.StatusNames())},
		{label: "priority:", value: priorityCell, width: vocabularyWidth(cfg.PriorityNames())},
	}})

	if len(b.Tags) > 0 {
		parts := make([]string, len(b.Tags))
		for i, t := range b.Tags {
			parts[i] = "#" + t
		}
		rows = append(rows, tableRow{label: "tags:", value: strings.Join(parts, " ")})
	}

	if b.Parent != "" {
		rows = append(rows, tableRow{label: "parent:", value: describeRelated(b.Parent, cfg)})
	}
	for _, id := range b.BlockedBy {
		rows = append(rows, tableRow{label: "blocked by:", value: describeRelated(id, cfg)})
	}
	for _, id := range b.Blocking {
		rows = append(rows, tableRow{label: "blocking:", value: describeRelated(id, cfg)})
	}

	if len(b.Extra) > 0 {
		keys := make([]string, 0, len(b.Extra))
		for k := range b.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rows = append(rows, tableRow{label: k + ":", value: formatExtraValue(b.Extra[k])})
		}
	}

	var stamps []tablePair
	if b.CreatedAt != nil {
		stamps = append(stamps, tablePair{value: b.CreatedAt.Format("2006-01-02 15:04 UTC"), width: 20})
	}
	if b.UpdatedAt != nil {
		stamps = append(stamps, tablePair{label: "updated:", value: b.UpdatedAt.Format("2006-01-02 15:04 UTC"), width: 20})
	}
	if b.Order != "" {
		stamps = append(stamps, tablePair{label: "order:", value: b.Order, width: 4})
	}
	if len(stamps) > 0 {
		rows = append(rows, tableRow{label: "created:", pairs: stamps})
	}

	return rows
}

// vocabularyWidth is the widest name in a config enum, which is what makes a
// pair column's width a property of the configuration rather than of the bean
// being rendered.
func vocabularyWidth(names []string) int {
	w := 0
	for _, n := range names {
		if d := ui.DisplayWidth(n); d > w {
			w = d
		}
	}
	return w
}

// describeRelated names a related bean by type and title, falling back to the
// bare id. A parent from another store, or one that has been deleted, is
// information the reader still needs -- an empty cell or an error would be
// worse than the id.
func describeRelated(id string, cfg *config.Config) string {
	resolver := &beangraph.CoreResolver{Core: core}
	related, err := resolver.Bean(context.Background(), id)
	if err != nil || related == nil {
		return id
	}

	tint := lipgloss.NewStyle()
	if tc := cfg.GetType(related.Type); tc != nil {
		tint = tint.Bold(tc.Emphasis)
		if tc.Color != "" {
			tint = tint.Foreground(ui.ResolveColor(tc.Color))
		}
	}
	// The separator is a middle dot rather than spacing: ui.WrapText splits
	// on fields, so a two-space gap between type and title survives only
	// until the value wraps, and "epic show --table: ..." then reads as one
	// run-on phrase.
	parts := []string{}
	if related.Type != "" {
		parts = append(parts, tint.Render(related.Type))
	}
	if related.Title != "" {
		parts = append(parts, related.Title)
	}
	parts = append(parts, ui.Muted.Render(id))
	return strings.Join(parts, " · ")
}
