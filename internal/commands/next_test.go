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
	oldType, oldTag, oldParent := nextType, nextTag, nextParent
	oldSort, oldDesc, oldJSON := nextSort, nextDesc, nextJSON
	nextType, nextTag, nextParent = nil, nil, ""
	nextSort, nextDesc, nextJSON = "", false, false
	t.Cleanup(func() {
		nextType, nextTag, nextParent = oldType, oldTag, oldParent
		nextSort, nextDesc, nextJSON = oldSort, oldDesc, oldJSON
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

// mkReadyBean creates a ready (todo) bean in the test store and returns it.
func mkReadyBean(t *testing.T, id, title, beanType, priority string, tags []string, parent string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:       id,
		Slug:     bean.Slugify(title),
		Title:    title,
		Status:   "todo",
		Type:     beanType,
		Priority: priority,
		Tags:     tags,
		Parent:   parent,
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create(%s) error = %v", id, err)
	}
	return b
}

// TestNextFiltersByType verifies that --type narrows the candidate set the
// ready filter is applied to, rather than being ignored.
func TestNextFiltersByType(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	// The bug carries lower priority, so without the filter the task would
	// win — the filter has to beat the sort order, not merely agree with it.
	task := mkReadyBean(t, "beans-ft01", "High priority task", "task", "high", nil, "")
	bug := mkReadyBean(t, "beans-fb01", "Low priority bug", "bug", "low", nil, "")

	nextType = []string{"bug"}

	out := string(captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	}))

	if !strings.Contains(out, bug.ID) {
		t.Errorf("output does not contain filtered bean %q, output = %q", bug.ID, out)
	}
	if strings.Contains(out, task.ID) {
		t.Errorf("output unexpectedly contains excluded bean %q, output = %q", task.ID, out)
	}
}

// TestNextFiltersByTag verifies --tag narrows the candidate set.
func TestNextFiltersByTag(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	untagged := mkReadyBean(t, "beans-gu01", "High priority untagged", "task", "high", nil, "")
	tagged := mkReadyBean(t, "beans-gt01", "Low priority tagged", "task", "low", []string{"cli"}, "")

	nextTag = []string{"cli"}

	out := string(captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	}))

	if !strings.Contains(out, tagged.ID) {
		t.Errorf("output does not contain tagged bean %q, output = %q", tagged.ID, out)
	}
	if strings.Contains(out, untagged.ID) {
		t.Errorf("output unexpectedly contains untagged bean %q, output = %q", untagged.ID, out)
	}
}

// TestNextFiltersByParent verifies --parent narrows the candidate set.
func TestNextFiltersByParent(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	epic := mkReadyBean(t, "beans-pe01", "Epic", "epic", "normal", nil, "")
	outside := mkReadyBean(t, "beans-po01", "High priority outside", "task", "high", nil, "")
	inside := mkReadyBean(t, "beans-pi01", "Low priority inside", "task", "low", nil, epic.ID)

	nextParent = epic.ID

	out := string(captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	}))

	if !strings.Contains(out, inside.ID) {
		t.Errorf("output does not contain child bean %q, output = %q", inside.ID, out)
	}
	if strings.Contains(out, outside.ID) {
		t.Errorf("output unexpectedly contains bean outside the parent %q, output = %q", outside.ID, out)
	}
}

