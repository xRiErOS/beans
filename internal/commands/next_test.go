package commands

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupNextTest installs a throwaway core and default config into the
// package globals nextCmd.RunE reads, mirroring setupStartTest.
func setupNextTest(t *testing.T) {
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

// resetNextFlags clears every next* package global the tests below touch
// and restores the pre-test values afterwards.
func resetNextFlags(t *testing.T) {
	t.Helper()
	oldJSON := nextJSON
	nextJSON = false
	t.Cleanup(func() {
		nextJSON = oldJSON
	})
}

// captureNextStdout redirects os.Stdout for the duration of fn and returns
// everything written to it.
func captureNextStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = orig
	if err := w.Close(); err != nil {
		t.Fatalf("closing pipe write end: %v", err)
	}

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return data
}

// TestNextReturnsHighestPriorityReady verifies that beans next picks the
// highest-priority ready (todo, not blocked) bean and displays it.
func TestNextReturnsHighestPriorityReady(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	low := &bean.Bean{
		ID:       "beans-low1",
		Slug:     bean.Slugify("Low priority task"),
		Title:    "Low priority task",
		Status:   "todo",
		Type:     "task",
		Priority: "low",
	}
	high := &bean.Bean{
		ID:       "beans-high1",
		Slug:     bean.Slugify("High priority task"),
		Title:    "High priority task",
		Status:   "todo",
		Type:     "task",
		Priority: "high",
	}
	if err := core.Create(low); err != nil {
		t.Fatalf("core.Create(low) error = %v", err)
	}
	if err := core.Create(high); err != nil {
		t.Fatalf("core.Create(high) error = %v", err)
	}

	out := captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	if !strings.Contains(got, high.ID) {
		t.Errorf("output does not contain highest-priority bean ID %q, output = %q", high.ID, got)
	}
	if strings.Contains(got, low.ID) {
		t.Errorf("output unexpectedly contains lower-priority bean ID %q, output = %q", low.ID, got)
	}
}

// TestNextRespectsBlockedExclusion verifies that a blocked todo bean is
// skipped in favor of an unblocked one.
func TestNextRespectsBlockedExclusion(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	// blocker is itself a valid ready bean (low priority), so it must lose
	// the pick to the higher-priority unblocked bean below — this also
	// proves the winner isn't just an artifact of ID tie-breaking.
	blocker := &bean.Bean{
		ID:       "beans-blk1",
		Slug:     bean.Slugify("Blocker"),
		Title:    "Blocker",
		Status:   "todo",
		Type:     "task",
		Priority: "low",
	}
	if err := core.Create(blocker); err != nil {
		t.Fatalf("core.Create(blocker) error = %v", err)
	}

	// blocked carries the highest priority of the three but must be
	// excluded entirely because it is blocked by an active (todo) blocker.
	blocked := &bean.Bean{
		ID:        "beans-blkd1",
		Slug:      bean.Slugify("Blocked high priority"),
		Title:     "Blocked high priority",
		Status:    "todo",
		Type:      "task",
		Priority:  "high",
		BlockedBy: []string{blocker.ID},
	}
	if err := core.Create(blocked); err != nil {
		t.Fatalf("core.Create(blocked) error = %v", err)
	}

	unblocked := &bean.Bean{
		ID:       "beans-unbl1",
		Slug:     bean.Slugify("Unblocked high priority"),
		Title:    "Unblocked high priority",
		Status:   "todo",
		Type:     "task",
		Priority: "high",
	}
	if err := core.Create(unblocked); err != nil {
		t.Fatalf("core.Create(unblocked) error = %v", err)
	}

	out := captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	if strings.Contains(got, blocked.ID) {
		t.Errorf("output unexpectedly contains blocked bean ID %q, output = %q", blocked.ID, got)
	}
	if strings.Contains(got, blocker.ID) {
		t.Errorf("output unexpectedly contains lower-priority blocker bean ID %q, output = %q", blocker.ID, got)
	}
	if !strings.Contains(got, unblocked.ID) {
		t.Errorf("output does not contain unblocked bean ID %q, output = %q", unblocked.ID, got)
	}
}

// TestNextReportsNoneReady verifies that with no ready beans, next reports
// this without erroring, both in plain and --json mode.
func TestNextReportsNoneReady(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		setupNextTest(t)
		resetNextFlags(t)

		out := captureNextStdout(t, func() {
			if err := nextCmd.RunE(nextCmd, nil); err != nil {
				t.Fatalf("nextCmd.RunE() error = %v", err)
			}
		})

		got := string(out)
		if !strings.Contains(got, "beans list") {
			t.Errorf("expected suggestion to run beans list, output = %q", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		setupNextTest(t)
		resetNextFlags(t)
		nextJSON = true

		out := captureNextStdout(t, func() {
			if err := nextCmd.RunE(nextCmd, nil); err != nil {
				t.Fatalf("nextCmd.RunE() error = %v", err)
			}
		})

		var resp struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(out, &resp); err != nil {
			t.Fatalf("decoding JSON output: %v; output = %s", err, out)
		}
		if !resp.Success {
			t.Errorf("expected success=true for empty-result JSON, got %+v; output = %s", resp, out)
		}
		if resp.Message == "" {
			t.Errorf("expected a non-empty message for empty-result JSON, got %+v; output = %s", resp, out)
		}
	})
}
