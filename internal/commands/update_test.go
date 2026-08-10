package commands

// Tests for parseLink and isKnownLinkType have been moved to content_test.go
// since those functions now live in content.go

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupUpdateTest installs a throwaway core and default config into the
// package globals updateCmd.RunE reads, and returns a bean already persisted
// in that core.
func setupUpdateTest(t *testing.T) *bean.Bean {
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

// resetUpdateFlags clears every update* package global the tests below touch
// and restores the pre-test values afterwards, so tests stay isolated despite
// sharing the updateCmd package-level flag vars.
func resetUpdateFlags(t *testing.T) {
	t.Helper()
	oldSet, oldUnset := updateSet, updateUnset
	updateSet, updateUnset = nil, nil
	t.Cleanup(func() {
		updateSet, updateUnset = oldSet, oldUnset
	})
}

// AC1/AC2/SC-01: --set writes extra front matter keys, and repeats accumulate.
func TestUpdateCmdSetWritesExtraKeys(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateSet = []string{"release=0-4-1", "klasse=bugfix"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Extra["release"] != "0-4-1" {
		t.Errorf("Extra[release] = %v, want %q", got.Extra["release"], "0-4-1")
	}
	if got.Extra["klasse"] != "bugfix" {
		t.Errorf("Extra[klasse] = %v, want %q", got.Extra["klasse"], "bugfix")
	}
}

// AC3: --unset removes a key that is present.
func TestUpdateCmdUnsetRemovesExtraKey(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateSet = []string{"release=0-4-1"}
	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() (set) error = %v", err)
	}
	resetUpdateFlags(t)

	updateSet = nil
	updateUnset = []string{"release"}
	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() (unset) error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if _, ok := got.Extra["release"]; ok {
		t.Errorf("expected release to be removed, Extra = %#v", got.Extra)
	}
}

// AC5: --unset of a key the bean does not carry leaves the bean unchanged
// and exits zero.
func TestUpdateCmdUnsetAbsentKeyIsNoop(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateUnset = []string{"nope"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if len(got.Extra) != 0 {
		t.Errorf("expected no Extra keys, got %#v", got.Extra)
	}
}

// AC4: --set on a reserved key fails naming the native flag, and the bean is
// left unchanged.
func TestUpdateCmdSetReservedKeyFails(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateSet = []string{"status=done"}

	err := updateCmd.RunE(updateCmd, []string{b.ID})
	if err == nil {
		t.Fatal("expected error for reserved key")
	}
	if !contains(err.Error(), "--status") {
		t.Errorf("expected error to name --status, got %q", err.Error())
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Status != "todo" {
		t.Errorf("expected status to be unchanged, got %q", got.Status)
	}
}

// AC6: --set without "=" is a usage error.
func TestUpdateCmdSetWithoutEqualsFails(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateSet = []string{"release"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err == nil {
		t.Fatal("expected error for --set without '='")
	}
}

// Conflict resolution: a key named by both --set and --unset in the same
// invocation ends up unset (see applyExtraOps).
func TestUpdateCmdSetAndUnsetSameKey(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateSet = []string{"release=0-4-1"}
	updateUnset = []string{"release"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() error = %v", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if _, ok := got.Extra["release"]; ok {
		t.Errorf("expected release to be unset, Extra = %#v", got.Extra)
	}
}

// --set/--unset alone (no other update flags) is a valid, sufficient change.
func TestUpdateCmdSetAloneCountsAsChange(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	updateSet = []string{"release=0-4-1"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() error = %v", err)
	}
}
