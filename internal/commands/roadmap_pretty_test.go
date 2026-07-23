package commands

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hmans/beans/pkg/bean"
)

func TestRoadmapShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"prefixed", "beans-tquh", "tquh"},
		{"multi hyphen prefix", "lean-stack-ewig", "ewig"},
		{"bare", "mf38", "mf38"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roadmapShortID(tt.id); got != tt.want {
				t.Errorf("roadmapShortID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestRoadmapRightBlock(t *testing.T) {
	tests := []struct {
		name     string
		b        *bean.Bean
		showPrio bool
		want     string
	}{
		{
			name:     "high priority shown",
			b:        &bean.Bean{ID: "beans-dg21", Status: "todo", Priority: "high"},
			showPrio: true,
			want:     "    high  todo         dg21",
		},
		{
			name:     "normal priority hidden",
			b:        &bean.Bean{ID: "beans-mf38", Status: "in-progress", Priority: "normal"},
			showPrio: true,
			want:     "          in-progress  mf38",
		},
		{
			name:     "priority suppressed for containers",
			b:        &bean.Bean{ID: "beans-ewig", Status: "todo", Priority: "critical"},
			showPrio: false,
			want:     "          todo         ewig",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roadmapRightBlock(tt.b, tt.showPrio)
			if got != tt.want {
				t.Errorf("roadmapRightBlock() =\n%q\nwant\n%q", got, tt.want)
			}
			if utf8.RuneCountInString(got) != roadmapRightW {
				t.Errorf("roadmapRightBlock() width = %d, want %d", utf8.RuneCountInString(got), roadmapRightW)
			}
		})
	}
}

func TestRoadmapWrapTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		width int
		want  []string
	}{
		{"fits", "Checkout Flow", 34, []string{"Checkout Flow"}},
		{"empty stays one line", "", 34, []string{""}},
		{
			name:  "wraps on word boundary",
			title: "Refactor payment reconciliation ledger to support multi-currency settlement",
			width: 34,
			want: []string{
				"Refactor payment reconciliation",
				"ledger to support multi-currency",
				"settlement",
			},
		},
		{
			name:  "hard-breaks an overlong word",
			title: "Supercalifragilisticexpialidocious",
			width: 10,
			want:  []string{"Supercalif", "ragilistic", "expialidoc", "ious"},
		},
		{"umlauts count as one cell", "Prüfung ändern", 8, []string{"Prüfung", "ändern"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roadmapWrapTitle(tt.title, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("roadmapWrapTitle() = %q (%d lines), want %q (%d lines)",
					got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestRoadmapWrapTitleRuneVsByteWidth pins D16 (utf8.RuneCountInString, not
// len()) inside the word-boundary-wrap decision of roadmapWrapTitle itself.
// "ab é": rune-sum of "ab" + space + "é" is 2+1+1 = 4 == width, so the words
// fit onto one line under rune counting. Byte-sum is 2+1+2 = 5 > width,
// because "é" is a 2-byte, 1-rune UTF-8 sequence — if the decision used
// len() it would wrap one word too early ("ab", "é" on separate lines).
// Both words individually stay within the width in both rune and byte
// counting, so the hard-break loop is never entered and this cannot panic
// regardless of which counting the mutation under test uses.
// Verified independently via `command python3` (rune sum 4, byte sum 5).
func TestRoadmapWrapTitleRuneVsByteWidth(t *testing.T) {
	got := roadmapWrapTitle("ab é", 4)
	want := []string{"ab é"}
	if len(got) != len(want) {
		t.Fatalf("roadmapWrapTitle(%q, 4) = %q (%d lines), want %q (%d lines)",
			"ab é", got, len(got), want, len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRoadmapLine(t *testing.T) {
	epic := &bean.Bean{ID: "beans-tquh", Title: "Checkout Flow", Type: "epic",
		Status: "in-progress", Priority: "normal"}

	got := roadmapLine("  ▸ Epic", epic, false, 80)
	// Prefix "  ▸ Epic" is 8 runes, so 9 spaces pad it to column 17.
	want := "  ▸ Epic         Checkout Flow" +
		strings.Repeat(" ", 23) +
		"          in-progress  tquh"
	if got != want {
		t.Errorf("roadmapLine() =\n%q\nwant\n%q", got, want)
	}
	if utf8.RuneCountInString(got) != 80 {
		t.Errorf("line width = %d, want 80", utf8.RuneCountInString(got))
	}
	// Title must start at column 17 (rune index), glyph counts as one cell.
	runes := []rune(got)
	if string(runes[roadmapTitleCol:roadmapTitleCol+len("Checkout")]) != "Checkout" {
		t.Errorf("title does not start at column %d: %q", roadmapTitleCol, got)
	}
}

func TestRoadmapLineWrapsWithHangingIndent(t *testing.T) {
	long := &bean.Bean{
		ID:       "beans-uswm",
		Title:    "Refactor payment reconciliation ledger to support multi-currency settlement",
		Type:     "task",
		Status:   "todo",
		Priority: "high",
	}
	got := roadmapLine("    - task", long, true, 80)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d:\n%s", len(lines), got)
	}
	// Attributes only on line 1 (D07).
	if !strings.HasSuffix(lines[0], "    high  todo         uswm") {
		t.Errorf("line 1 missing right block: %q", lines[0])
	}
	// Continuation lines: hanging indent, no attributes.
	for i, l := range lines[1:] {
		if !strings.HasPrefix(l, strings.Repeat(" ", roadmapTitleCol)) {
			t.Errorf("continuation %d not indented to %d: %q", i, roadmapTitleCol, l)
		}
		if strings.Contains(l, "uswm") {
			t.Errorf("continuation %d carries attributes: %q", i, l)
		}
	}
}

func TestRoadmapLineOverlongPrefix(t *testing.T) {
	// D17: a custom type longer than the raster gets exactly one space.
	b := &bean.Bean{ID: "beans-zz99", Title: "Titel", Type: "verylongcustomtype",
		Status: "todo", Priority: "normal"}
	got := roadmapLine("      - verylongcustomtype", b, true, 80)
	if !strings.HasPrefix(got, "      - verylongcustomtype Titel") {
		t.Errorf("overlong prefix not followed by single space + title: %q", got)
	}
}

// TestRoadmapLinePrefixExactlyAtTitleCol pins the D17 boundary itself: a
// prefix of EXACTLY roadmapTitleCol (17) runes. D17 says "Präfix >= 17"
// (epic bean beans-1ec3) — at prefixW == 17 the overflow branch (one
// separating space) must fire, not the normal padding branch. The two
// branches happen to produce the same padding COUNT at this exact boundary
// (roadmapTitleCol-prefixW degenerates to 0 spaces), so a mutated `>`
// instead of `>=` produces prefix+title with NO separating space, while the
// correct `>=` produces prefix+" "+title. TestRoadmapLineOverlongPrefix uses
// a 26-rune prefix and does not exercise this boundary at all.
// Verified independently via `command python3`: correct/mutated outputs
// differ ("================= Word" vs "=================Word").
// prettyFixture builds the roadmapData literal that mirrors the DESIGN.md
// "## Ziel-Format (eingefroren)" demo hierarchy: Milestone Payment
// Integration -> Epic Checkout Flow -> (task, bug, Feature Stripe card
// entry -> task), plus Feature Apple Pay express button (direct under the
// milestone) -> task, plus a loose task directly under the milestone, and a
// "No Milestone" section with an orphan Epic Observability -> task and an
// orphan task. IDs/titles/types/status/priority/parent per beans-g5hz
// "Notes for T3".
func prettyFixture() *roadmapData {
	milestone := &bean.Bean{ID: "beans-fexy", Title: "Payment Integration", Type: "milestone", Status: "todo"}
	epic := &bean.Bean{ID: "beans-9m0d", Title: "Checkout Flow", Type: "epic", Status: "in-progress", Parent: "beans-fexy"}
	task9zpz := &bean.Bean{ID: "beans-9zpz", Title: "Validate card number (Luhn)", Type: "task", Status: "in-progress", Parent: "beans-9m0d"}
	bugWa9y := &bean.Bean{ID: "beans-wa9y", Title: "Total rounds off by one cent", Type: "bug", Status: "todo", Priority: "critical", Parent: "beans-9m0d"}
	feature1vvd := &bean.Bean{ID: "beans-1vvd", Title: "Stripe card entry", Type: "feature", Status: "todo", Priority: "high", Parent: "beans-9m0d"}
	taskB58r := &bean.Bean{ID: "beans-b58r", Title: "Refactor payment reconciliation ledger to support multi-currency settlement", Type: "task", Status: "todo", Priority: "high", Parent: "beans-1vvd"}
	feature9bi1 := &bean.Bean{ID: "beans-9bi1", Title: "Apple Pay express button", Type: "feature", Status: "draft", Priority: "high", Parent: "beans-fexy"}
	taskLnff := &bean.Bean{ID: "beans-lnff", Title: "Wire up sheet", Type: "task", Status: "todo", Parent: "beans-9bi1"}
	task635g := &bean.Bean{ID: "beans-635g", Title: "Update pricing copy", Type: "task", Status: "todo", Priority: "low", Parent: "beans-fexy"}
	epicH5km := &bean.Bean{ID: "beans-h5km", Title: "Observability", Type: "epic", Status: "todo"}
	taskXm6j := &bean.Bean{ID: "beans-xm6j", Title: "Add trace IDs", Type: "task", Status: "todo", Parent: "beans-h5km"}
	taskNfun := &bean.Bean{ID: "beans-nfun", Title: "Rotate signing key", Type: "task", Status: "todo"}

	return &roadmapData{
		Milestones: []milestoneGroup{
			{
				Milestone: milestone,
				Epics: []epicGroup{
					{
						Epic:  epic,
						Items: []*bean.Bean{task9zpz, bugWa9y},
						Features: []featureGroup{
							{Feature: feature1vvd, Items: []*bean.Bean{taskB58r}},
						},
					},
				},
				Features: []featureGroup{
					{Feature: feature9bi1, Items: []*bean.Bean{taskLnff}},
				},
				Other: []*bean.Bean{task635g},
			},
		},
		Unscheduled: &unscheduledGroup{
			Epics: []epicGroup{
				{Epic: epicH5km, Items: []*bean.Bean{taskXm6j}},
			},
			Other: []*bean.Bean{taskNfun},
		},
	}
}

// TestRoadmapClampWidth pins EARS-1/D08: clamp(terminalCols, 80, 110), with
// 0 (no terminal) landing on the floor like any other too-small value.
func TestRoadmapClampWidth(t *testing.T) {
	tests := []struct {
		name string
		cols int
		want int
	}{
		{"below floor", 40, 80},
		{"zero: no terminal (D08)", 0, 80},
		{"at floor", 80, 80},
		{"within range", 96, 96},
		{"at cap", 110, 110},
		{"above cap", 200, 110},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roadmapClampWidth(tt.cols); got != tt.want {
				t.Errorf("roadmapClampWidth(%d) = %d, want %d", tt.cols, got, tt.want)
			}
		})
	}
}

// TestRenderRoadmapPrettyAt80 pins the eingefroren DESIGN.md target format
// verbatim (extracted programmatically from docs/roadmap-tty-output/DESIGN.md
// "## Ziel-Format (eingefroren)" via `command python3`, not hand-typed --
// see task report for the extraction command). It is byte-for-byte the same
// block render-prototype.py produces against the equivalent demo data.
func TestRenderRoadmapPrettyAt80(t *testing.T) {
	want := `Roadmap
════════════════════════════════════════════════════════════════════════════════

■ Milestone      Payment Integration                           todo         fexy
  ▸ Epic         Checkout Flow                                 in-progress  9m0d
    - task       Validate card number (Luhn)                   in-progress  9zpz
    - bug        Total rounds off by one cent        critical  todo         wa9y
    ▪ Feature    Stripe card entry                       high  todo         1vvd
      - task     Refactor payment reconciliation         high  todo         b58r
                 ledger to support multi-currency
                 settlement
  ▪ Feature      Apple Pay express button                high  draft        9bi1
    - task       Wire up sheet                                 todo         lnff
  - task         Update pricing copy                      low  todo         635g

No Milestone

  ▸ Epic         Observability                                 todo         h5km
    - task       Add trace IDs                                 todo         xm6j
  - task         Rotate signing key                            todo         nfun
`
	got := renderRoadmapPretty(prettyFixture(), 80)
	if got != want {
		t.Errorf("renderRoadmapPretty() =\n%s\nwant\n%s", got, want)
	}
}

// TestRenderRoadmapPrettyLineWidths pins EARS-7 across all three clamp
// corners (floor, mid, cap): no rendered line may exceed W runes.
func TestRenderRoadmapPrettyLineWidths(t *testing.T) {
	for _, w := range []int{80, 96, 110} {
		got := renderRoadmapPretty(prettyFixture(), w)
		for i, line := range strings.Split(got, "\n") {
			if n := utf8.RuneCountInString(line); n > w {
				t.Errorf("width %d: line %d has %d runes (> %d): %q", w, i, n, w, line)
			}
		}
	}
}

// TestRenderRoadmapPrettyTitleColumn pins SC-404: every non-trivial rendered
// line (bean first-lines and their wrapped continuations) carries a
// non-space rune at column 17 -- the title never drifts off the fixed
// raster. Lines shorter than 18 runes (header, blank lines, "No Milestone")
// are skipped, since indexing rune 17 would be meaningless/out of range.
func TestRenderRoadmapPrettyTitleColumn(t *testing.T) {
	got := renderRoadmapPretty(prettyFixture(), 80)
	for i, line := range strings.Split(got, "\n") {
		runes := []rune(line)
		if len(runes) <= roadmapTitleCol {
			continue
		}
		if runes[roadmapTitleCol] == ' ' {
			t.Errorf("line %d: rune at column %d is a space (title misaligned): %q", i, roadmapTitleCol, line)
		}
	}
}

// TestRenderRoadmapPrettyPriorityVisibility pins EARS-4/D10/D15 per row
// type: Milestone and Epic rows never show priority, Feature-branch and
// leaf rows do. All four beans here share Priority "high" so the assertion
// is only satisfiable if the walker passes the correct showPrio bool for
// each of the four row kinds -- the frozen DESIGN.md fixture alone cannot
// catch a showPrio mix-up on Milestone/Epic, since their priority there is
// empty and would render identically (blank) whether shown or suppressed.
func TestRenderRoadmapPrettyPriorityVisibility(t *testing.T) {
	ms := &bean.Bean{ID: "beans-aaaa", Title: "M", Type: "milestone", Status: "todo", Priority: "high"}
	epic := &bean.Bean{ID: "beans-bbbb", Title: "E", Type: "epic", Status: "todo", Priority: "high", Parent: "beans-aaaa"}
	feat := &bean.Bean{ID: "beans-cccc", Title: "F", Type: "feature", Status: "todo", Priority: "high", Parent: "beans-bbbb"}
	leaf := &bean.Bean{ID: "beans-dddd", Title: "L", Type: "task", Status: "todo", Priority: "high", Parent: "beans-cccc"}

	data := &roadmapData{
		Milestones: []milestoneGroup{
			{
				Milestone: ms,
				Epics: []epicGroup{
					{Epic: epic, Features: []featureGroup{
						{Feature: feat, Items: []*bean.Bean{leaf}},
					}},
				},
			},
		},
	}
	got := renderRoadmapPretty(data, 80)
	lines := strings.Split(got, "\n")
	if len(lines) < 7 {
		t.Fatalf("expected at least 7 lines, got %d:\n%s", len(lines), got)
	}
	msLine, epicLine, featLine, leafLine := lines[3], lines[4], lines[5], lines[6]
	if strings.Contains(msLine, "high") {
		t.Errorf("milestone row must not show priority (D10): %q", msLine)
	}
	if strings.Contains(epicLine, "high") {
		t.Errorf("epic row must not show priority (D10): %q", epicLine)
	}
	if !strings.Contains(featLine, "high") {
		t.Errorf("feature row must show priority (D15): %q", featLine)
	}
	if !strings.Contains(leafLine, "high") {
		t.Errorf("leaf row must show priority: %q", leafLine)
	}
}

// TestRenderRoadmapPrettyEmpty pins EARS-9: an empty roadmapData renders
// only the header and separator line.
func TestRenderRoadmapPrettyEmpty(t *testing.T) {
	got := renderRoadmapPretty(&roadmapData{}, 80)
	want := "Roadmap\n" + strings.Repeat("═", 80) + "\n"
	if got != want {
		t.Errorf("renderRoadmapPretty(empty) =\n%q\nwant\n%q", got, want)
	}
}

func TestRoadmapLinePrefixExactlyAtTitleCol(t *testing.T) {
	prefix := strings.Repeat("=", roadmapTitleCol) // synthetic, exactly 17 runes
	if n := utf8.RuneCountInString(prefix); n != roadmapTitleCol {
		t.Fatalf("test setup: prefix has %d runes, want %d", n, roadmapTitleCol)
	}
	b := &bean.Bean{ID: "beans-ex17", Title: "Word", Type: "task",
		Status: "todo", Priority: "normal"}
	got := roadmapLine(prefix, b, false, 80)
	want := prefix + " " + "Word"
	if !strings.HasPrefix(got, want) {
		t.Errorf("prefix at exactly title col %d: got %q, want prefix %q", roadmapTitleCol, got, want)
	}
}
