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

// setupMilestonesTest installs a throwaway core and default config into the
// package globals milestonesCmd.RunE reads, mirroring setupUpdateTest.
func setupMilestonesTest(t *testing.T) {
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

// resetMilestonesFlags clears every milestones* package global the tests
// below touch and restores the pre-test values afterwards.
func resetMilestonesFlags(t *testing.T) {
	t.Helper()
	oldJSON, oldAll := milestonesJSON, milestonesAll
	milestonesJSON, milestonesAll = false, false
	t.Cleanup(func() {
		milestonesJSON, milestonesAll = oldJSON, oldAll
	})
}

// captureMilestonesStdout redirects os.Stdout for the duration of fn and
// returns everything written to it.
func captureMilestonesStdout(t *testing.T, fn func()) []byte {
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

// TestMilestonesListsByType verifies that only Type == "milestone" beans
// appear in the output, not epics/tasks/features.
func TestMilestonesListsByType(t *testing.T) {
	setupMilestonesTest(t)
	resetMilestonesFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("A milestone"), Title: "A milestone", Status: "todo", Type: "milestone"}
	epic := &bean.Bean{ID: "beans-epic1", Slug: bean.Slugify("An epic"), Title: "An epic", Status: "todo", Type: "epic"}
	if err := core.Create(milestone); err != nil {
		t.Fatalf("core.Create(milestone) error = %v", err)
	}
	if err := core.Create(epic); err != nil {
		t.Fatalf("core.Create(epic) error = %v", err)
	}

	out := captureMilestonesStdout(t, func() {
		if err := milestonesCmd.RunE(milestonesCmd, nil); err != nil {
			t.Fatalf("milestonesCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	if !strings.Contains(got, "A milestone") {
		t.Errorf("expected output to contain milestone title, got %q", got)
	}
	if strings.Contains(got, "An epic") {
		t.Errorf("expected output to NOT contain epic title, got %q", got)
	}
}

// TestMilestonesShowsProgress verifies the "N/M completed" figure counts
// descendants (not just direct children), per the descendantProgress
// formula.
func TestMilestonesShowsProgress(t *testing.T) {
	setupMilestonesTest(t)
	resetMilestonesFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	epic := &bean.Bean{ID: "beans-epic1", Slug: bean.Slugify("Epic"), Title: "Epic", Status: "todo", Type: "epic", Parent: "beans-mile1"}
	doneTask := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("Done task"), Title: "Done task", Status: "completed", Type: "task", Parent: "beans-epic1"}
	todoTask := &bean.Bean{ID: "beans-task2", Slug: bean.Slugify("Todo task"), Title: "Todo task", Status: "todo", Type: "task", Parent: "beans-epic1"}
	for _, b := range []*bean.Bean{milestone, epic, doneTask, todoTask} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureMilestonesStdout(t, func() {
		if err := milestonesCmd.RunE(milestonesCmd, nil); err != nil {
			t.Fatalf("milestonesCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	// 3 descendants total (epic + 2 tasks), 1 completed.
	if !strings.Contains(got, "1/3") {
		t.Errorf("expected output to contain 1/3 progress, got %q", got)
	}
}

// TestMilestonesExcludesCompletedScrappedByDefault verifies that completed
// and scrapped milestones are hidden unless --all is passed.
func TestMilestonesExcludesCompletedScrappedByDefault(t *testing.T) {
	setupMilestonesTest(t)
	resetMilestonesFlags(t)

	active := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Active milestone"), Title: "Active milestone", Status: "todo", Type: "milestone"}
	done := &bean.Bean{ID: "beans-mile2", Slug: bean.Slugify("Done milestone"), Title: "Done milestone", Status: "completed", Type: "milestone"}
	scrapped := &bean.Bean{ID: "beans-mile3", Slug: bean.Slugify("Scrapped milestone"), Title: "Scrapped milestone", Status: "scrapped", Type: "milestone"}
	for _, b := range []*bean.Bean{active, done, scrapped} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureMilestonesStdout(t, func() {
		if err := milestonesCmd.RunE(milestonesCmd, nil); err != nil {
			t.Fatalf("milestonesCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	if !strings.Contains(got, "Active milestone") {
		t.Errorf("expected output to contain active milestone, got %q", got)
	}
	if strings.Contains(got, "Done milestone") {
		t.Errorf("expected output to NOT contain completed milestone by default, got %q", got)
	}
	if strings.Contains(got, "Scrapped milestone") {
		t.Errorf("expected output to NOT contain scrapped milestone by default, got %q", got)
	}
}

// TestMilestonesAllFlagIncludesThem verifies that --all includes
// completed/scrapped milestones.
func TestMilestonesAllFlagIncludesThem(t *testing.T) {
	setupMilestonesTest(t)
	resetMilestonesFlags(t)
	milestonesAll = true

	active := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Active milestone"), Title: "Active milestone", Status: "todo", Type: "milestone"}
	done := &bean.Bean{ID: "beans-mile2", Slug: bean.Slugify("Done milestone"), Title: "Done milestone", Status: "completed", Type: "milestone"}
	scrapped := &bean.Bean{ID: "beans-mile3", Slug: bean.Slugify("Scrapped milestone"), Title: "Scrapped milestone", Status: "scrapped", Type: "milestone"}
	for _, b := range []*bean.Bean{active, done, scrapped} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureMilestonesStdout(t, func() {
		if err := milestonesCmd.RunE(milestonesCmd, nil); err != nil {
			t.Fatalf("milestonesCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	for _, want := range []string{"Active milestone", "Done milestone", "Scrapped milestone"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q with --all, got %q", want, got)
		}
	}
}

// TestMilestonesJSONOutput verifies that --json emits a JSON array of
// {bean, completed, total} entries, decodable via real stdout capture.
func TestMilestonesJSONOutput(t *testing.T) {
	setupMilestonesTest(t)
	resetMilestonesFlags(t)
	milestonesJSON = true

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	task := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("Task"), Title: "Task", Status: "completed", Type: "task", Parent: "beans-mile1"}
	for _, b := range []*bean.Bean{milestone, task} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureMilestonesStdout(t, func() {
		if err := milestonesCmd.RunE(milestonesCmd, nil); err != nil {
			t.Fatalf("milestonesCmd.RunE() error = %v", err)
		}
	})

	var entries []struct {
		Bean      *bean.Bean `json:"bean"`
		Completed int        `json:"completed"`
		Total     int        `json:"total"`
	}
	if err := json.Unmarshal(out, &entries); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 milestone entry, got %d: %s", len(entries), out)
	}
	if entries[0].Bean == nil || entries[0].Bean.ID != milestone.ID {
		t.Errorf("expected bean %q in JSON entry, got %+v", milestone.ID, entries[0].Bean)
	}
	if entries[0].Completed != 1 || entries[0].Total != 1 {
		t.Errorf("expected completed=1 total=1, got completed=%d total=%d", entries[0].Completed, entries[0].Total)
	}
}
