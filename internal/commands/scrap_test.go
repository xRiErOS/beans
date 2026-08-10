package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/hmans/beans/internal/output"
)

// setupScrapTest installs a throwaway core and default config into the
// package globals scrapCmd.RunE reads, and returns a bean already persisted
// in that core.
func setupScrapTest(t *testing.T) *bean.Bean {
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

// resetScrapFlags clears every scrap* package global the tests below touch
// and restores the pre-test values afterwards, so tests stay isolated despite
// sharing the scrapCmd package-level flag vars.
func resetScrapFlags(t *testing.T) {
	t.Helper()
	oldJSON, oldReason := scrapJSON, scrapReason
	scrapJSON, scrapReason = false, ""
	t.Cleanup(func() {
		scrapJSON, scrapReason = oldJSON, oldReason
	})
}

// TestScrapRequiresMissingReason verifies that omitting --reason errors without mutating the bean.
func TestScrapRequiresMissingReason(t *testing.T) {
	b := setupScrapTest(t)
	resetScrapFlags(t)

	if err := scrapCmd.RunE(scrapCmd, []string{b.ID}); err == nil {
		t.Errorf("scrapCmd.RunE() expected error for missing reason, got nil")
	}

	// Verify bean was not mutated
	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "todo" {
		t.Errorf("bean status = %q, want %q (should not have changed)", got.Status, "todo")
	}
}

// TestScrapRequiresEmptyReason verifies that --reason "" errors without mutating the bean.
func TestScrapRequiresEmptyReason(t *testing.T) {
	b := setupScrapTest(t)
	resetScrapFlags(t)

	scrapReason = ""
	if err := scrapCmd.RunE(scrapCmd, []string{b.ID}); err == nil {
		t.Errorf("scrapCmd.RunE() expected error for empty reason, got nil")
	}

	// Verify bean was not mutated
	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "todo" {
		t.Errorf("bean status = %q, want %q (should not have changed)", got.Status, "todo")
	}
}

// TestScrapSetsStatusAndAppendsReason verifies that --reason appends the reason section and sets status to scrapped.
func TestScrapSetsStatusAndAppendsReason(t *testing.T) {
	b := setupScrapTest(t)
	resetScrapFlags(t)

	scrapReason = "Superseded by beans-xyz"
	if err := scrapCmd.RunE(scrapCmd, []string{b.ID}); err != nil {
		t.Fatalf("scrapCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "scrapped" {
		t.Errorf("bean status = %q, want %q", got.Status, "scrapped")
	}
	if !strings.Contains(got.Body, "## Reason for Scrapping") {
		t.Errorf("body does not contain '## Reason for Scrapping', body = %q", got.Body)
	}
	if !strings.Contains(got.Body, "Superseded by beans-xyz") {
		t.Errorf("body does not contain reason text, body = %q", got.Body)
	}
}

// TestScrapRejectsUnknownID verifies that unknown bean ID returns an error.
func TestScrapRejectsUnknownID(t *testing.T) {
	setupScrapTest(t)
	resetScrapFlags(t)

	scrapReason = "Some reason"
	if err := scrapCmd.RunE(scrapCmd, []string{"beans-notfound"}); err == nil {
		t.Errorf("scrapCmd.RunE() expected error for unknown ID, got nil")
	}
}

// TestScrapJSONOutput verifies that --json flag produces valid JSON with expected shape.
func TestScrapJSONOutput(t *testing.T) {
	b := setupScrapTest(t)
	resetScrapFlags(t)

	// Capture stdout to verify JSON output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = w

	scrapJSON = true
	scrapReason = "Test reason for scrapping"
	runErr := scrapCmd.RunE(scrapCmd, []string{b.ID})

	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}

	captured, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}

	if runErr != nil {
		t.Fatalf("scrapCmd.RunE() error = %v", runErr)
	}

	// Parse the JSON output
	dec := json.NewDecoder(bytes.NewReader(captured))
	var resp output.Response
	if err := dec.Decode(&resp); err != nil {
		t.Fatalf("decoding JSON output error = %v; output = %s", err, captured)
	}

	// Verify the response structure
	if !resp.Success {
		t.Errorf("response success = false, want true")
	}
	if resp.Bean == nil {
		t.Errorf("response bean = nil, want *bean.Bean")
	}
	if resp.Bean != nil && resp.Bean.ID != b.ID {
		t.Errorf("response bean ID = %q, want %q", resp.Bean.ID, b.ID)
	}
	if resp.Bean != nil && resp.Bean.Status != "scrapped" {
		t.Errorf("response bean status = %q, want %q", resp.Bean.Status, "scrapped")
	}
	if resp.Message != "Bean scrapped" {
		t.Errorf("response message = %q, want %q", resp.Message, "Bean scrapped")
	}

	// Also verify the bean was actually persisted with correct status
	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "scrapped" {
		t.Errorf("persisted bean status = %q, want %q", got.Status, "scrapped")
	}
}
