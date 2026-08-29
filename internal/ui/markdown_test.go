package ui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// withTrueColor forces lipgloss's default renderer to TrueColor for the
// duration of the test and restores whatever profile was in force before.
// lipgloss.SetColorProfile exists "mostly for testing purposes" per its own
// doc comment: without it, output captured under `go test` (no controlling
// tty) renders with no ANSI codes at all, which would make colour-bearing
// assertions vacuously true regardless of whether the code under test
// actually applies any colour.
func withTrueColor(t *testing.T) {
	t.Helper()
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(old) })
}

var ansiEscape = regexp.MustCompile("\x1b\\[[0-9;]*m")

// stripANSI removes CSI colour/style escape sequences, leaving the text a
// plain-text assertion can match against regardless of the active theme or
// colour profile.
func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

func TestRenderMarkdownLeavesNoTrailingPadding(t *testing.T) {
	withTrueColor(t)
	// glamour pads every line to the full width with coloured spaces, which is
	// the visible rubbish this replaces. A code line carrying trailing spaces
	// in the source (common in pasted terminal output) must not leak them
	// into the rendered output either — that path preserves the line
	// verbatim except for the trailing trim, so it is the one place a
	// dropped trim would actually show up.
	in := "A short paragraph.\n\n```\ncode line with trailing spaces   \n```\n"
	out := RenderMarkdown(in, 80)
	for _, line := range strings.Split(out, "\n") {
		plain := stripANSI(line)
		if plain != strings.TrimRight(plain, " ") {
			t.Errorf("line has trailing padding: %q", plain)
		}
	}
}

func TestRenderMarkdownWrapsProse(t *testing.T) {
	withTrueColor(t)
	long := strings.Repeat("word ", 60)
	for _, line := range strings.Split(RenderMarkdown(long, 50), "\n") {
		if got := DisplayWidth(stripANSI(line)); got > 50 {
			t.Errorf("line is %d cells, want at most 50: %q", got, stripANSI(line))
		}
	}
}

func TestRenderMarkdownWrapsBeforeColouring(t *testing.T) {
	withTrueColor(t)
	// WrapText enforces its width bound on whatever bytes it is given, so a
	// bug that colours prose before wrapping it does not show up as an
	// overlong line — WrapText will happily hard-break mid-escape-sequence
	// to stay under budget. What it does do is eat part of the width budget
	// on invisible escape bytes, so fewer real words fit on the first line
	// than plain-text wrapping would allow. Pin the exact word count to
	// catch that regression.
	long := strings.Repeat("word ", 60)
	lines := strings.Split(RenderMarkdown(long, 50), "\n")
	if len(lines) == 0 {
		t.Fatal("no output")
	}
	first := strings.TrimSpace(stripANSI(lines[0]))
	want := strings.TrimSpace(strings.Repeat("word ", 9))
	if first != want {
		t.Errorf("first line = %q, want %q (wrapping must run on plain text before any colour is applied)", first, want)
	}
}

func TestRenderMarkdownMarksHeadings(t *testing.T) {
	withTrueColor(t)
	out := stripANSI(RenderMarkdown("## Identified Improvements", 80))
	if strings.Contains(out, "##") {
		t.Errorf("heading markers should not survive: %q", out)
	}
	if !strings.Contains(out, "Identified Improvements") {
		t.Errorf("heading text is missing: %q", out)
	}
}

func TestRenderMarkdownTurnsDashesIntoBullets(t *testing.T) {
	withTrueColor(t)
	out := stripANSI(RenderMarkdown("- first item\n- second item", 80))
	if !strings.Contains(out, "•") {
		t.Errorf("list items should render as bullets: %q", out)
	}
}

func TestRenderMarkdownKeepsCodeBlocksVerbatim(t *testing.T) {
	withTrueColor(t)
	in := "text\n\n```\n  indented   code\n```\n"
	out := stripANSI(RenderMarkdown(in, 80))
	if !strings.Contains(out, "  indented   code") {
		t.Errorf("code block lost its spacing: %q", out)
	}
	if strings.Contains(out, "```") {
		t.Errorf("fence markers should not survive: %q", out)
	}
}

func TestRenderMarkdownHandlesAnEmptyBody(t *testing.T) {
	if got := RenderMarkdown("", 80); strings.TrimSpace(got) != "" {
		t.Errorf("empty body rendered %q, want nothing", got)
	}
}
