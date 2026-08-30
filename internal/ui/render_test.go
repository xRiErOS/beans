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
