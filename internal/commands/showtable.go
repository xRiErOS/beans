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
	"title:", "tags:", "parent:", "blocked by:", "blocking:",
}

// bandIDWidth budgets the id cell in the top band. Ids are prefix plus a
// four-character suffix, which lands at twelve cells for the common prefixes,
// and padding to a constant is what keeps "type:" starting at the same cell
// in every bean. A longer id pushes the rest of the band right, which is
// visible and rare -- the same trade the label column makes.
const bandIDWidth = 12

// tableBlock is one horizontally ruled section of the grid. A band spans the
// full width and carries several label/value pairs, left-packed with one
// right-aligned trailer; a field is the two-column label/value form whose
// value wraps.
type tableBlock struct {
	band  bool
	label string
	value string
	left  []string
	right string
}

// renderBeanTable lays out one bean's whole front matter, capped at width
// cells including the borders.
//
// The arrangement is the PO's: a band across the top carrying what the bean
// *is* (id, type, status, and priority against the right edge), a band across
// the bottom carrying the managed stamps, and between them one horizontally
// ruled field per line of front matter. The bands separate identity and
// bookkeeping from content, which is what a flat label column could not do --
// there, id and created sat in the same shape as the title.
//
// It is an arrangement of the same fields renderBeanHeader prints, not a
// selection of them: dropping one here would reintroduce exactly the defect
// beans-p1d0 fixed, and TestTableCarriesEveryFrontMatterField pins that.
func renderBeanTable(b *bean.Bean, cfg *config.Config, width int) string {
	blocks := beanTableBlocks(b, cfg)

	labelWidth := 0
	for _, l := range tableLabelVocabulary {
		if w := ui.DisplayWidth(l); w > labelWidth {
			labelWidth = w
		}
	}
	for _, bl := range blocks {
		if w := ui.DisplayWidth(bl.label); w > labelWidth {
			labelWidth = w
		}
	}

	// A field row is "│ " + label + " │ " + value + " │": seven cells of
	// border and padding on top of the two text columns. A band row is
	// "│ " + content + " │": four.
	valueWidth := width - labelWidth - 7
	if valueWidth < 1 {
		valueWidth = 1
	}
	bandWidth := width - 4
	if bandWidth < 1 {
		bandWidth = 1
	}

	bar := ui.TreeLine.Render("│")
	rule := func(left, right string) string {
		// The rules run straight through the column line rather than
		// meeting it in a ┼: they separate whole records, and a junction
		// on every one of them turns the grid into graph paper.
		return ui.TreeLine.Render(left+strings.Repeat("─", width-2)+right) + "\n"
	}

	var sb strings.Builder
	sb.WriteString(rule("┌", "┐"))
	for i, bl := range blocks {
		if i > 0 {
			sb.WriteString(rule("├", "┤"))
		}
		if bl.band {
			sb.WriteString(bar + " " + padVisible(packBand(bl, bandWidth), bandWidth) + " " + bar + "\n")
			continue
		}
		for j, line := range wrapFieldValue(bl, valueWidth) {
			label := ""
			if j == 0 {
				label = bl.label
			}
			sb.WriteString(bar + " " + ui.Muted.Render(ui.PadRight(label, labelWidth)) +
				" " + bar + " " + padVisible(line, valueWidth) + " " + bar + "\n")
		}
	}
	sb.WriteString(rule("└", "┘"))

	return sb.String()
}

