// internal/ui/columns_test.go
package ui

import (
	"fmt"
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
	// Progress rows are in the sweep on purpose: Title is width minus every
	// other column's cost, so a sum built only from the fields NewColumns
	// just returned is an identity — it balances no matter what those fields
	// are, even if a whole column's cost silently stopped being subtracted.
	// wantProgress is computed here from the same inputs (progressBarWidth
	// plus the counter text), never read off c.ProgressWidth, so it stays
	// correct even if NewColumns forgets to reserve room for it — which is
	// exactly the prototype's overflow: a milestone row's progress column
	// pushed the line past the terminal because nothing had reserved its
	// width.
	rows := []Row{
		{Bean: &bean.Bean{ID: "beans-abcd", Title: strings.Repeat("x", 200), Type: "milestone"},
			Progress: &Progress{Done: 3, Total: 10}},
		{Bean: &bean.Bean{ID: "beans-abcd", Title: "short", Type: "task"}},
	}
	wantProgress := progressBarWidth + 2 + DisplayWidth(fmt.Sprintf("%d/%d", 3, 10)) // 6 + 2 + 4 = 12
	for w := 60; w <= 200; w++ {
		c := NewColumns(rows, w, true, config.Default())
		total := c.Indent + c.Type + c.Gap + c.ID + c.Gap + c.Title +
			c.Gap + c.Status + c.Gap + c.Prio + c.Gap + wantProgress
		if c.Tags > 0 {
			total += c.Gap + c.Tags
		}
		if total > w {
			t.Errorf("width %d: columns plus the reserved progress column sum to %d", w, total)
		}
	}
}

// TestFixedCostIsPinnedNotJustBalanced pins every column's width at one known
// fixture against numbers worked out independently of NewColumns, so a change
// to Gap or a tags width is caught rather than silently absorbed into a
// bigger or smaller Title. TestColumnsNeverExceedTheTerminal cannot do this:
// Title is defined as "whatever budget() leaves over", so any sum built from
// NewColumns's own output fields balances to the width by construction,
// whatever those fields happen to be — that identity is what let Gap going
// from 2 to 3, or the wide-tags width going from 24 to 28, leave every test
// in this file green.
//
// At width 200 with a 10-cell ID and tags showing, the budget affords every
// upgrade: Indent 0 (depth 0), Type 10 (long), ID 10, Status 11 (long),
// Prio 8 (long), Tags 24 (width >= 120), Gap 2, six gaps (before type, ID,
// title, status, prio, tags). These numbers were worked out by hand from the
// brief's constants, not read back from a run of NewColumns.
func TestFixedCostIsPinnedNotJustBalanced(t *testing.T) {
	rows := testRows(strings.Repeat("x", 200), "short")
	c := NewColumns(rows, 200, true, config.Default())

	wantIndent, wantGap, wantID, wantType, wantStatus, wantPrio, wantTags := 0, 2, 10, 10, 11, 8, 24
	if c.Indent != wantIndent {
		t.Errorf("Indent = %d, want %d", c.Indent, wantIndent)
	}
	if c.Gap != wantGap {
		t.Errorf("Gap = %d, want %d", c.Gap, wantGap)
	}
	if c.ID != wantID {
		t.Errorf("ID = %d, want %d", c.ID, wantID)
	}
	if c.Type != wantType || !c.LongType {
		t.Errorf("Type = %d (long=%v), want %d long", c.Type, c.LongType, wantType)
	}
	if c.Status != wantStatus || !c.LongStatus {
		t.Errorf("Status = %d (long=%v), want %d long", c.Status, c.LongStatus, wantStatus)
	}
	if c.Prio != wantPrio || !c.LongPrio {
		t.Errorf("Prio = %d (long=%v), want %d long", c.Prio, c.LongPrio, wantPrio)
	}
	if c.Tags != wantTags {
		t.Errorf("Tags = %d, want %d", c.Tags, wantTags)
	}

	// Six gaps: before type, before ID, before title, before status, before
	// prio, before tags. Title is what is left after every other pinned
	// number is subtracted from the terminal width — a literal number, not a
	// re-derivation of budget().
	wantFixed := wantIndent + wantType + wantGap + wantID + wantGap + wantGap +
		wantStatus + wantGap + wantPrio + wantGap + wantTags
	wantTitle := 200 - wantFixed
	if c.Title != wantTitle {
		t.Errorf("Title = %d, want %d (200 minus the independently pinned fixed cost of %d)",
			c.Title, wantTitle, wantFixed)
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
