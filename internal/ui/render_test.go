// internal/ui/render_test.go
package ui

import (
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

func TestTaskCarriesNoColour(t *testing.T) {
	s := styleFor(&bean.Bean{Type: "task"}, config.Default())
	if s.hasColor {
		t.Error("task should stay in the terminal's own text colour")
	}
	if s.bold {
		t.Error("task should not be emphasised")
	}
}

func TestContainersAreEmphasised(t *testing.T) {
	for _, typ := range []string{"milestone", "epic"} {
		if !styleFor(&bean.Bean{Type: typ}, config.Default()).bold {
			t.Errorf("%s should be emphasised", typ)
		}
	}
	for _, typ := range []string{"feature", "bug"} {
		if styleFor(&bean.Bean{Type: typ}, config.Default()).bold {
			t.Errorf("%s should not be emphasised", typ)
		}
	}
}

func TestEveryColouredTypeTintsItsTitle(t *testing.T) {
	// The title is the same thing as the type word; feature and bug went
	// uncoloured once because tinting was tied to emphasis.
	for _, typ := range []string{"milestone", "epic", "feature", "bug"} {
		s := styleFor(&bean.Bean{Type: typ}, config.Default())
		if !s.hasColor {
			t.Errorf("%s title should carry the type colour", typ)
		}
	}
}

func TestTagCellShowsTheFirstTagEvenWhenItMustBeCut(t *testing.T) {
	// A column holding nothing but "+3" tells the reader nothing.
	got := stripANSI(tagCell([]string{"a-very-long-tag-indeed", "b", "c"}, 12))
	if !strings.HasPrefix(got, "#a-very") {
		t.Errorf("tag cell = %q, want it to start with the elided first tag", got)
	}
}

func TestTagCellMarksTheRemainder(t *testing.T) {
	got := stripANSI(tagCell([]string{"one", "two", "three", "four"}, 10))
	if !strings.Contains(got, "+") {
		t.Errorf("tag cell = %q, want a +N marker", got)
	}
	if DisplayWidth(got) > 10 {
		t.Errorf("tag cell = %q is %d cells, want at most 10", got, DisplayWidth(got))
	}
}

func TestTagCellNeverExceedsItsWidth(t *testing.T) {
	sets := [][]string{
		{"short"},
		{"one", "two"},
		{"a-really-quite-long-tag-name"},
		{"note-intern", "slug-tailwind-upgrade"},
	}
	for _, tags := range sets {
		for w := 1; w <= 30; w++ {
			if got := DisplayWidth(stripANSI(tagCell(tags, w))); got > w {
				t.Errorf("tagCell(%v, %d) is %d cells wide", tags, w, got)
			}
		}
	}
}

func TestProgressCellShowsBarAndCounter(t *testing.T) {
	c := Columns{ProgressWidth: 15, Counter: 7}
	got := stripANSI(progressCell(&Progress{Done: 131, Total: 139}, c))
	if !strings.Contains(got, "131/139") {
		t.Errorf("progress cell = %q, want the counter", got)
	}
	if !strings.Contains(got, "█") {
		t.Errorf("progress cell = %q, want a filled bar", got)
	}
}

func TestProgressCellHandlesZeroTotal(t *testing.T) {
	c := Columns{ProgressWidth: 13, Counter: 5}
	got := stripANSI(progressCell(&Progress{Done: 0, Total: 0}, c))
	if strings.Contains(got, "█") {
		t.Errorf("progress cell = %q, want an empty bar for 0/0", got)
	}
}

// The tests above never assert on colour itself — they strip ANSI and check
// content, so they cannot be fooled by the no-tty vacuous-colour trap, but
// they also never prove any colour is actually applied. The brief specifies
// no test for statusCell or prioCell at all, and cellStyle.render is only
// exercised indirectly through styleFor's returned fields. These companions
// close that gap: withTrueColor forces real ANSI so a missing- or
// wrong-colour bug shows up rather than being silently absorbed by the
// no-tty default.

func TestCellStyleRenderCarriesColourOnlyWhenResolved(t *testing.T) {
	withTrueColor(t)
	// feature is coloured but not emphasised (unlike epic/milestone, which
	// are both) — isolating the two lets this test catch a bug that drops
	// the foreground while bold still leaves its own escape code behind.
	coloured := styleFor(&bean.Bean{Type: "feature"}, config.Default())
	plain := styleFor(&bean.Bean{Type: "task"}, config.Default())

	got := coloured.render("feature-42")
	if stripANSI(got) == got {
		t.Errorf("coloured type rendered with no ANSI codes: %q", got)
	}
	if want := "task-1"; plain.render(want) != want {
		t.Errorf("uncoloured, unemphasised style should pass text through unchanged, got %q", plain.render(want))
	}
}

func TestStatusCellColoursVaryByStatus(t *testing.T) {
	withTrueColor(t)
	cfg := config.Default()
	c := Columns{Status: 11}
	codeFor := func(status string) string {
		out := statusCell(&bean.Bean{Status: status}, c, cfg)
		codes := ansiEscape.FindAllString(out, -1)
		if len(codes) == 0 {
			t.Fatalf("status cell for %q carries no ANSI colour under a true-colour profile: %q", status, out)
		}
		return codes[0]
	}
	if codeFor("todo") == codeFor("in-progress") {
		t.Error("todo and in-progress should not render with the same colour")
	}
}

func TestPrioCellHidesNormalPriority(t *testing.T) {
	// The density decision from the design: normal repeats on nearly every
	// row in the table, so it earns no ink there (unlike the detail view).
	c := Columns{Prio: 3}
	got := prioCell(&bean.Bean{Priority: "normal"}, c, config.Default())
	if stripANSI(got) != "   " {
		t.Errorf("normal priority = %q, want a blank cell", stripANSI(got))
	}
}

func TestPrioCellColoursVaryByPriority(t *testing.T) {
	withTrueColor(t)
	cfg := config.Default()
	c := Columns{Prio: 3}
	codeFor := func(pri string) string {
		out := prioCell(&bean.Bean{Priority: pri}, c, cfg)
		codes := ansiEscape.FindAllString(out, -1)
		if len(codes) == 0 {
			t.Fatalf("prio cell for %q carries no ANSI colour under a true-colour profile: %q", pri, out)
		}
		return codes[0]
	}
	// critical and high are both bold, so a mismatch here can only come from
	// colour, not weight — low would also differ from critical on bold alone
	// and mask a hardcoded-colour bug the way it did during mutation testing.
	if codeFor("critical") == codeFor("high") {
		t.Error("critical and high should not render with the same colour")
	}
}

func demoRows() []Row {
	m := &bean.Bean{ID: "beans-fexy", Title: "0.5 Output alignment", Type: "milestone", Status: "in-progress", Priority: "normal", Tags: []string{"release"}}
	e := &bean.Bean{ID: "beans-9m0d", Title: "Shared presentation vocabulary", Type: "epic", Status: "in-progress", Priority: "normal", Tags: []string{"ui", "accepted"}}
	t1 := &bean.Bean{ID: "beans-9zpz", Title: "Extract the colour resolution into one place", Type: "task", Status: "in-progress", Priority: "high"}
	b1 := &bean.Bean{ID: "beans-wa9y", Title: "Status column drifts by one cell when tags are on", Type: "bug", Status: "todo", Priority: "critical", Tags: []string{"regression"}}
	return []Row{
		{Bean: m, Depth: 0, IsLast: false},
		{Bean: e, Depth: 1, AncestorsLast: []bool{false}, IsLast: false},
		{Bean: t1, Depth: 2, AncestorsLast: []bool{false, false}, IsLast: false},
		{Bean: b1, Depth: 2, AncestorsLast: []bool{false, false}, IsLast: true},
	}
}

func TestNoLineExceedsTheWidthInEitherForm(t *testing.T) {
	// The original sweep started at 70 and never reached the tree's title-body
	// floor, which only misfires below width 47 (tags off) or 67 (tags on)
	// for demoRows() — a Task-10-shaped gap: the right property, swept in a
	// range the guarded code never runs in. 40 sits inside that window (and
	// is the exact value traced by hand in review) without falling into the
	// narrower width below 39 where even the fix legitimately still overflows
	// — demoRows()'s longest id is 10 cells wide, and minRenderableTitle is
	// the documented last resort for exactly that case (see columns.go): a
	// terminal too narrow for id+status+priority to fit at all is not this
	// bug, it is the accepted floor doing its job. 20 and 30 were tried first
	// and both land in that legitimately-still-overflowing zone for *both*
	// forms, which is why they are not in this list.
	//
	// Legend() (columns.go) is out of scope for this file and is not
	// width-aware at all — its longest line ("status ...") is a fixed 68
	// cells regardless of the width argument, so it overflows any width below
	// that on its own, in both forms, independent of anything render.go
	// decides. That is a real, separate violation of the same "no line
	// exceeds width" property this test polices, but it belongs to
	// columns.go, which this task does not touch — see the report. To keep
	// this test targeted at what render.go itself is responsible for (the
	// row/tree layout), the legend block — everything from the first blank
	// line on, which is exactly where Columns.Legend's own output begins — is
	// excluded from the narrow-width check below.
	for _, form := range []Form{FormTable, FormTree} {
		for _, w := range []int{40, 70, 80, 100, 110, 130, 160} {
			for _, tags := range []bool{false, true} {
				out := Render(demoRows(), form, "Demo", w, tags, config.Default())
				for i, line := range strings.Split(out, "\n") {
					if line == "" {
						break // start of the legend block; not this file's responsibility
					}
					if got := DisplayWidth(stripANSI(line)); got > w {
						t.Errorf("form=%s width=%d tags=%v line %d is %d cells:\n%s",
							form, w, tags, i, got, stripANSI(line))
					}
				}
			}
		}
	}
}

func TestTableFormHasAHeaderAndNoTreeCharacters(t *testing.T) {
	out := stripANSI(Render(demoRows(), FormTable, "Demo", 110, false, config.Default()))
	if !strings.Contains(out, "TITLE") {
		t.Error("table form must carry a header")
	}
	for _, glyph := range []string{"├", "└", "│"} {
		if strings.Contains(out, glyph) {
			t.Errorf("table form must be flat, found %q", glyph)
		}
	}
}

// TestTableFormFlattensBeforeMeasuringColumns guards render.go:185's flat
// argument to NewColumns. demoRows() carries real depth (0, 1, 2, 2), and
// the flat-vs-tree distinction has no tree glyphs to give it away in table
// form (renderTable never calls Connector/Stem) — the only symptom of
// passing rows instead of flat is that NewColumns' Indent (3*maxDepth) is
// folded into the header's TYPE column but not into the row's own type
// cell, so the header silently drifts out of alignment with its column.
func TestTableFormFlattensBeforeMeasuringColumns(t *testing.T) {
	out := stripANSI(Render(demoRows(), FormTable, "Demo", 110, false, config.Default()))
	lines := strings.Split(out, "\n")
	if len(lines) < 4 {
		t.Fatalf("expected at least a title, rule, header and one row, got %d lines:\n%s", len(lines), out)
	}
	header, firstRow := lines[2], lines[3]
	headerID := strings.Index(header, "ID")
	rowID := strings.Index(firstRow, "beans-fexy")
	if headerID < 0 || rowID < 0 {
		t.Fatalf("could not locate ID column in header %q or row %q", header, firstRow)
	}
	if headerID != rowID {
		t.Errorf("header's ID column starts at %d, first row's id at %d — the tree depth carried by demoRows() must be dropped before NewColumns measures, not just hidden by the absence of glyphs", headerID, rowID)
	}
}

func TestTreeFormHasTreeCharactersAndNoHeader(t *testing.T) {
	out := stripANSI(Render(demoRows(), FormTree, "Demo", 110, false, config.Default()))
	if strings.Contains(out, "TITLE") {
		t.Error("tree form promises no columns and must carry no header")
	}
	if !strings.Contains(out, "├─ ") {
		t.Error("tree form must draw its connectors")
	}
}

func TestTreeFormRunsTheConnectorIntoTheTypeWord(t *testing.T) {
	// The connector must not end two columns short of the word it connects.
	out := stripANSI(Render(demoRows(), FormTree, "Demo", 110, false, config.Default()))
	if !strings.Contains(out, "├─ epic") && !strings.Contains(out, "├─ E") {
		t.Errorf("connector does not lead into the type word:\n%s", out)
	}
}

func TestWrappedTitlesKeepTheVerticalLines(t *testing.T) {
	rows := []Row{
		{Bean: &bean.Bean{ID: "a", Title: "root", Type: "epic", Status: "todo"}, Depth: 0, IsLast: false},
		{Bean: &bean.Bean{ID: "b", Title: strings.Repeat("word ", 30), Type: "task", Status: "todo"},
			Depth: 1, AncestorsLast: []bool{false}, IsLast: false},
		{Bean: &bean.Bean{ID: "c", Title: "after", Type: "task", Status: "todo"},
			Depth: 1, AncestorsLast: []bool{false}, IsLast: true},
	}
	out := stripANSI(Render(rows, FormTree, "Demo", 80, false, config.Default()))
	lines := strings.Split(out, "\n")
	found := false
	for i, l := range lines {
		if strings.Contains(l, "├─ ") && i+1 < len(lines) {
			if strings.Contains(lines[i+1], "│") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("a wrapped row severed its branch:\n%s", out)
	}
}

// TestColourSurvivesAWrappedTitleInBothForms guards the prototype regression
// the user caught by eye: only the first line of a wrapped title took the
// type's colour, continuation lines fell back to the terminal's default. All
// other colour-bearing tests in this file strip ANSI first and would not
// notice this; this one deliberately does not.
func TestColourSurvivesAWrappedTitleInBothForms(t *testing.T) {
	withTrueColor(t)
	rows := []Row{
		{Bean: &bean.Bean{ID: "beans-x", Title: strings.Repeat("word ", 30), Type: "epic", Status: "todo"}, Depth: 0, IsLast: true},
	}
	colourBefore := func(t *testing.T, line, target string) string {
		t.Helper()
		idx := strings.Index(line, target)
		if idx < 0 {
			t.Fatalf("could not find %q in line %q", target, line)
		}
		matches := ansiEscape.FindAllString(line[:idx], -1)
		if len(matches) == 0 {
			t.Fatalf("line %q carries no ANSI colour before %q under a true-colour profile", line, target)
		}
		return matches[len(matches)-1]
	}
	for _, form := range []Form{FormTable, FormTree} {
		out := Render(rows, form, "Demo", 40, false, config.Default())
		lines := strings.Split(out, "\n")
		firstIdx := -1
		for i, l := range lines {
			if strings.Contains(l, "beans-x") {
				firstIdx = i
				break
			}
		}
		if firstIdx < 0 || firstIdx+1 >= len(lines) {
			t.Fatalf("form=%s: expected a wrapped title with at least one continuation line:\n%s", form, out)
		}
		first, cont := lines[firstIdx], lines[firstIdx+1]
		firstColour := colourBefore(t, first, "word")
		contColour := colourBefore(t, cont, "word")
		if firstColour != contColour {
			t.Errorf("form=%s: title colour changed across the wrap: first line %q, continuation %q", form, firstColour, contColour)
		}
	}
}

func TestSectionHeadingsAppear(t *testing.T) {
	rows := demoRows()
	rows[2].Section = "No Milestone"
	out := stripANSI(Render(rows, FormTree, "Roadmap", 110, false, config.Default()))
	if !strings.Contains(out, "No Milestone") {
		t.Error("section heading missing")
	}
}

func TestParseForm(t *testing.T) {
	for _, s := range []string{"table", "tree"} {
		if _, ok := ParseForm(s); !ok {
			t.Errorf("ParseForm(%q) rejected a valid form", s)
		}
	}
	if _, ok := ParseForm("grid"); ok {
		t.Error(`ParseForm("grid") accepted an unknown form`)
	}
}
