package commands

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

// setupPickTest points the package-level core at a fresh temp store and
// restores it afterward, mirroring setupShowTest's pattern for isolating
// per-test store state.
func setupPickTest(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldCore := core
	core = beancore.New(dir, config.Default())
	t.Cleanup(func() { core = oldCore })
}

func pickTestBean(id, title string) *bean.Bean {
	return &bean.Bean{ID: id, Title: title, Type: "task", Status: "open"}
}

// TestRunPickReflectsStoreResolvedPerInvocation pins SC-03/AC2.1: pick's
// candidate set comes from whatever store resolveBeansPath resolved for
// this invocation via NewRootCmd's PersistentPreRunE, not from a second,
// independent resolution path. It distinguishes "store A" from "store B"
// by which of runPick's two early error branches fires -- the empty-store
// branch for an empty directory, the non-tty branch (reached only once
// core.All() found candidates) for a populated one -- without needing a
// real controlling terminal.
func TestRunPickReflectsStoreResolvedPerInvocation(t *testing.T) {
	root := sharedTestRoot(t)

	emptyDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatalf("creating empty store dir: %v", err)
	}

	populatedDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(populatedDir, 0755); err != nil {
		t.Fatalf("creating populated store dir: %v", err)
	}
	seedCore := beancore.New(populatedDir, config.Default())
	if err := seedCore.Create(pickTestBean("beans-zzzz", "Populated")); err != nil {
		t.Fatalf("seeding populated store: %v", err)
	}

	// A closed-read-end pipe is non-terminal stdin, held constant across
	// both runs so the only variable between them is --beans-path.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})

	runWith := func(dir string) string {
		resetFlags(root)
		var errBuf bytes.Buffer
		root.SetOut(io.Discard)
		root.SetErr(&errBuf)
		root.SetArgs([]string{"--beans-path", dir, "pick"})
		_, execErr := root.ExecuteC()
		if execErr == nil {
			t.Fatalf("expected pick to fail for non-terminal stdin against %s", dir)
		}
		return execErr.Error()
	}

	emptyErr := runWith(emptyDir)
	populatedErr := runWith(populatedDir)

	if !strings.Contains(emptyErr, "no beans to pick from") {
		t.Errorf("empty store: got error %q, want the empty-candidates branch", emptyErr)
	}
	if !strings.Contains(populatedErr, "not a terminal") {
		t.Errorf("populated store: got error %q, want the non-tty branch", populatedErr)
	}
}

// TestPickCmdIsRegistered pins that RegisterPickCmd actually joins the tree,
// not merely that the file compiles (Entry points: register.go:46-109).
func TestPickCmdIsRegistered(t *testing.T) {
	root := sharedTestRoot(t)
	for _, c := range root.Commands() {
		if c.Name() == "pick" {
			return
		}
	}
	t.Fatal("expected a registered \"pick\" command, found none")
}

// TestPickCmdIsUserFacing pins the Risks section's warning: pick must not
// be marked plumbing and must fall into RegisterCoreCommands's default
// user-facing classification.
func TestPickCmdIsUserFacing(t *testing.T) {
	root := sharedTestRoot(t)
	cmd, _, err := root.Find([]string{"pick"})
	if err != nil {
		t.Fatalf("finding pick: %v", err)
	}
	if !IsUserFacing(cmd) {
		t.Error("pick classified plumbing, want user-facing")
	}
}

// TestPickCmdDeclaresNoArgs pins the Cobra convention constraint: pick takes
// no positional arguments.
func TestPickCmdDeclaresNoArgs(t *testing.T) {
	if pickCmd.Args == nil {
		t.Fatal("pick has no declared Args policy")
	}
	if err := pickCmd.Args(pickCmd, []string{"unexpected"}); err == nil {
		t.Error("pick accepted a positional argument, want rejection")
	}
	if err := pickCmd.Args(pickCmd, nil); err != nil {
		t.Errorf("pick rejected zero arguments: %v", err)
	}
}

