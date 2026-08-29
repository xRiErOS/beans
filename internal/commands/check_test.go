package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// TestCheckHelperProcess is not a real test: it is the in-process side of
// the subprocess trick runCheckInTestStore uses to exercise checkCmd safely.
//
// checkCmd.RunE ends with `if totalIssues > 0 { os.Exit(1) }`, and it always
// finds at least one issue today: config.DefaultTypes still carries an
// empty Color for "task", and the Catppuccin-only ui.IsValidColor (task 3)
// rejects an empty colour as invalid. task-3-report.md flags this exact gap
// as the surface Task 20 was meant to close, but this batch scopes Task 20
// to a test only (no check.go edit), so the underlying bug stands. Given
// that, calling checkCmd.RunE in-process would os.Exit the whole `go test`
// binary -- taking every other test in this package down with it. Running
// it as a subprocess keeps the os.Exit call harmless: only the subprocess
// exits, and the parent test just reads what it printed first.
func TestCheckHelperProcess(t *testing.T) {
	if os.Getenv("BEANS_CHECK_HELPER") != "1" {
		t.Skip("only runs as a subprocess of runCheckInTestStore")
	}

	lipgloss.SetColorProfile(termenv.TrueColor)
	ui.SetTheme(os.Getenv("BEANS_CHECK_THEME"))

	beansDir := os.Getenv("BEANS_CHECK_DIR")
	testCfg := config.Default()
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		os.Stderr.WriteString("loading core: " + err.Error() + "\n")
		os.Exit(2)
	}
	core, cfg = testCore, testCfg

	_ = checkCmd.RunE(checkCmd, nil) // may os.Exit(1); that's fine, we're a subprocess
}

// runCheckInTestStore runs `beans check` against a small empty store, with
// the theme currently active in the calling (parent) test process, in a
// subprocess -- see TestCheckHelperProcess for why. It returns whatever the
// subprocess wrote to stdout and stderr.
func runCheckInTestStore(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	beansDir := filepath.Join(dir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("creating test .beans dir: %v", err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestCheckHelperProcess$")
	cmd.Env = append(os.Environ(),
		"BEANS_CHECK_HELPER=1",
		"BEANS_CHECK_DIR="+beansDir,
		"BEANS_CHECK_THEME="+ui.ActiveTheme().Name,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("running check subprocess: %v\noutput:\n%s", err, out)
		}
		// An *exec.ExitError just means checkCmd.RunE found issues and
		// exited 1, which -- given the standing bug above -- is the normal
		// case, not a test failure.
	}
	return string(out)
}

// TestCheckInheritsTheThemePalette proves check's output actually follows
// ui.SetTheme, the way ui.Success/ui.Danger/ui.Bold/ui.Warning were always
// meant to: check.go renders exclusively through those shared styles, never
// a colour of its own, so a theme switch must change what it prints.
func TestCheckInheritsTheThemePalette(t *testing.T) {
	t.Cleanup(func() { ui.SetTheme("mocha") })

	ui.SetTheme("mocha")
	mocha := runCheckInTestStore(t)
	ui.SetTheme("latte")
	latte := runCheckInTestStore(t)

	if mocha == latte {
		t.Error("check does not follow the theme; it should through ui.Success and friends")
	}
}

// TestCheckStaysAReportNotATable guards the report shape: check prints tick
// lines, not a bean listing with a TITLE column header.
func TestCheckStaysAReportNotATable(t *testing.T) {
	out := stripANSITest(runCheckInTestStore(t))
	if strings.Contains(out, "TITLE") {
		t.Errorf("check is a report of tick lines, not a bean listing:\n%s", out)
	}
	if !strings.Contains(out, "✓") {
		t.Errorf("check must keep its tick lines:\n%s", out)
	}
}
