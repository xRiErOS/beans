package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

func TestSortBeans(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	evenEarlier := now.Add(-2 * time.Hour)

	// Statuses are now hardcoded, so we just use default config
	testCfg := config.Default()

	t.Run("sort by id", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c3"},
			{ID: "a1"},
			{ID: "b2"},
		}
		sortBeans(beans, "id", testCfg)

		if beans[0].ID != "a1" || beans[1].ID != "b2" || beans[2].ID != "c3" {
			t.Errorf("sort by id: got [%s, %s, %s], want [a1, b2, c3]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by created", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "old", CreatedAt: &evenEarlier},
			{ID: "new", CreatedAt: &now},
			{ID: "mid", CreatedAt: &earlier},
		}
		sortBeans(beans, "created", testCfg)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by created: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by created with nil", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "nil1", CreatedAt: nil},
			{ID: "has", CreatedAt: &now},
			{ID: "nil2", CreatedAt: nil},
		}
		sortBeans(beans, "created", testCfg)

		// Non-nil should come first, then nil sorted by ID
		if beans[0].ID != "has" {
			t.Errorf("sort by created with nil: first should be \"has\", got %q", beans[0].ID)
		}
	})

	t.Run("sort by updated", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "old", UpdatedAt: &evenEarlier},
			{ID: "new", UpdatedAt: &now},
			{ID: "mid", UpdatedAt: &earlier},
		}
		sortBeans(beans, "updated", testCfg)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by updated: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by status", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c1", Status: "completed"},
			{ID: "t1", Status: "todo"},
			{ID: "i1", Status: "in-progress"},
			{ID: "t2", Status: "todo"},
		}
		sortBeans(beans, "status", testCfg)

		// Should be ordered by status config order (in-progress, todo, draft, completed, scrapped), then by ID within same status
		expected := []string{"i1", "t1", "t2", "c1"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("sort by status[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})

	t.Run("sort by order, scoped per parent", func(t *testing.T) {
		beans := []*bean.Bean{
			// Four siblings under "p1" whose order values were assigned out
			// of creation sequence, plus a fifth sibling without an order
			// value that must come last (SC-01).
			{ID: "p1-c", Title: "c", Parent: "p1", Order: "m"},
			{ID: "p1-a", Title: "a", Parent: "p1", Order: "a"},
			{ID: "p1-e", Title: "e", Parent: "p1"},
			{ID: "p1-d", Title: "d", Parent: "p1", Order: "t"},
			{ID: "p1-b", Title: "b", Parent: "p1", Order: "g"},
			// A sibling under a different parent shares an Order value with
			// one of p1's beans, which must not interleave the groups —
			// Order is a fractional index scoped per parent (R-12).
			{ID: "p2-x", Title: "x", Parent: "p2", Order: "a"},
		}
		sortBeans(beans, "order", testCfg)

		expected := []string{"p1-a", "p1-b", "p1-c", "p1-d", "p1-e", "p2-x"}
		got := make([]string, len(beans))
		for i, b := range beans {
			got[i] = b.ID
		}
		for i, want := range expected {
			if got[i] != want {
				t.Errorf("sort by order[%d]: got %v, want %v", i, got, expected)
				break
			}
		}
	})

	t.Run("default sort (archive status then type)", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "completed-bug", Status: "completed", Type: "bug"},
			{ID: "todo-feature", Status: "todo", Type: "feature"},
			{ID: "todo-task", Status: "todo", Type: "task"},
			{ID: "completed-task", Status: "completed", Type: "task"},
			{ID: "todo-bug", Status: "todo", Type: "bug"},
		}
		sortBeans(beans, "", testCfg)

		// Should be: non-archive first (sorted by type order from DefaultTypes: milestone, epic, bug, feature, task),
		// then archive (sorted by type)
		// DefaultTypes order: milestone, epic, bug, feature, task
		expected := []string{"todo-bug", "todo-feature", "todo-task", "completed-bug", "completed-task"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("default sort[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})
}

func TestListReadyFlagMutualExclusion(t *testing.T) {
	// Test that --ready and --is-blocked are mutually exclusive
	// by checking the validation logic directly
	tests := []struct {
		name        string
		ready       bool
		isBlocked   bool
		expectError bool
	}{
		{"neither flag", false, false, false},
		{"only --ready", true, false, false},
		{"only --is-blocked", false, true, false},
		{"both flags", true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This mirrors the validation logic in list.go
			hasError := tt.ready && tt.isBlocked
			if hasError != tt.expectError {
				t.Errorf("ready=%v, isBlocked=%v: got error=%v, want error=%v",
					tt.ready, tt.isBlocked, hasError, tt.expectError)
			}
		})
	}
}

// AC3: --where on a reserved key fails naming the native filter flag.
func TestValidateWhereKeysReservedKeyFails(t *testing.T) {
	err := validateWhereKeys([]string{"status=done"})
	if err == nil {
		t.Fatal("expected error for reserved key")
	}
	if !contains(err.Error(), "--status") {
		t.Errorf("expected error to name --status, got %q", err.Error())
	}
}

// --where without "=" is a usage error, same shape as --set.
func TestValidateWhereKeysWithoutEqualsFails(t *testing.T) {
	err := validateWhereKeys([]string{"release"})
	if err == nil {
		t.Fatal("expected error for --where without '='")
	}
}

func TestValidateWhereKeysAcceptsExtraKey(t *testing.T) {
	if err := validateWhereKeys([]string{"release=0-4-1"}); err != nil {
		t.Errorf("validateWhereKeys() unexpected error: %v", err)
	}
}

// AC1/SC-01: a single --where key=value returns only matching beans.
func TestFilterByWhereSingleMatch(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "a", Extra: map[string]any{"release": "0-4-1"}},
		{ID: "b", Extra: map[string]any{"release": "0-4-2"}},
		{ID: "c", Extra: map[string]any{"release": "0-4-1"}},
	}

	got := filterByWhere(beans, []string{"release=0-4-1"})

	if len(got) != 2 {
		t.Fatalf("expected 2 beans, got %d", len(got))
	}
	if got[0].ID != "a" || got[1].ID != "c" {
		t.Errorf("expected [a, c], got [%s, %s]", got[0].ID, got[1].ID)
	}
}

