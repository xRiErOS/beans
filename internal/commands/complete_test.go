package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupCompleteTest installs a throwaway core and default config into the
// package globals completeCmd.RunE reads, and returns a bean already persisted
// in that core.
func setupCompleteTest(t *testing.T) *bean.Bean {
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
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}
	return b
}

// resetCompleteFlags clears every complete* package global the tests below touch
// and restores the pre-test values afterwards, so tests stay isolated despite
// sharing the completeCmd package-level flag vars.
func resetCompleteFlags(t *testing.T) {
	t.Helper()
	oldJSON, oldSummary := completeJSON, completeSummary
	completeJSON, completeSummary = false, ""
	t.Cleanup(func() {
		completeJSON, completeSummary = oldJSON, oldSummary
	})
}

// TestCompleteSetsStatus verifies that beans complete <id> sets status to completed.
func TestCompleteSetsStatus(t *testing.T) {
	b := setupCompleteTest(t)
	resetCompleteFlags(t)

	if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("bean status = %q, want %q", got.Status, "completed")
	}
}

// TestCompleteWithSummaryAppendsSection verifies that --summary appends a properly formatted section.
func TestCompleteWithSummaryAppendsSection(t *testing.T) {
	b := setupCompleteTest(t)
	resetCompleteFlags(t)

	completeSummary = "Implemented via PR #42"
	if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("bean status = %q, want %q", got.Status, "completed")
	}
	if !strings.Contains(got.Body, "## Summary of Changes") {
		t.Errorf("body does not contain '## Summary of Changes', body = %q", got.Body)
	}
	if !strings.Contains(got.Body, "Implemented via PR #42") {
		t.Errorf("body does not contain summary text, body = %q", got.Body)
	}
}

// TestCompleteRejectsUnknownID verifies that unknown bean ID returns an error.
func TestCompleteRejectsUnknownID(t *testing.T) {
	setupCompleteTest(t)
	resetCompleteFlags(t)

	if err := completeCmd.RunE(completeCmd, []string{"beans-notfound"}); err == nil {
		t.Errorf("completeCmd.RunE() expected error for unknown ID, got nil")
	}
}

// TestCompleteJSONOutput verifies that --json flag produces JSON output.
func TestCompleteJSONOutput(t *testing.T) {
	b := setupCompleteTest(t)
	resetCompleteFlags(t)

	completeJSON = true
	if err := completeCmd.RunE(completeCmd, []string{b.ID}); err != nil {
		t.Fatalf("completeCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "completed" {
		t.Errorf("bean status = %q, want %q", got.Status, "completed")
	}
}
