package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// setupShowTest installs a throwaway core and the default config into the
// package globals that showStyledBean reads (cfg.GetStatus, core.ImplicitStatus)
// and restores both afterwards.
func setupShowTest(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.Default()
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })
}

func showTestBean(id, title, body string) *bean.Bean {
	return &bean.Bean{
		ID:     id,
		Slug:   bean.Slugify(title),
		Title:  title,
		Status: "todo",
		Type:   "task",
		Body:   body,
	}
}

func mustRender(t *testing.T, b *bean.Bean) string {
	t.Helper()
	content, err := b.Render()
	if err != nil {
		t.Fatalf("bean.Render() error = %v", err)
	}
	return string(content)
}

// REQ-01 / AC-01.1: without a terminal on stdout, the default output path must
// be the byte-identical raw rendering -- the same text `--raw` produces. A
// mutation that keeps the glamour path for the non-TTY branch turns this red.
func TestShowOutputSwitchesOnTTY(t *testing.T) {
	setupShowTest(t)
	b := showTestBean("beans-test1", "A test bean", "# Heading\n\nSome body text.\n")

	t.Run("non-tty is byte-identical to raw", func(t *testing.T) {
		got, err := showOutput(b, false, false)
		if err != nil {
			t.Fatalf("showOutput() error = %v", err)
		}
		want := mustRender(t, b)
		if got != want {
			t.Errorf("non-TTY output differs from bean.Render()\n got: %q\nwant: %q", got, want)
		}
	})

	// REQ-02 / AC-02.1: the terminal branch keeps the styled rendering. ANSI is
	// deliberately not asserted (D04) -- under `go test` stdout is not a
	// terminal, so lipgloss and glamour degrade their colour profile and emit
	// no escape sequences. The horizontal rule is the stable marker.
	t.Run("tty renders the styled representation", func(t *testing.T) {
		got, err := showOutput(b, true, false)
		if err != nil {
			t.Fatalf("showOutput() error = %v", err)
		}
		rule := strings.Repeat("─", 50)
		if !strings.Contains(got, rule) {
			t.Errorf("expected TTY output to contain the 50-char horizontal rule, got %q", got)
		}
		raw := mustRender(t, b)
		if got == raw {
			t.Error("TTY output must differ from the raw rendering")
		}
	})
}

// REQ-04 / AC-04.1: without a terminal, consecutive beans are joined by
// "\n---\n\n". The assertion is an equality against the expected concatenation
// rather than a substring count: bean.Render emits "\n---\n\n" around its own
// closing front-matter fence, so a global count would not isolate the joint.
func TestShowOutputAllSeparatorNonTTY(t *testing.T) {
	setupShowTest(t)
	b1 := showTestBean("beans-test4", "First bean", "First body.\n")
	b2 := showTestBean("beans-test5", "Second bean", "Second body.\n")

	got, err := showOutputAll([]*bean.Bean{b1, b2}, false, false)
	if err != nil {
		t.Fatalf("showOutputAll() error = %v", err)
	}

	want := mustRender(t, b1) + "\n---\n\n" + mustRender(t, b2)
	if got != want {
		t.Errorf("non-TTY join differs\n got: %q\nwant: %q", got, want)
	}
}