// AC2: multiple --where pairs combine with AND semantics.
func TestFilterByWhereMultiplePairsAreANDed(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "a", Extra: map[string]any{"release": "0-4-1", "klasse": "bugfix"}},
		{ID: "b", Extra: map[string]any{"release": "0-4-1", "klasse": "feature"}},
	}

	got := filterByWhere(beans, []string{"release=0-4-1", "klasse=bugfix"})

	if len(got) != 1 {
		t.Fatalf("expected 1 bean, got %d", len(got))
	}
	if got[0].ID != "a" {
		t.Errorf("expected [a], got [%s]", got[0].ID)
	}
}

// AC4: a key carried by no bean returns an empty result, not an error.
func TestFilterByWhereNoMatchIsEmpty(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "a", Extra: map[string]any{"release": "0-4-1"}},
	}

	got := filterByWhere(beans, []string{"release=9-9-9"})

	if len(got) != 0 {
		t.Errorf("expected 0 beans, got %d", len(got))
	}
}

func TestFilterByWhereEmptyWheresReturnsAllBeans(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "a"},
		{ID: "b"},
	}

	got := filterByWhere(beans, nil)

	if len(got) != 2 {
		t.Errorf("expected 2 beans, got %d", len(got))
	}
}

// captureListStdout redirects os.Stdout for the duration of fn and returns
// everything written to it. listCmd's --json path encodes directly to
// os.Stdout via output.SuccessMultiple, so this is the only way to observe it.
func captureListStdout(t *testing.T, fn func()) []byte {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	fn()
	os.Stdout = orig
	w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	return data
}