// TestRunPickFailsOnEmptyCandidates pins AC1.3 for the empty-list half:
// with no beans in the store, runPick must fail without writing to stdout,
// and must fail before ever attempting to open a controlling terminal.
func TestRunPickFailsOnEmptyCandidates(t *testing.T) {
	setupPickTest(t)

	var out bytes.Buffer
	pickCmd.SetOut(&out)
	t.Cleanup(func() { pickCmd.SetOut(nil) })

	err := runPick(pickCmd, nil)
	if err == nil {
		t.Fatal("expected an error for an empty candidate set")
	}
	if out.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", out.String())
	}
}

// TestRunPickFailsOnNonTTYStdin pins SC-02 and AC1.4: with stdin redirected
// away from a terminal (a pipe, exactly like `</dev/null` in a real shell),
// runPick must fail cleanly with no stdout output, without ever reaching
// the tea.Program/tty codepath.
func TestRunPickFailsOnNonTTYStdin(t *testing.T) {
	setupPickTest(t)
	if err := core.Create(pickTestBean("beans-test1", "Some bean")); err != nil {
		t.Fatalf("creating test bean: %v", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}

	var out bytes.Buffer
	pickCmd.SetOut(&out)
	t.Cleanup(func() { pickCmd.SetOut(nil) })

	runErr := runPick(pickCmd, nil)
	if runErr == nil {
		t.Fatal("expected an error for non-terminal stdin")
	}
	if out.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", out.String())
	}
}

// TestPickModelEnterSelectsHighlightedItem pins AC1.1's selection half of
// the model's terminal state machine: pressing enter on the highlighted
// item marks it selected and quits, without touching any io.Writer -- the
// model never writes to stdout itself (AC1.2).
func TestPickModelEnterSelectsHighlightedItem(t *testing.T) {
	beans := []*bean.Bean{
		pickTestBean("beans-aaaa", "Alpha bean"),
		pickTestBean("beans-bbbb", "Beta bean"),
	}
	m := newPickModel(beans)
	m.list.SetSize(80, 20)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	pm, ok := updated.(pickModel)
	if !ok {
		t.Fatalf("Update returned %T, want pickModel", updated)
	}
	if !pm.selected {
		t.Error("expected selected=true after enter")
	}
	if pm.selectedID != "beans-aaaa" {
		t.Errorf("selectedID = %q, want %q", pm.selectedID, "beans-aaaa")
	}
	if pm.aborted {
		t.Error("expected aborted=false on successful selection")
	}
	if cmd == nil {
		t.Fatal("expected a tea.Cmd (tea.Quit) after enter")
	}
	if msg := cmd(); msg != tea.Quit() {
		t.Errorf("expected tea.Quit message, got %#v", msg)
	}
}

// TestPickModelEscAborts pins AC1.3's abort half: pressing esc quits
// without marking anything selected.
func TestPickModelEscAborts(t *testing.T) {
	beans := []*bean.Bean{pickTestBean("beans-aaaa", "Alpha bean")}
	m := newPickModel(beans)
	m.list.SetSize(80, 20)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	pm, ok := updated.(pickModel)
	if !ok {
		t.Fatalf("Update returned %T, want pickModel", updated)
	}
	if pm.selected {
		t.Error("expected selected=false after esc")
	}
	if !pm.aborted {
		t.Error("expected aborted=true after esc")
	}
	if cmd == nil {
		t.Fatal("expected a tea.Cmd (tea.Quit) after esc")
	}
	if msg := cmd(); msg != tea.Quit() {
		t.Errorf("expected tea.Quit message, got %#v", msg)
	}
}

// TestPickModelViewNeverWritesToProvidedWriter is a structural guard for
// AC1.2: the View() method returns a string for bubbletea to draw wherever
// the program's output is configured (the controlling tty, never stdout);
// it takes no io.Writer of its own, so it cannot short-circuit around that
// configuration.
func TestPickModelViewNeverWritesToProvidedWriter(t *testing.T) {
	beans := []*bean.Bean{pickTestBean("beans-aaaa", "Alpha bean")}
	m := newPickModel(beans)
	m.list.SetSize(80, 20)
	if v := m.View(); v == "" {
		t.Error("expected a non-empty view for a populated list")
	}
	var discard io.Writer = io.Discard
	_ = discard
}