// packBand left-packs a band's pairs and pushes its trailer against the right
// edge, which is what makes priority and order findable without reading the
// line: they are always in the same corner.
func packBand(bl tableBlock, bandWidth int) string {
	left := strings.Join(bl.left, "    ")
	if bl.right == "" {
		return left
	}

	gap := bandWidth - visibleWidth(left) - visibleWidth(bl.right)
	if gap < 1 {
		// Too narrow to separate the two: cut the left group so the
		// trailer survives, since it is the shorter and the more
		// positional of the two.
		left = ui.Truncate(stripANSI(left), bandWidth-visibleWidth(bl.right)-1)
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + bl.right
}

// wrapFieldValue folds a field's value and pushes its trailer -- the related
// bean's id -- against the right edge of the first line.
//
// Right-aligning the id is what makes a relation scannable: it lands in the
// same column in every row, so comparing two relations is reading one column
// rather than two phrases of different length. It also settles where the id
// goes when the cell wraps, which neither leading nor trailing it did: a
// leading id interrupted the type and title, a trailing one ended up alone
// on the continuation line looking like a truncated remnant.
func wrapFieldValue(bl tableBlock, valueWidth int) []string {
	if bl.right == "" {
		return wrapVisible(bl.value, valueWidth)
	}

	trailer := visibleWidth(bl.right)
	first := valueWidth - trailer - 2
	if first < 1 {
		// No room to share the line: the id keeps its own, since it is
		// the part that identifies the row.
		return append([]string{padLeftVisible(bl.right, valueWidth)},
			wrapVisible(bl.value, valueWidth)...)
	}

	lines := wrapVisibleFirst(bl.value, first, valueWidth)
	gap := valueWidth - visibleWidth(lines[0]) - trailer
	lines[0] = lines[0] + strings.Repeat(" ", gap) + bl.right
	return lines
}

// padLeftVisible right-aligns s in width cells, counting visible cells only.
func padLeftVisible(s string, width int) string {
	if pad := width - visibleWidth(s); pad > 0 {
		return strings.Repeat(" ", pad) + s
	}
	return s
}

// wrapVisibleFirst wraps s with a narrower budget for the first line, which
// is what leaves room for a right-aligned trailer beside it.
func wrapVisibleFirst(s string, firstWidth, width int) []string {
	lines := wrapVisible(s, firstWidth)
	if len(lines) < 2 {
		return lines
	}
	// Re-flow everything after the first line at the full width, so the
	// narrowing costs one line's worth of words rather than the whole
	// cell's.
	rest := strings.Join(lines[1:], " ")
	return append(lines[:1], wrapVisible(rest, width)...)
}

// wrapVisible wraps s to width *visible* cells, leaving the ANSI sequences a
// styled value carries intact.
//
// ui.WrapText cannot be used directly here: it measures the string it is
// handed, and a styled value carries escape bytes that occupy no cells, so a
// coloured relation wrapped several cells early while the grid around it
// stayed aligned. Splitting on whitespace keeps each sequence inside the
// token that opened it, so no line can end mid-escape; a single token wider
// than the column is handed to ui.WrapText stripped, because there is no way
// to hard-break inside a coloured run without splitting its sequence.
func wrapVisible(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	tokens := strings.Fields(s)
	if len(tokens) == 0 {
		return []string{""}
	}

	var lines []string
	cur := ""
	for _, tok := range tokens {
		switch {
		case visibleWidth(tok) > width:
			if cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
			lines = append(lines, ui.WrapText(stripANSI(tok), width)...)
		case cur == "":
			cur = tok
		case visibleWidth(cur)+1+visibleWidth(tok) <= width:
			cur += " " + tok
		default:
			lines = append(lines, cur)
			cur = tok
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
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

// beanTableBlocks turns one bean into the sketch's sequence: identity band,
// then one ruled field per front matter entry, then the stamps band.
func beanTableBlocks(b *bean.Bean, cfg *config.Config) []tableBlock {
	tint := lipgloss.NewStyle()
	if tc := cfg.GetType(b.Type); tc != nil {
		tint = tint.Bold(tc.Emphasis)
		if tc.Color != "" {
			tint = tint.Foreground(ui.ResolveColor(tc.Color))
		}
	}

	statusColor, statusBold := "gray", true
	if sc := cfg.GetStatus(b.Status); sc != nil {
		statusColor, statusBold = sc.Color, !sc.Archive
	}
	statusCell := lipgloss.NewStyle().Foreground(ui.ResolveColor(statusColor)).
		Bold(statusBold).Render(b.Status)
	if implicit, from := core.ImplicitStatus(b.ID); implicit != "" {
		statusCell += " " + ui.Muted.Render("↑"+implicit+" ("+from+")")
	}

	identity := tableBlock{band: true, left: []string{
		ui.Muted.Render("id:") + " " + padVisible(tint.Render(b.ID), bandIDWidth),
		ui.Muted.Render("type:") + " " + padVisible(tint.Render(b.Type), vocabularyWidth(cfg.TypeNames())),
		ui.Muted.Render("status:") + " " + statusCell,
	}}
	if b.Priority != "" {
		priorityColor := "gray"
		if pc := cfg.GetPriority(b.Priority); pc != nil {
			priorityColor = pc.Color
		}
		identity.right = ui.Muted.Render("priority:") + " " +
			lipgloss.NewStyle().Foreground(ui.ResolveColor(priorityColor)).Render(b.Priority)
	}
	blocks := []tableBlock{identity, {label: "title:", value: b.Title}}

	if len(b.Tags) > 0 {
		parts := make([]string, len(b.Tags))
		for i, t := range b.Tags {
			parts[i] = "#" + t
		}
		blocks = append(blocks, tableBlock{label: "tags:", value: strings.Join(parts, " ")})
	}

	if b.Parent != "" {
		text, trailer := describeRelated(b.Parent, cfg)
		blocks = append(blocks, tableBlock{label: "parent:", value: text, right: trailer})
	}
	for _, id := range b.BlockedBy {
		text, trailer := describeRelated(id, cfg)
		blocks = append(blocks, tableBlock{label: "blocked by:", value: text, right: trailer})
	}
	for _, id := range b.Blocking {
		text, trailer := describeRelated(id, cfg)
		blocks = append(blocks, tableBlock{label: "blocking:", value: text, right: trailer})
	}

	// Extra keys are front matter the schema does not name -- policy fields
	// like branch, topic or release. Sorted, because a map has no order and
	// a reader needs a stable one.
	if len(b.Extra) > 0 {
		keys := make([]string, 0, len(b.Extra))
		for k := range b.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			blocks = append(blocks, tableBlock{label: k + ":", value: formatExtraValue(b.Extra[k])})
		}
	}

	stamps := tableBlock{band: true}
	if b.CreatedAt != nil {
		stamps.left = append(stamps.left, ui.Muted.Render("created:")+" "+
			b.CreatedAt.Format("2006-01-02 15:04 UTC"))
	}
	if b.UpdatedAt != nil {
		stamps.left = append(stamps.left, ui.Muted.Render("updated:")+" "+
			b.UpdatedAt.Format("2006-01-02 15:04 UTC"))
	}
	if b.Order != "" {
		stamps.right = ui.Muted.Render("order:") + " " + b.Order
	}
	if len(stamps.left) > 0 || stamps.right != "" {
		blocks = append(blocks, stamps)
	}

	return blocks
}

// vocabularyWidth is the widest name in a config enum, which is what makes a
// band's columns a property of the configuration rather than of the bean
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
func describeRelated(id string, cfg *config.Config) (string, string) {
	resolver := &beangraph.CoreResolver{Core: core}
	related, err := resolver.Bean(context.Background(), id)
	if err != nil || related == nil {
		// Nothing to describe: the id becomes the value itself rather
		// than a trailer beside an empty cell.
		return id, ""
	}

	tint := lipgloss.NewStyle()
	if tc := cfg.GetType(related.Type); tc != nil {
		tint = tint.Bold(tc.Emphasis)
		if tc.Color != "" {
			tint = tint.Foreground(ui.ResolveColor(tc.Color))
		}
	}

	// The separator is a middle dot rather than spacing because wrapVisible
	// splits on fields, so a two-space gap survives only until the value
	// wraps and "epic show --table: ..." then reads as one run-on phrase.
	parts := []string{}
	if related.Type != "" {
		parts = append(parts, tint.Render(related.Type))
	}
	if related.Title != "" {
		parts = append(parts, related.Title)
	}
	return strings.Join(parts, " · "), ui.Muted.Render(id)
}
