package commands

import (
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
)

// tagsFixture is a one-milestone roadmap where the milestone and one leaf
// carry tags and a second leaf carries none, so a single render exercises
// both the tagged and the untagged path.
func tagsFixture() *roadmapData {
	ms := &bean.Bean{ID: "beans-ms01", Title: "Payments", Type: "milestone", Status: "todo", Tags: []string{"planning", "cli"}}
	tagged := &bean.Bean{ID: "beans-it01", Title: "Wire up sheet", Type: "task", Status: "todo", Parent: ms.ID, Tags: []string{"ux"}}
	untagged := &bean.Bean{ID: "beans-it02", Title: "Rotate signing key", Type: "task", Status: "todo", Parent: ms.ID}

	return &roadmapData{
		Milestones: []milestoneGroup{{
			Milestone: ms,
			Other:     []*bean.Bean{tagged, untagged},
		}},
	}
}

// TestRoadmapTagLineRendersAtHangingIndent pins the TTY tag row: its own
// line beneath the title, starting at the title column, tags prefixed with
// "#" and separated by a single space.
func TestRoadmapTagLineRendersAtHangingIndent(t *testing.T) {
	out := renderRoadmapPretty(tagsFixture(), 110, true)
	lines := strings.Split(out, "\n")

	msIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "Payments") {
			msIdx = i
			break
		}
	}
	if msIdx < 0 {
		t.Fatalf("milestone row not found in:\n%s", out)
	}

	want := strings.Repeat(" ", roadmapTitleCol) + "#planning #cli"
	if got := lines[msIdx+1]; got != want {
		t.Errorf("tag line = %q, want %q", got, want)
	}
}

// TestRoadmapTagLineOnlyForTaggedBeans pins that an untagged bean gets no
// row of its own -- no blank line, no stray "#".
func TestRoadmapTagLineOnlyForTaggedBeans(t *testing.T) {
	out := renderRoadmapPretty(tagsFixture(), 110, true)
	lines := strings.Split(out, "\n")

	for i, l := range lines {
		if !strings.Contains(l, "Rotate signing key") {
			continue
		}
		if i+1 < len(lines) && strings.Contains(lines[i+1], "#") {
			t.Errorf("untagged bean followed by tag line %q", lines[i+1])
		}
		return
	}
	t.Fatalf("untagged row not found in:\n%s", out)
}

// TestRoadmapPrettyWithoutTagsFlagUnchanged pins that --tags is opt-in: the
// default render is byte-identical to what it was before the flag existed.
func TestRoadmapPrettyWithoutTagsFlagUnchanged(t *testing.T) {
	out := renderRoadmapPretty(tagsFixture(), 110, false)

	if strings.Contains(out, "#planning") || strings.Contains(out, "#ux") {
		t.Errorf("tags leaked into the default render:\n%s", out)
	}
}

// TestRoadmapTagLineWrapsAtTitleWidth pins that a long tag list wraps like a
// title does: every continuation line starts at the title column, and no
// line runs past the roadmap width.
func TestRoadmapTagLineWrapsAtTitleWidth(t *testing.T) {
	many := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliett", "kilo", "lima"}
	ms := &bean.Bean{ID: "beans-ms01", Title: "Payments", Type: "milestone", Status: "todo", Tags: many}
	data := &roadmapData{Milestones: []milestoneGroup{{Milestone: ms}}}

	const width = 80
	out := renderRoadmapPretty(data, width, true)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	var tagLines []string
	for _, l := range lines {
		if strings.Contains(l, "#") {
			tagLines = append(tagLines, l)
		}
	}
	if len(tagLines) < 2 {
		t.Fatalf("expected the tag list to wrap over several lines, got %d:\n%s", len(tagLines), out)
	}
	for _, l := range tagLines {
		if !strings.HasPrefix(l, strings.Repeat(" ", roadmapTitleCol)+"#") {
			t.Errorf("tag line not at the title column: %q", l)
		}
		if len([]rune(l)) > width {
			t.Errorf("tag line runs past width %d: %q", width, l)
		}
	}
}

// TestRoadmapTagLineFollowsWrappedTitle pins the ordering when both wrap:
// the tag row comes after every continuation line of the title, not between
// them.
func TestRoadmapTagLineFollowsWrappedTitle(t *testing.T) {
	long := "Refactor payment reconciliation ledger to support multi-currency settlement across regions"
	ms := &bean.Bean{ID: "beans-ms01", Title: long, Type: "milestone", Status: "todo", Tags: []string{"ledger"}}
	data := &roadmapData{Milestones: []milestoneGroup{{Milestone: ms}}}

	out := renderRoadmapPretty(data, 80, true)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

	last := lines[len(lines)-1]
	if last != strings.Repeat(" ", roadmapTitleCol)+"#ledger" {
		t.Errorf("last line = %q, want the tag row after the wrapped title\nfull output:\n%s", last, out)
	}
	if strings.Contains(lines[len(lines)-2], "#ledger") {
		t.Errorf("tag row rendered twice:\n%s", out)
	}
}

// TestRoadmapMarkdownTagLine pins the Markdown mirror: tags on their own
// line, as inline code, indented under a list item so they stay part of it.
func TestRoadmapMarkdownTagLine(t *testing.T) {
	out := renderRoadmapMarkdown(tagsFixture(), false, "", true)

	if !strings.Contains(out, "\n`#planning` `#cli`\n") {
		t.Errorf("milestone tag line missing from:\n%s", out)
	}
	if !strings.Contains(out, "\n  `#ux`\n") {
		t.Errorf("leaf tag line missing or not indented in:\n%s", out)
	}
	if strings.Contains(out, "`#`") {
		t.Errorf("untagged bean produced an empty tag line:\n%s", out)
	}
}

// TestRoadmapMarkdownWithoutTagsFlagUnchanged pins that the Markdown path is
// also opt-in.
func TestRoadmapMarkdownWithoutTagsFlagUnchanged(t *testing.T) {
	withFlag := renderRoadmapMarkdown(tagsFixture(), false, "", false)

	if strings.Contains(withFlag, "#planning") || strings.Contains(withFlag, "#ux") {
		t.Errorf("tags leaked into the default Markdown render:\n%s", withFlag)
	}
}
