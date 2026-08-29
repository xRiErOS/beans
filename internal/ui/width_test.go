// internal/ui/width_test.go
package ui

import (
	"reflect"
	"testing"
)

func TestDisplayWidthCountsCellsNotBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"abc", 3},
		{"", 0},
		{"├─ ", 3},    // three runes, seven bytes
		{"│  └─ ", 6}, // the deepest connector shape
		{"Größe", 5},  // umlaut is one cell
		{"日本", 4},     // east asian wide: two cells each
	}
	for _, c := range cases {
		if got := DisplayWidth(c.in); got != c.want {
			t.Errorf("DisplayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestPadRightFillsButNeverCuts(t *testing.T) {
	if got := PadRight("ab", 5); got != "ab   " {
		t.Errorf("PadRight(\"ab\", 5) = %q, want %q", got, "ab   ")
	}
	if got := PadRight("abcdef", 3); got != "abcdef" {
		t.Errorf("PadRight must not cut, got %q", got)
	}
	if got := DisplayWidth(PadRight("├─", 6)); got != 6 {
		t.Errorf("padded multi-byte width = %d, want 6", got)
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"abcdef", 10, "abcdef"},
		{"abcdef", 6, "abcdef"},
		{"abcdef", 4, "abc…"},
		{"abcdef", 1, "…"},
		{"abcdef", 0, ""},
		{"abcdef", -3, ""},
	}
	for _, c := range cases {
		if got := Truncate(c.in, c.width); got != c.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
		}
	}
}

func TestWrapText(t *testing.T) {
	got := WrapText("the quick brown fox", 10)
	want := []string{"the quick", "brown fox"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WrapText = %#v, want %#v", got, want)
	}
}

func TestWrapTextHardBreaksAWordThatCannotFit(t *testing.T) {
	got := WrapText("supercalifragilistic", 8)
	want := []string{"supercal", "ifragili", "stic"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("WrapText = %#v, want %#v", got, want)
	}
}

func TestWrapTextNeverReturnsEmpty(t *testing.T) {
	if got := WrapText("", 10); len(got) != 1 || got[0] != "" {
		t.Errorf("WrapText(\"\", 10) = %#v, want one empty line", got)
	}
}

func TestWrapTextNoLineExceedsTheWidth(t *testing.T) {
	long := "Canary-Instanz sproutling-test — Staged-Rollout & Dogfood auf NAS"
	for _, line := range WrapText(long, 24) {
		if w := DisplayWidth(line); w > 24 {
			t.Errorf("line %q is %d cells wide, want at most 24", line, w)
		}
	}
}
