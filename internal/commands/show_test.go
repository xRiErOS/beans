package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
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
		got, err := showOutput(b, false)
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
		got, err := showOutput(b, true)
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

	got, err := showOutputAll([]*bean.Bean{b1, b2}, false)
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

	got, err := showOutputAll([]*bean.Bean{b1, b2}, true)
	if err != nil {
		t.Fatalf("showOutputAll() error = %v", err)
	}

	sep := "\n" + ui.Muted.Render(strings.Repeat("═", 60)) + "\n\n"
	if n := strings.Count(got, sep); n != 1 {
		t.Errorf("expected exactly 1 occurrence of the TTY separator, got %d", n)
	}

	first, err := showOutput(b1, true)
	if err != nil {
		t.Fatalf("showOutput() error = %v", err)
	}
	second, err := showOutput(b2, true)
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

	got, err := showOutput(b, false)
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

	got, err := showOutput(b, false)
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
	out, err := showOutput(b, true)
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