// TestNextCombinesFiltersWithReadySemantics verifies that the new filters
// compose with each other and do not disable the ready filter.
func TestNextCombinesFiltersWithReadySemantics(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	epic := mkReadyBean(t, "beans-ce01", "Epic", "epic", "normal", nil, "")
	// Matches every filter and outranks the winner on priority, but is
	// in-progress and therefore not ready.
	inProgress := &bean.Bean{
		ID:       "beans-cp01",
		Slug:     bean.Slugify("In progress match"),
		Title:    "In progress match",
		Status:   "in-progress",
		Type:     "bug",
		Priority: "high",
		Tags:     []string{"cli"},
		Parent:   epic.ID,
	}
	if err := core.Create(inProgress); err != nil {
		t.Fatalf("core.Create(inProgress) error = %v", err)
	}
	// Matches type and tag but hangs under no parent.
	wrongParent := mkReadyBean(t, "beans-cw01", "Wrong parent", "bug", "high", []string{"cli"}, "")
	want := mkReadyBean(t, "beans-cm01", "Full match", "bug", "low", []string{"cli"}, epic.ID)

	nextType = []string{"bug"}
	nextTag = []string{"cli"}
	nextParent = epic.ID

	out := string(captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	}))

	if !strings.Contains(out, want.ID) {
		t.Errorf("output does not contain the fully matching bean %q, output = %q", want.ID, out)
	}
	for _, unwanted := range []*bean.Bean{inProgress, wrongParent} {
		if strings.Contains(out, unwanted.ID) {
			t.Errorf("output unexpectedly contains %q, output = %q", unwanted.ID, out)
		}
	}
}

// TestNextSortPicksByRequestedField verifies that --sort changes which bean
// next hands back, rather than being accepted and ignored.
func TestNextSortPicksByRequestedField(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	// Priority order and ID order disagree, so whichever bean comes back
	// identifies which ordering actually ran.
	first := mkReadyBean(t, "beans-aaa1", "Alphabetically first, low priority", "task", "low", nil, "")
	high := mkReadyBean(t, "beans-zzz1", "Alphabetically last, high priority", "task", "high", nil, "")

	nextSort = "id"

	out := string(captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	}))

	if !strings.Contains(out, first.ID) {
		t.Errorf("--sort id did not pick the lowest ID %q, output = %q", first.ID, out)
	}
	if strings.Contains(out, high.ID) {
		t.Errorf("--sort id unexpectedly picked %q, output = %q", high.ID, out)
	}
}

// TestNextDescReversesSelection verifies --desc flips which end of the sorted
// list next takes its answer from.
func TestNextDescReversesSelection(t *testing.T) {
	setupNextTest(t)
	resetNextFlags(t)

	first := mkReadyBean(t, "beans-aaa2", "Alphabetically first", "task", "normal", nil, "")
	last := mkReadyBean(t, "beans-zzz2", "Alphabetically last", "task", "normal", nil, "")

	nextSort = "id"
	nextDesc = true

	out := string(captureNextStdout(t, func() {
		if err := nextCmd.RunE(nextCmd, nil); err != nil {
			t.Fatalf("nextCmd.RunE() error = %v", err)
		}
	}))

	if !strings.Contains(out, last.ID) {
		t.Errorf("--sort id --desc did not pick the highest ID %q, output = %q", last.ID, out)
	}
	if strings.Contains(out, first.ID) {
		t.Errorf("--sort id --desc unexpectedly picked %q, output = %q", first.ID, out)
	}
}

// TestNextReportsNoneReadyWithFilters verifies that an empty result caused by
// a filter says so and names the filter, instead of reading like "there is no
// work left".
func TestNextReportsNoneReadyWithFilters(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		setupNextTest(t)
		resetNextFlags(t)
		mkReadyBean(t, "beans-nf01", "A ready task", "task", "high", nil, "")

		nextType = []string{"bug"}

		out := string(captureNextStdout(t, func() {
			if err := nextCmd.RunE(nextCmd, nil); err != nil {
				t.Fatalf("nextCmd.RunE() error = %v", err)
			}
		}))

		if !strings.Contains(out, "--type bug") {
			t.Errorf("empty-result message does not name the active filter, output = %q", out)
		}
	})

	t.Run("json", func(t *testing.T) {
		setupNextTest(t)
		resetNextFlags(t)
		mkReadyBean(t, "beans-nf02", "A ready task", "task", "high", nil, "")

		nextType = []string{"bug"}
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
			t.Errorf("expected success=true, got %+v; output = %s", resp, out)
		}
		if !strings.Contains(resp.Message, "--type bug") {
			t.Errorf("JSON message does not name the active filter, got %q", resp.Message)
		}
	})
}
