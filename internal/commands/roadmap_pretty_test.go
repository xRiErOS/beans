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
			if len(got) != roadmapRightW {
				t.Errorf("roadmapRightBlock() width = %d, want %d", len(got), roadmapRightW)
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
