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
