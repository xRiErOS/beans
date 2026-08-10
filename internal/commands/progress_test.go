package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupProgressTest installs a throwaway core and default config into the
// package globals progressCmd.RunE reads, mirroring setupMilestonesTest.
func setupProgressTest(t *testing.T) {
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

// resetProgressFlags clears every progress* package global the tests below
// touch and restores the pre-test values afterwards.
func resetProgressFlags(t *testing.T) {
	t.Helper()
	oldJSON, oldParent := progressJSON, progressParent
	progressJSON, progressParent = false, ""
	t.Cleanup(func() {
		progressJSON, progressParent = oldJSON, oldParent
	})
}

// captureProgressStdout redirects os.Stdout for the duration of fn and
// returns everything written to it.
func captureProgressStdout(t *testing.T, fn func()) []byte {
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

// TestProgressCountsByStatus verifies plain-text output shows a count for
// every configured status (not just the four illustrated in the epic), and
// that the counts reflect the actual beans present.
func TestProgressCountsByStatus(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	beans := []*bean.Bean{
		{ID: "beans-a1", Slug: bean.Slugify("A"), Title: "A", Status: "in-progress", Type: "task"},
		{ID: "beans-a2", Slug: bean.Slugify("B"), Title: "B", Status: "todo", Type: "task"},
		{ID: "beans-a3", Slug: bean.Slugify("C"), Title: "C", Status: "todo", Type: "task"},
		{ID: "beans-a4", Slug: bean.Slugify("D"), Title: "D", Status: "draft", Type: "task"},
		{ID: "beans-a5", Slug: bean.Slugify("E"), Title: "E", Status: "completed", Type: "task"},
		{ID: "beans-a6", Slug: bean.Slugify("F"), Title: "F", Status: "scrapped", Type: "task"},
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	for _, want := range []string{
		"In Progress: 1",
		"Todo: 2",
		"Draft: 1",
		"Completed: 1",
		"Scrapped: 1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q, got %q", want, got)
		}
	}
}

// TestProgressComputesPercent verifies the epic's own worked example:
// 2 in-progress + 15 todo + 23 completed + 3 scrapped -> 23/(2+15+23) = 57%.
func TestProgressComputesPercent(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	seedProgressCounts(t, map[string]int{
		"in-progress": 2,
		"todo":        15,
		"completed":   23,
		"scrapped":    3,
	})

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	got := string(out)
	if !strings.Contains(got, "57% complete") {
		t.Errorf("expected output to contain %q, got %q", "57% complete", got)
	}
}

// TestProgressParentFlagScopesToDescendants verifies that --parent limits
// the counted beans to the given bean's descendants (via
// buildChildrenIndex/descendants), not the whole workspace.
func TestProgressParentFlagScopesToDescendants(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("Milestone"), Title: "Milestone", Status: "todo", Type: "milestone"}
	inScope := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("In scope"), Title: "In scope", Status: "completed", Type: "task", Parent: "beans-mile1"}
	outOfScope := &bean.Bean{ID: "beans-task2", Slug: bean.Slugify("Out of scope"), Title: "Out of scope", Status: "completed", Type: "task"}
	for _, b := range []*bean.Bean{milestone, inScope, outOfScope} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	progressParent = "beans-mile1"
	progressJSON = true

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	var result progressResult
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	// Only the milestone's one descendant (in-scope task) should be
	// counted; the unrelated top-level task must be excluded.
	if result.Total != 1 {
		t.Errorf("expected total=1 (scoped to descendants), got %d", result.Total)
	}
	if result.Completed != 1 {
		t.Errorf("expected completed=1, got %d", result.Completed)
	}
}

// TestProgressParentFlagErrorsOnUnknownID verifies that --parent with an
// ID that does not resolve to any bean returns an error instead of
// silently reporting 0/0 progress (resolver.Bean returns (nil, nil) for
// unknown IDs, so this must be checked explicitly).
func TestProgressParentFlagErrorsOnUnknownID(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)
	progressParent = "beans-doesnotexist"

	err := progressCmd.RunE(progressCmd, nil)
	if err == nil {
		t.Fatal("expected an error for an unknown --parent ID, got nil")
	}
}

// TestProgressJSONOutputHasNoBarString verifies that --json returns raw
// counts/percent only -- no bar/progress-bar string field, since bars are
// presentation-only.
func TestProgressJSONOutputHasNoBarString(t *testing.T) {
	setupProgressTest(t)
	resetProgressFlags(t)
	progressJSON = true

	task := &bean.Bean{ID: "beans-a1", Slug: bean.Slugify("A"), Title: "A", Status: "completed", Type: "task"}
	if err := core.Create(task); err != nil {
		t.Fatalf("core.Create error = %v", err)
	}

	out := captureProgressStdout(t, func() {
		if err := progressCmd.RunE(progressCmd, nil); err != nil {
			t.Fatalf("progressCmd.RunE() error = %v", err)
		}
	})

	if strings.Contains(string(out), "━") { // "━"
		t.Errorf("expected JSON output to NOT contain a bar string, got %q", out)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	for _, forbidden := range []string{"bar", "progressBar", "progress_bar"} {
		if _, ok := raw[forbidden]; ok {
			t.Errorf("expected JSON output to NOT contain a %q field, got %s", forbidden, out)
		}
	}
	for _, required := range []string{"counts", "completed", "total", "percent"} {
		if _, ok := raw[required]; !ok {
			t.Errorf("expected JSON output to contain a %q field, got %s", required, out)
		}
	}
}

// seedProgressCounts creates n beans of the given status for each entry in
// counts, so tests can assert on aggregate counts without caring about
// individual bean identity.
func seedProgressCounts(t *testing.T, counts map[string]int) {
	t.Helper()
	i := 0
	for status, n := range counts {
		for j := 0; j < n; j++ {
			i++
			id := fmt.Sprintf("beans-%s-%d", status, i)
			title := fmt.Sprintf("%s %d", status, i)
			b := &bean.Bean{ID: id, Slug: bean.Slugify(title), Title: title, Status: status, Type: "task"}
			if err := core.Create(b); err != nil {
				t.Fatalf("core.Create error = %v", err)
			}
		}
	}
}
