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
