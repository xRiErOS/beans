// internal/ui/columns_test.go
package ui

import (
	"testing"

	"github.com/hmans/beans/pkg/bean"
)

func TestConnectorIsEmptyAtRoot(t *testing.T) {
	r := Row{Depth: 0}
	if got := r.Connector(); got != "" {
		t.Errorf("Connector() = %q, want empty at depth 0", got)
	}
}

func TestConnectorShapes(t *testing.T) {
	cases := []struct {
		name          string
		ancestorsLast []bool
		isLast        bool
		want          string
	}{
		{"first level, has siblings", []bool{true}, false, "├─ "},
		{"first level, last child", []bool{true}, true, "└─ "},
		{"second level under a parent with siblings", []bool{true, false}, false, "│  ├─ "},
		{"second level under a last child", []bool{true, true}, true, "   └─ "},
	}
	for _, c := range cases {
		r := Row{Depth: len(c.ancestorsLast), AncestorsLast: c.ancestorsLast, IsLast: c.isLast}
		if got := r.Connector(); got != c.want {
			t.Errorf("%s: Connector() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestStemContinuesOnlyWhileSiblingsFollow(t *testing.T) {
	withSiblings := Row{Depth: 1, AncestorsLast: []bool{true}, IsLast: false}
	if got := withSiblings.Stem(); got != "│  " {
		t.Errorf("Stem() = %q, want %q for a row with siblings after it", got, "│  ")
	}

	last := Row{Depth: 1, AncestorsLast: []bool{true}, IsLast: true}
	if got := last.Stem(); got != "   " {
		t.Errorf("Stem() = %q, want three spaces for a last child", got)
	}
}

func TestStemAndConnectorHaveTheSameWidth(t *testing.T) {
	// A continuation line must start exactly under its own title, so the stem
	// has to occupy the cells the connector occupied.
	r := Row{Depth: 2, AncestorsLast: []bool{true, false}, IsLast: false}
	if DisplayWidth(r.Stem()) != DisplayWidth(r.Connector()) {
		t.Errorf("stem %d cells, connector %d cells — must match",
			DisplayWidth(r.Stem()), DisplayWidth(r.Connector()))
	}
}

// TestRowsFromFlatItemsResetsAncestryOnShallowerDepth guards the ancestry
// stack against carrying a stale entry forward when a flat list returns to a
// shallower depth. Without the truncation, a grandchild's ancestry would leak
// into its uncle's row.
func TestRowsFromFlatItemsResetsAncestryOnShallowerDepth(t *testing.T) {
	items := []FlatItem{
		{Depth: 0, IsLast: false}, // root, has a sibling root after it
		{Depth: 1, IsLast: false}, // root's first child, has a sibling
		{Depth: 2, IsLast: true},  // grandchild, last at its level
		{Depth: 1, IsLast: true},  // back to depth 1: root's second (last) child
	}
	rows := RowsFromFlatItems(items)

	last := rows[3]
	if len(last.AncestorsLast) != 1 {
		t.Fatalf("returning to depth 1 kept a stale ancestor: AncestorsLast = %v, want length 1",
			last.AncestorsLast)
	}
	if last.AncestorsLast[0] != false {
		t.Errorf("AncestorsLast[0] = %v, want %v (the root's own IsLast)", last.AncestorsLast[0], false)
	}
}

// TestFlatRowsAreAllDepthZero pins down the table form's contract: no tree
// shape survives, every bean lands at depth 0 in its original order.
func TestFlatRowsAreAllDepthZero(t *testing.T) {
	beans := []*bean.Bean{{ID: "a"}, {ID: "b"}}
	rows := FlatRows(beans)

	if len(rows) != 2 {
		t.Fatalf("FlatRows returned %d rows, want 2", len(rows))
	}
	for i, r := range rows {
		if r.Depth != 0 {
			t.Errorf("row %d: Depth = %d, want 0", i, r.Depth)
		}
		if !r.IsLast {
			t.Errorf("row %d: IsLast = %v, want true", i, r.IsLast)
		}
		if r.Bean != beans[i] {
			t.Errorf("row %d: Bean pointer mismatch", i)
		}
	}
}
