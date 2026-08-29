// internal/ui/columns_test.go
package ui

import (
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
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

func testRows(titles ...string) []Row {
	rows := make([]Row, 0, len(titles))
	for _, ti := range titles {
		rows = append(rows, Row{Bean: &bean.Bean{
			ID: "beans-abcd", Title: ti, Type: "task", Status: "todo", Priority: "normal",
		}})
	}
	return rows
}

func TestNarrowTerminalKeepsEveryAxisShort(t *testing.T) {
	c := NewColumns(testRows("a short title"), 80, true, config.Default())
	if c.LongStatus || c.LongType || c.LongPrio {
		t.Errorf("at 80 columns nothing should be long: type=%v status=%v prio=%v",
			c.LongType, c.LongStatus, c.LongPrio)
	}
}

func TestStatusIsBoughtBeforeType(t *testing.T) {
	// The buying order is status, then type, then priority. At a width that
	// affords exactly one upgrade, it must be status.
	rows := testRows(strings.Repeat("x", 200))
	for w := 80; w <= 200; w++ {
		c := NewColumns(rows, w, true, config.Default())
		if c.LongType && !c.LongStatus {
			t.Fatalf("at width %d type went long before status", w)
		}
		if c.LongPrio && !c.LongType {
			t.Fatalf("at width %d priority went long before type", w)
		}
	}
}

func TestTitleNeverDropsBelowTheFloorForAnUpgrade(t *testing.T) {
	rows := testRows(strings.Repeat("x", 200))
	for w := 80; w <= 200; w++ {
		c := NewColumns(rows, w, true, config.Default())
		if (c.LongStatus || c.LongType || c.LongPrio) && c.Title < minTitleWidth {
			t.Fatalf("width %d: bought a long form and left the title at %d", w, c.Title)
		}
	}
}

func TestColumnsNeverExceedTheTerminal(t *testing.T) {
	rows := testRows(strings.Repeat("x", 200), "short")
	for w := 60; w <= 200; w++ {
		c := NewColumns(rows, w, true, config.Default())
		total := c.Indent + c.Type + c.Gap + c.ID + c.Gap + c.Title +
			c.Gap + c.Status + c.Gap + c.Prio
		if c.Tags > 0 {
			total += c.Gap + c.Tags
		}
		if total > w {
			t.Errorf("width %d: columns sum to %d", w, total)
		}
	}
}

func TestTagsGiveWayBeforeTheTitleIsCrushed(t *testing.T) {
	c := NewColumns(testRows(strings.Repeat("x", 200)), 62, true, config.Default())
	if c.Tags != 0 {
		t.Errorf("at 62 columns the tags column should have been dropped, got %d", c.Tags)
	}
}

func TestIndentFollowsTheDeepestRow(t *testing.T) {
	rows := []Row{
		{Bean: &bean.Bean{ID: "a", Title: "root", Type: "epic"}, Depth: 0},
		{Bean: &bean.Bean{ID: "b", Title: "kid", Type: "task"}, Depth: 1, AncestorsLast: []bool{true}},
		{Bean: &bean.Bean{ID: "c", Title: "gk", Type: "task"}, Depth: 2, AncestorsLast: []bool{true, true}},
	}
	c := NewColumns(rows, 110, false, config.Default())
	if c.Indent != 6 {
		t.Errorf("Indent = %d, want 6 for a depth-2 tree", c.Indent)
	}
}

func TestFlatRowsNeedNoIndent(t *testing.T) {
	c := NewColumns(FlatRows([]*bean.Bean{{ID: "a", Title: "x", Type: "task"}}), 110, false, config.Default())
	if c.Indent != 0 {
		t.Errorf("Indent = %d, want 0 for flat rows", c.Indent)
	}
}

func TestCounterWidthIsMeasuredNotAssumed(t *testing.T) {
	// A real store reached 131/139 and burst a fixed five-cell counter.
	//
	// The n/m counter itself is digits and a slash — always one byte per
	// cell, so len() and DisplayWidth() can never disagree on it; no fixture
	// could make that half of this test catch a DisplayWidth-for-len swap.
	// The same row-scanning loop also measures the bean ID, and IDs are free
	// text (a slug carrying an umlaut is realistic in this codebase), so
	// "beans-über" here is what actually exercises DisplayWidth: 11 bytes,
	// 10 cells, because ü is two UTF-8 bytes but one terminal cell.
	rows := []Row{
		{Bean: &bean.Bean{ID: "beans-über", Title: "m", Type: "milestone"}, Progress: &Progress{Done: 131, Total: 139}},
		{Bean: &bean.Bean{ID: "b", Title: "n", Type: "milestone"}, Progress: &Progress{Done: 0, Total: 5}},
	}
	c := NewColumns(rows, 110, false, config.Default())
	if c.ID != 10 {
		t.Errorf("ID = %d, want 10 cells for \"beans-über\" (11 bytes, 10 cells) — len() would report 11", c.ID)
	}
	if c.Counter != 7 {
		t.Errorf("Counter = %d, want 7 for \"131/139\"", c.Counter)
	}
	if c.ProgressWidth != 6+2+7 {
		t.Errorf("ProgressWidth = %d, want bar(6) + gap(2) + counter(7)", c.ProgressWidth)
	}
}

func TestTypeTextFollowsTheLongFlag(t *testing.T) {
	b := &bean.Bean{Type: "milestone"}
	short := Columns{LongType: false}
	if got := short.TypeText(b); got != "M" {
		t.Errorf("short type = %q, want %q", got, "M")
	}
	long := Columns{LongType: true}
	if got := long.TypeText(b); got != "milestone" {
		t.Errorf("long type = %q, want %q", got, "milestone")
	}
}

func TestPrioTextHidesNormal(t *testing.T) {
	c := Columns{LongPrio: true}
	if got := c.PrioText(&bean.Bean{Priority: "normal"}); got != "" {
		t.Errorf("normal priority = %q, want empty", got)
	}
	if got := c.PrioText(&bean.Bean{Priority: "high"}); got != "high" {
		t.Errorf("high priority = %q, want %q", got, "high")
	}
	short := Columns{LongPrio: false}
	if got := short.PrioText(&bean.Bean{Priority: "critical"}); got != "‼" {
		t.Errorf("short critical = %q, want %q", got, "‼")
	}
}