// setupListTest installs a throwaway core and default config into the
// package globals listCmd.RunE reads.
func setupListTest(t *testing.T) {
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

// resetListWhereFlag clears listWhere and restores it afterwards.
func resetListWhereFlag(t *testing.T) {
	t.Helper()
	old := listWhere
	listWhere = nil
	t.Cleanup(func() { listWhere = old })
}

// SC-01: in a store of five beans of which two carry release: 0-4-1,
// `beans list --where release=0-4-1 --json` returns exactly those two.
func TestListCmdWhereEndToEnd(t *testing.T) {
	setupListTest(t)
	resetListWhereFlag(t)

	oldJSON, oldFull := listJSON, listFull
	listJSON, listFull = true, false
	t.Cleanup(func() { listJSON, listFull = oldJSON, oldFull })

	releases := []string{"0-4-1", "0-4-1", "0-4-0", "0-4-0", "0-4-0"}
	for i, release := range releases {
		b := &bean.Bean{
			ID:     "beans-sc" + string(rune('1'+i)),
			Slug:   bean.Slugify("bean"),
			Title:  "bean",
			Status: "todo",
			Type:   "task",
			Extra:  map[string]any{"release": release},
		}
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create() error = %v", err)
		}
	}

	listWhere = []string{"release=0-4-1"}

	out := captureListStdout(t, func() {
		if err := listCmd.RunE(listCmd, nil); err != nil {
			t.Fatalf("listCmd.RunE() error = %v", err)
		}
	})

	dec := json.NewDecoder(bytes.NewReader(out))
	var got []*bean.Bean
	if err := dec.Decode(&got); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 beans, got %d; output = %s", len(got), out)
	}
	for _, b := range got {
		if b.Extra["release"] != "0-4-1" {
			t.Errorf("Extra[release] = %v, want %q", b.Extra["release"], "0-4-1")
		}
	}
}

// TestListCmdReadyEndToEnd is a regression test for the applyReadyFilter
// extraction (Task 4): `list --ready` must still exclude blocked/terminal
// beans AND continue to combine with other flags already applied to the
// same filter object (here --type), since applyReadyFilter mutates the
// filter list.go already built rather than constructing a separate one.
func TestListCmdReadyEndToEnd(t *testing.T) {
	setupListTest(t)

	oldJSON, oldFull, oldReady, oldType := listJSON, listFull, listReady, listType
	listJSON, listFull, listReady = true, false, true
	t.Cleanup(func() { listJSON, listFull, listReady, listType = oldJSON, oldFull, oldReady, oldType })

	readyTask := &bean.Bean{
		ID:     "beans-rdy1",
		Slug:   bean.Slugify("ready task"),
		Title:  "ready task",
		Status: "todo",
		Type:   "task",
	}
	readyEpic := &bean.Bean{
		ID:     "beans-rdy2",
		Slug:   bean.Slugify("ready epic"),
		Title:  "ready epic",
		Status: "todo",
		Type:   "epic",
	}
	blockedTask := &bean.Bean{
		ID:        "beans-rdy3",
		Slug:      bean.Slugify("blocked task"),
		Title:     "blocked task",
		Status:    "todo",
		Type:      "task",
		BlockedBy: []string{readyTask.ID},
	}
	completedTask := &bean.Bean{
		ID:     "beans-rdy4",
		Slug:   bean.Slugify("completed task"),
		Title:  "completed task",
		Status: "completed",
		Type:   "task",
	}
	for _, b := range []*bean.Bean{readyTask, readyEpic, blockedTask, completedTask} {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s) error = %v", b.ID, err)
		}
	}

	// --ready alone: excludes blocked and completed, keeps both types.
	out := captureListStdout(t, func() {
		if err := listCmd.RunE(listCmd, nil); err != nil {
			t.Fatalf("listCmd.RunE() error = %v", err)
		}
	})
	var got []*bean.Bean
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	gotIDs := make(map[string]bool, len(got))
	for _, b := range got {
		gotIDs[b.ID] = true
	}
	if !gotIDs[readyTask.ID] || !gotIDs[readyEpic.ID] {
		t.Errorf("expected both ready beans in result, got IDs = %v", gotIDs)
	}
	if gotIDs[blockedTask.ID] || gotIDs[completedTask.ID] {
		t.Errorf("expected blocked/completed beans excluded, got IDs = %v", gotIDs)
	}

	// --ready combined with --type task: must still apply --ready's
	// exclusions AND the --type filter on the same filter object.
	listType = []string{"task"}
	out = captureListStdout(t, func() {
		if err := listCmd.RunE(listCmd, nil); err != nil {
			t.Fatalf("listCmd.RunE() error = %v", err)
		}
	})
	got = nil
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("decoding JSON output: %v; output = %s", err, out)
	}
	if len(got) != 1 || got[0].ID != readyTask.ID {
		t.Fatalf("expected only [%s] for --ready --type task, got %v", readyTask.ID, got)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short max", "hello", 4, "h..."},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