// REQ-02 / AC-02.2: with a terminal, consecutive beans are joined by the
// separator showCmd.RunE builds today from three fmt.Println calls -- a blank
// line, a rule of 60 U+2550, and another blank line.
func TestShowOutputAllSeparatorTTY(t *testing.T) {
	setupShowTest(t)
	b1 := showTestBean("beans-test6", "First bean", "First body.\n")
	b2 := showTestBean("beans-test7", "Second bean", "Second body.\n")

	got, err := showOutputAll([]*bean.Bean{b1, b2}, true, false)
	if err != nil {
		t.Fatalf("showOutputAll() error = %v", err)
	}

	sep := "\n" + ui.Muted.Render(strings.Repeat("═", 60)) + "\n\n"
	if n := strings.Count(got, sep); n != 1 {
		t.Errorf("expected exactly 1 occurrence of the TTY separator, got %d", n)
	}

	first, err := showOutput(b1, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	second, err := showOutput(b2, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	if want := first + sep + second; got != want {
		t.Errorf("TTY join differs\n got: %q\nwant: %q", got, want)
	}
}

// REQ-01 / AC-01.3: a bean without a body still round-trips byte-identically,
// including the trailing POSIX newline bean.Render appends after the closing
// front-matter fence.
func TestShowOutputEmptyBodyNonTTY(t *testing.T) {
	setupShowTest(t)
	b := showTestBean("beans-test2", "Bean without body", "")

	got, err := showOutput(b, false, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	if want := mustRender(t, b); got != want {
		t.Errorf("non-TTY output differs from bean.Render()\n got: %q\nwant: %q", got, want)
	}
	if !strings.HasSuffix(got, "---\n\n") {
		t.Errorf("expected output to end with %q, got tail %q", "---\n\n", got[max(0, len(got)-10):])
	}
}

// REQ-01: guards the wiring, not just the branch. The unit tests above call
// showOutput directly, so they stay green even if showCmd.RunE stops consulting
// term.IsTerminal and hardcodes the styled path. This test runs the command
// itself with os.Stdout replaced by a pipe -- which is exactly the non-TTY
// case -- and pins the output to bean.Render().
func TestShowCmdNonTTYWiring(t *testing.T) {
	setupShowTest(t)

	b := showTestBean("beans-test8", "Wired bean", "# Heading\n\nSome body text.\n")
	if err := core.Create(b); err != nil {
		t.Fatalf("failed to create test bean: %v", err)
	}

	oldJSON, oldRaw, oldBody, oldETag := showJSON, showRaw, showBodyOnly, showETagOnly
	showJSON, showRaw, showBodyOnly, showETagOnly = false, false, false, false
	t.Cleanup(func() {
		showJSON, showRaw, showBodyOnly, showETagOnly = oldJSON, oldRaw, oldBody, oldETag
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	oldStdout := os.Stdout
	os.Stdout = w

	runErr := showCmd.RunE(showCmd, []string{b.ID})

	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}
	captured, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	if runErr != nil {
		t.Fatalf("showCmd.RunE() error = %v", runErr)
	}

	stored, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if want := mustRender(t, stored); string(captured) != want {
		t.Errorf("piped command output differs from bean.Render()\n got: %q\nwant: %q", string(captured), want)
	}
}

// REQ-05 / AC-05.1: the non-TTY path must not hard-wrap. A 300-character
// paragraph occupying one source line stays one output line; the glamour
// renderer would break it at 80 columns.
func TestShowNonTTYPreservesLineStructure(t *testing.T) {
	setupShowTest(t)
	longLine := strings.Repeat("lorem ipsum dolor sit amet ", 12) // 324 chars
	b := showTestBean("beans-test3", "Bean with a long paragraph", longLine+"\n")

	got, err := showOutput(b, false, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}

	var found int
	for _, line := range strings.Split(got, "\n") {
		if line == longLine {
			found++
		}
	}
	if found != 1 {
		t.Errorf("expected the %d-character paragraph as exactly 1 line, found %d", len(longLine), found)
	}
}

// runShowInTestStore returns the styled (TTY) show output for one bean
// carrying the given body. It calls showOutput directly with isTTY forced
// true, the same way the other styled-output tests in this file do --
// routing through showCmd.RunE would capture stdout via an os.Pipe, which is
// itself not a terminal and would silently select the non-TTY branch instead.
func runShowInTestStore(t *testing.T, body string) string {
	t.Helper()
	setupShowTest(t)
	b := showTestBean("beans-detail1", "A detail bean", body)
	out, err := showOutput(b, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	return out
}

// TestShowEmitsNoGlamourPadding guards against glamour's signature defect:
// trailing spaces -- often colour-painted -- running to the right margin of
// every line. withTrueColor forces a real colour profile so stripANSITest
// has ANSI codes to strip; without it, `go test` has no controlling tty, no
// colour is ever emitted, and colour-painted padding would disappear before
// the trailing-space check could see it.
func TestShowEmitsNoGlamourPadding(t *testing.T) {
	withTrueColor(t)
	out := runShowInTestStore(t, "a bean with a body")
	for _, line := range strings.Split(out, "\n") {
		plain := stripANSITest(line)
		if plain != strings.TrimRight(plain, " ") {
			t.Errorf("line carries trailing padding: %q", plain)
		}
	}
}

// TestShowHeaderCarriesTypeIDAndStatus pins the minimum content of the new
// attribute header: the bean's type and status must both still be visible
// once colour is stripped.
func TestShowHeaderCarriesTypeIDAndStatus(t *testing.T) {
	withTrueColor(t)
	out := stripANSITest(runShowInTestStore(t, "a bean with a body"))
	for _, want := range []string{"task", "todo"} {
		if !strings.Contains(out, want) {
			t.Errorf("detail header is missing %q:\n%s", want, out)
		}
	}
}

// TestShowHeaderOrdersTypeIDTitleThenStatus pins the vertical reading order
// the task requires: type, id, then title, then status (and priority, when
// present) -- the same order the beans table reads across, read down
// instead. It asserts on index positions rather than a fixed line layout, so
// it survives spacing changes but still catches the order flipping.
func TestShowHeaderOrdersTypeIDTitleThenStatus(t *testing.T) {
	withTrueColor(t)
	out := stripANSITest(runShowInTestStore(t, "a bean with a body"))

	typeIdx := strings.Index(out, "task")
	idIdx := strings.Index(out, "beans-detail1")
	titleIdx := strings.Index(out, "A detail bean")
	statusIdx := strings.Index(out, "todo")

	if typeIdx < 0 || idIdx < 0 || titleIdx < 0 || statusIdx < 0 {
		t.Fatalf("expected all of type/id/title/status present, got indices %d/%d/%d/%d in:\n%s",
			typeIdx, idIdx, titleIdx, statusIdx, out)
	}
	if !(typeIdx < idIdx && idIdx < titleIdx && titleIdx < statusIdx) {
		t.Errorf("header order wrong: type=%d id=%d title=%d status=%d, want type < id < title < status\n%s",
			typeIdx, idIdx, titleIdx, statusIdx, out)
	}
}

// TestShowUsesNoBackgroundBadges guards against glamour's other signature
// defect: background-painted badges that don't survive next to a flat
// raster. withTrueColor forces a real colour profile so this failure mode --
// a Background() call slipping back into the attribute header -- would
// actually emit the escape sequence for the assertion to catch.
func TestShowUsesNoBackgroundBadges(t *testing.T) {
	withTrueColor(t)
	out := runShowInTestStore(t, "a bean with a body")
	if strings.Contains(out, "\x1b[48;") {
		t.Error("detail view still paints background badges")
	}
}

// TestShowHeaderCarriesCreatedAndUpdatedTimestamps guards against silently
// dropping user-visible information under this rewrite: the plan's only
// authorised behaviour change is removing glamour, everything else is
// presentation. The old header carried created/updated timestamps as muted
// text; the new attribute header must still carry them somewhere sensible.
func TestShowHeaderCarriesCreatedAndUpdatedTimestamps(t *testing.T) {
	setupShowTest(t)
	created := time.Date(2026, 1, 2, 15, 4, 0, 0, time.UTC)
	updated := time.Date(2026, 3, 4, 9, 30, 0, 0, time.UTC)
	b := showTestBean("beans-detail2", "A timestamped bean", "body text")
	b.CreatedAt = &created
	b.UpdatedAt = &updated

	out, err := showOutput(b, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	plain := stripANSITest(out)

	for _, want := range []string{
		"created 2026-01-02 15:04 UTC",
		"updated 2026-03-04 09:30 UTC",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("detail header is missing %q:\n%s", want, plain)
		}
	}
}

// TestShowDetailShowsNormalPriority guards against a second unauthorised
// behaviour change: hiding a "normal" priority in the detail view. Hiding it
// is correct in the table (Task 10's prioCell), where it removes a repeated
// word from every row and buys density; a detail view shows exactly one
// bean, has no density to buy, and previously showed the badge for it. The
// title and body deliberately avoid the word "normal" so the only possible
// source of the substring is the priority attribute itself.
func TestShowDetailShowsNormalPriority(t *testing.T) {
	setupShowTest(t)
	b := showTestBean("beans-detail3", "Detail priority bean", "body text")
	b.Priority = "normal"

	out, err := showOutput(b, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	plain := stripANSITest(out)
	if !strings.Contains(plain, "normal") {
		t.Errorf("detail view hides the normal priority:\n%s", plain)
	}
}

// TestShowDetailShowsUnknownStatus guards against the attribute vanishing
// entirely when the config has no matching entry -- the one case where a
// reader most needs to see the raw value, not less of it. The old code fell
// back to the "gray" legacy colour alias in that case; the new code must
// keep rendering the value even though its colour resolution has nothing to
// key off.
func TestShowDetailShowsUnknownStatus(t *testing.T) {
	setupShowTest(t)
	b := showTestBean("beans-detail5", "Bean with an odd status", "body text")
	b.Status = "wibble"

	out, err := showOutput(b, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	plain := stripANSITest(out)
	if !strings.Contains(plain, "wibble") {
		t.Errorf("detail view hides an unconfigured status:\n%s", plain)
	}
}

// showFullBean returns a bean carrying every front matter field the format
// allows -- tags, both blocking directions, order and unknown ("extra")
// keys -- so a detail-view test can assert on the whole front matter rather
// than the subset the header happened to render.
func showFullBean(body string) *bean.Bean {
	created := time.Date(2026, 9, 5, 10, 15, 5, 0, time.UTC)
	updated := time.Date(2026, 9, 5, 20, 27, 9, 0, time.UTC)
	return &bean.Bean{
		ID:        "beans-full1",
		Slug:      "a-full-bean",
		Title:     "A full bean",
		Status:    "todo",
		Type:      "task",
		Priority:  "high",
		Tags:      []string{"reviewed", "backend"},
		CreatedAt: &created,
		UpdatedAt: &updated,
		Order:     "a0",
		Parent:    "beans-paren",
		Blocking:  []string{"beans-block1"},
		BlockedBy: []string{"beans-blkby1"},
		Extra: map[string]any{
			"branch":  "feature/beans-full1-a-full-bean",
			"release": "0-9-0",
			"reviews": []any{"beans-rev1", "beans-rev2"},
			"gate":    map[string]any{"suite": "green", "reviewer": "ReviewSix"},
		},
		Body: body,
	}
}

// TestShowHeaderCarriesWholeFrontMatter pins that the styled detail view
// withholds nothing the file holds, and that it carries it *above* the body:
// tags used to be printed after the rendered markdown, which put them below
// a screenful of text on any real bean, and blocked_by and unknown keys were
// not rendered at all. Splitting on the horizontal rule is what makes this a
// placement assertion and not just a "appears somewhere" one.
func TestShowHeaderCarriesWholeFrontMatter(t *testing.T) {
	setupShowTest(t)
	b := showFullBean("Body text that mentions nothing else.\n")

	out, err := showOutput(b, true, false)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	plain := stripANSITest(out)

	rule := strings.Index(plain, strings.Repeat("─", 10))
	if rule < 0 {
		t.Fatalf("no horizontal rule in styled output:\n%s", plain)
	}
	header := plain[:rule]

	for _, want := range []string{
		"#reviewed", "#backend",
		"parent: beans-paren",
		"blocking: beans-block1",
		"blocked by: beans-blkby1",
		"branch: feature/beans-full1-a-full-bean",
		"release: 0-9-0",
		"gate: {reviewer: ReviewSix, suite: green}",
		"reviews: [beans-rev1, beans-rev2]",
		"order a0",
	} {
		if !strings.Contains(header, want) {
			t.Errorf("header is missing %q\nheader:\n%s", want, header)
		}
	}
}

// TestShowMetaDropsBodyOnATerminal: --meta answers "what is this bean" for a
// reader who does not want to page through the body. The header must survive
// whole; the body and the rule that introduces it must be gone.
func TestShowMetaDropsBodyOnATerminal(t *testing.T) {
	setupShowTest(t)
	b := showFullBean("Distinctive body sentinel.\n")

	out, err := showOutput(b, true, true)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	plain := stripANSITest(out)

	if strings.Contains(plain, "Distinctive body sentinel") {
		t.Errorf("--meta printed the body:\n%s", plain)
	}
	if strings.Contains(plain, strings.Repeat("─", 10)) {
		t.Errorf("--meta printed the body rule:\n%s", plain)
	}
	for _, want := range []string{"beans-full1", "A full bean", "#reviewed", "blocked by: beans-blkby1"} {
		if !strings.Contains(plain, want) {
			t.Errorf("--meta is missing %q\ngot:\n%s", want, plain)
		}
	}
}

// TestShowMetaOffATerminalParsesAsABean: off a terminal, show emits source
// markdown a parser can read; --meta must not break that contract by
// emitting a styled or truncated form. The output has to parse back into the
// same front matter with an empty body.
func TestShowMetaOffATerminalParsesAsABean(t *testing.T) {
	setupShowTest(t)
	b := showFullBean("Distinctive body sentinel.\n")

	out, err := showOutput(b, false, true)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	if strings.Contains(out, "Distinctive body sentinel") {
		t.Errorf("--meta printed the body:\n%s", out)
	}

	got, err := bean.Parse(strings.NewReader(out))
	if err != nil {
		t.Fatalf("bean.Parse() error = %v, output:\n%s", err, out)
	}
	if strings.TrimSpace(got.Body) != "" {
		t.Errorf("parsed body = %q, want empty", got.Body)
	}
	if strings.Join(got.Tags, ",") != "reviewed,backend" {
		t.Errorf("parsed tags = %v, want [reviewed backend]", got.Tags)
	}
	if strings.Join(got.BlockedBy, ",") != "beans-blkby1" {
		t.Errorf("parsed blocked_by = %v, want [beans-blkby1]", got.BlockedBy)
	}
	if got.Extra["release"] != "0-9-0" {
		t.Errorf("parsed extra[release] = %v, want %q", got.Extra["release"], "0-9-0")
	}

	// The source bean must be untouched: --meta reads, it does not edit.
	if b.Body != "Distinctive body sentinel.\n" {
		t.Errorf("--meta mutated the bean's body to %q", b.Body)
	}
}

// TestShowMetaOffATerminalEmitsNoEmptyDocument: each --meta block ends with
// the front matter's own closing "---", so the raw separator on top of it
// produced an empty document between every pair of beans -- a stream that
// still "looks like" markdown but hands a parser a bean with no fields.
func TestShowMetaOffATerminalEmitsNoEmptyDocument(t *testing.T) {
	setupShowTest(t)
	b1 := showFullBean("First body.\n")
	b2 := showTestBean("beans-second", "A second bean", "Second body.\n")

	got, err := showOutputAll([]*bean.Bean{b1, b2}, false, true)
	if err != nil {
		t.Fatalf("showOutputAll() error = %v", err)
	}

	// Two beans are exactly four delimiter lines: an opening and a closing
	// "---" each. A fifth is the stray separator, and the empty document it
	// opens is what a parser then reads as a bean with no fields.
	var delimiters int
	for _, line := range strings.Split(got, "\n") {
		if line == "---" {
			delimiters++
		}
	}
	if delimiters != 4 {
		t.Errorf("got %d \"---\" lines, want 4 (two per bean):\n%s", delimiters, got)
	}
	if !strings.Contains(got, "beans-full1") || !strings.Contains(got, "beans-second") {
		t.Errorf("both beans must appear:\n%s", got)
	}
}

// TestExtraValueStaysOnOneLine pins the header's one-line-per-key contract
// against the YAML emitter's line-breaking. yaml.v3 does not break flow
// output today -- yaml_emitter_initialize sets best_width to -1 and
// emitterc.go raises that to MaxInt32, while the 80-column default only
// applies when the width is set explicitly through the private
// yaml_emitter_set_width -- so no whitespace collapsing is needed. That is a
// property of the dependency, not of this code, which is exactly why it
// belongs in a test: a yaml upgrade that starts wrapping at 80 columns would
// silently tear one header line into several without one.
//
// The space-bearing values matter: a plain scalar containing spaces is the
// only place the emitter has a break candidate at all (emitterc.go:1620).
func TestExtraValueStaysOnOneLine(t *testing.T) {
	wide := make([]any, 40)
	for i := range wide {
		wide[i] = fmt.Sprintf("beans-r%03d", i)
	}

	cases := map[string]any{
		"wide sequence": wide,
		"wide mapping": map[string]any{
			"reviewer": "ReviewSix", "suite": "green", "commit": "8c08a5a",
			"branch": "feature/beans-full1-a-rather-long-branch-name",
			"note":   "a fairly long sentence value that contains spaces",
		},
		"nested":            map[string]any{"gate": map[string]any{"suite": "green"}, "reviews": wide},
		"space-bearing seq": []any{"a fairly long sentence value that contains spaces and more", "and a second one to push well past eighty columns"},
	}

	for name, v := range cases {
		got := formatExtraValue(v)
		if strings.Contains(got, "\n") {
			t.Errorf("formatExtraValue(%s) spans more than one line:\n%s", name, got)
		}
	}

	// Flow style is what keeps it readable on that one line: block markers
	// folded onto a single line ("- a - b") are not the value's notation.
	if got := formatExtraValue([]any{"a", "b"}); got != "[a, b]" {
		t.Errorf("formatExtraValue([a b]) = %q, want %q", got, "[a, b]")
	}
	if got := formatExtraValue(map[string]any{"k": "v"}); got != "{k: v}" {
		t.Errorf("formatExtraValue({k: v}) = %q, want %q", got, "{k: v}")
	}
}
