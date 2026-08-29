// internal/ui/columns.go
package ui

import (
	"strings"

	"github.com/hmans/beans/pkg/bean"
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
