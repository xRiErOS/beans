// internal/ui/columns.go
package ui

import (
	"fmt"
	"strings"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// Progress is a milestone's descendant completion, shown as a bar plus n/m.
type Progress struct {
	Done  int
	Total int
}

// Row is one line of output: a bean plus the tree shape needed to draw it.
//
// AncestorsLast records, for each ancestor, whether that ancestor was the last
// child at its level. That is what decides whether a vertical line continues
// past this row or stops. Index 0 corresponds to the root and is never
// consulted by Connector or Stem — roots draw nothing, so the stem starts one
// level in.
type Row struct {
	Bean          *bean.Bean
	Depth         int
	AncestorsLast []bool
	IsLast        bool
	// Section, when set, prints as a heading above this row — the roadmap's
	// "No Milestone" bucket is the only current use.
	Section  string
	Progress *Progress
}

// Connector is the tree prefix that leads into this row's type word.
// Roots draw nothing, so the stem starts one level in.
func (r Row) Connector() string {
	if r.Depth == 0 {
		return ""
	}
	var sb strings.Builder
	for _, last := range r.AncestorsLast[1:] {
		if last {
			sb.WriteString("   ")
		} else {
			sb.WriteString("│  ")
		}
	}
	if r.IsLast {
		sb.WriteString("└─ ")
	} else {
		sb.WriteString("├─ ")
	}
	return sb.String()
}

// Stem is what a wrapped row's continuation lines carry in place of the
// connector: the ancestors' verticals, plus this row's own vertical while
// siblings still follow it. Same width as Connector by construction.
func (r Row) Stem() string {
	if r.Depth == 0 {
		return ""
	}
	var sb strings.Builder
	for _, last := range r.AncestorsLast[1:] {
		if last {
			sb.WriteString("   ")
		} else {
			sb.WriteString("│  ")
		}
	}
	if r.IsLast {
		sb.WriteString("   ")
	} else {
		sb.WriteString("│  ")
	}
	return sb.String()
}

// RowsFromFlatItems adapts the existing tree flattening to the row model.
//
// ancestry is truncated back to the current item's depth before use: a flat
// list returning to a shallower depth (e.g. after a grandchild, back to a
// second-level sibling) must not carry forward an ancestor from the deeper
// branch it just left.
func RowsFromFlatItems(items []FlatItem) []Row {
	rows := make([]Row, 0, len(items))
	ancestry := []bool{}
	for _, it := range items {
		if it.Depth < len(ancestry) {
			ancestry = ancestry[:it.Depth]
		}
		for len(ancestry) < it.Depth {
			ancestry = append(ancestry, true)
		}
		anc := make([]bool, len(ancestry))
		copy(anc, ancestry)
		rows = append(rows, Row{
			Bean:          it.Bean,
			Depth:         it.Depth,
			AncestorsLast: anc,
			IsLast:        it.IsLast,
		})
		ancestry = append(ancestry, it.IsLast)
	}
	return rows
}

// FlatRows puts every bean at depth 0, which is what the table form needs:
// dropping the tree rather than hiding it is what makes the rows sortable.
func FlatRows(beans []*bean.Bean) []Row {
	rows := make([]Row, 0, len(beans))
	for _, b := range beans {
		rows = append(rows, Row{Bean: b, Depth: 0, IsLast: true})
	}
	return rows
}

// minTitleWidth gates the long-form upgrades: an upgrade is only bought while
// the title keeps at least this many cells. It is a purchase criterion, never
// an actual width — using it as one is how a row grew past its terminal.
const minTitleWidth = 45

// progressBarWidth is the filled/empty bar in the milestone progress column.
const progressBarWidth = 6

// tagsCrushWidth is the floor below which the tags column is dropped rather
// than left to crush the title. It is well below minTitleWidth on purpose:
// giving up tags costs the reader far less than giving up a long-form column.
const tagsCrushWidth = 25

// Columns holds the resolved widths for one rendering.
//
// Short and long forms are decided here and nowhere else. Cell renderers ask
// this type for their text, so the header can never promise a form the cells
// do not deliver.
type Columns struct {
	Width  int
	Gap    int
	Indent int // tree depth times three; folded into the type cell, not its own column

	Type   int
	ID     int
	Title  int
	Status int
	Prio   int
	Tags   int

	// ProgressWidth is 0 unless rows carry Progress. Named for the width, not
	// the thing, so it does not read like the Progress type beside it.
	ProgressWidth int
	Counter       int // width of "n/m", measured from the data

	LongType   bool
	LongStatus bool
	LongPrio   bool
}

// NewColumns resolves every width for the given rows at the given terminal
// width. It starts compact and buys long forms in a fixed order — status,
// then type, then priority — while the title can afford them.
func NewColumns(rows []Row, width int, showTags bool, cfg *config.Config) Columns {
	c := Columns{Width: width, Gap: 2, Type: 1, Status: 1, Prio: 1}

	maxDepth := 0
	idWidth := 2
	counter := 0
	hasProgress := false
	for _, r := range rows {
		if r.Depth > maxDepth {
			maxDepth = r.Depth
		}
		if w := DisplayWidth(r.Bean.ID); w > idWidth {
			idWidth = w
		}
		if r.Progress != nil {
			hasProgress = true
			if w := DisplayWidth(fmt.Sprintf("%d/%d", r.Progress.Done, r.Progress.Total)); w > counter {
				counter = w
			}
		}
	}
	c.Indent = 3 * maxDepth
	c.ID = idWidth
	if hasProgress {
		c.Counter = counter
		c.ProgressWidth = progressBarWidth + 2 + counter
	}
	if showTags {
		if width >= 120 {
			c.Tags = 24
		} else {
			c.Tags = 18
		}
	}

	budget := func() int {
		fixed := c.Indent + c.Type + c.Gap + c.ID + c.Gap + c.Gap + c.Status + c.Gap + c.Prio
		if c.Tags > 0 {
			fixed += c.Gap + c.Tags
		}
		if c.ProgressWidth > 0 {
			fixed += c.Gap + c.ProgressWidth
		}
		return width - fixed
	}

	// Buying order: status first, because T and D and C are the codes a reader
	// stumbles over most; then type; then priority, whose symbols read fine.
	upgrades := []struct {
		field *int
		flag  *bool
		want  int
	}{
		{&c.Status, &c.LongStatus, 11},
		{&c.Type, &c.LongType, 10},
		{&c.Prio, &c.LongPrio, 8},
	}
	for _, u := range upgrades {
		cost := u.want - *u.field
		if budget()-cost < minTitleWidth {
			// The order is fixed: once one upgrade cannot be afforded, a
			// later, cheaper upgrade must not be bought either — otherwise
			// priority could go long while type and status stay short.
			break
		}
		*u.field = u.want
		*u.flag = true
	}

	c.Title = budget()
	if c.Title < tagsCrushWidth && c.Tags > 0 {
		// Tags give way before the title is crushed: the title is content, the
		// tags are metadata. This floor is deliberately below minTitleWidth —
		// dropping tags is a much cheaper trade than denying a long-form
		// upgrade, so it kicks in earlier.
		c.Tags = 0
		c.Title = budget()
	}
	if c.Title < 12 {
		c.Title = 12
	}
	return c
}

// TypeText renders the type in the form this layout decided on.
func (c Columns) TypeText(b *bean.Bean) string {
	if c.LongType {
		return b.Type
	}
	return ShortType(b.Type)
}

// StatusText renders the status in the form this layout decided on.
func (c Columns) StatusText(b *bean.Bean) string {
	if c.LongStatus {
		return b.Status
	}
	return ShortStatus(b.Status)
}

// PrioText renders the priority, or "" for normal — an unremarkable priority
// earns no ink.
func (c Columns) PrioText(b *bean.Bean) string {
	if b.Priority == "" || b.Priority == "normal" {
		return ""
	}
	if c.LongPrio {
		return b.Priority
	}
	return GetPrioritySymbol(b.Priority)
}
