package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupStartTest installs a throwaway core and default config into the
// package globals startCmd.RunE reads, and returns a bean already persisted
// in that core.
func setupStartTest(t *testing.T) *bean.Bean {
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

	b := &bean.Bean{
		ID:     "beans-test1",
		Slug:   bean.Slugify("A test bean"),
		Title:  "A test bean",
		Status: "todo",
		Type:   "task",
		Body:   "Test body content",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}
	return b
}

// resetStartFlags clears every start* package global the tests below touch
// and restores the pre-test values afterwards, so tests stay isolated despite
// sharing the startCmd package-level flag vars.
func resetStartFlags(t *testing.T) {
	t.Helper()
	oldJSON := startJSON
	startJSON = false
	t.Cleanup(func() {
		startJSON = oldJSON
	})
}

// TestStartSetsStatusInProgress verifies that beans start <id> sets status to in-progress.
func TestStartSetsStatusInProgress(t *testing.T) {
	b := setupStartTest(t)
	resetStartFlags(t)

	if err := startCmd.RunE(startCmd, []string{b.ID}); err != nil {
		t.Fatalf("startCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "in-progress" {
		t.Errorf("bean status = %q, want %q", got.Status, "in-progress")
	}
}

// TestStartDisplaysBeanDetails verifies that start displays the bean details
// by delegating to show command.
func TestStartDisplaysBeanDetails(t *testing.T) {
	b := setupStartTest(t)
	resetStartFlags(t)

	// Capture stdout to verify display output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	runErr := startCmd.RunE(startCmd, []string{b.ID})

	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}

	captured, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}

	if runErr != nil {
		t.Fatalf("startCmd.RunE() error = %v", runErr)
	}

	output := string(captured)

	// Verify output contains bean ID and title (show command output includes these)
	if !strings.Contains(output, b.ID) {
		t.Errorf("output does not contain bean ID %q, output = %q", b.ID, output)
	}
	if !strings.Contains(output, b.Title) {
		t.Errorf("output does not contain bean title %q, output = %q", b.Title, output)
	}
	if !strings.Contains(output, "in-progress") {
		t.Errorf("output does not contain status 'in-progress', output = %q", output)
	}
}

// TestStartRejectsUnknownID verifies that unknown bean ID returns an error.
func TestStartRejectsUnknownID(t *testing.T) {
	setupStartTest(t)
	resetStartFlags(t)

	if err := startCmd.RunE(startCmd, []string{"beans-notfound"}); err == nil {
		t.Errorf("startCmd.RunE() expected error for unknown ID, got nil")
	}
}
