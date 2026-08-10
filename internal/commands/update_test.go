package commands

// Tests for parseLink and isKnownLinkType have been moved to content_test.go
// since those functions now live in content.go

import (
	"bytes"
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

// B01 regression: --set alone (no other update flags, so no field-update
// write runs first) used to pass ifMatch=nil to the extra-key write,
// silently ignoring a caller-supplied --if-match entirely -- a stale or
// outright wrong etag was accepted just the same as a correct one. This is
// the same optimistic-concurrency contract --priority/--status already
// honor; --set must not be a silent side door around it.
//
// Asserts against the file on disk, not core.Get(): Core.Update takes b by
// pointer and applyExtraOps mutates that same pointer before the etag check
// runs, so c.beans[id] (and therefore core.Get) reflects the attempted
// mutation regardless of whether the write to disk was accepted -- this is
// documented in Core.Update's own comment and is exactly why it validates
// ifMatch against the on-disk content instead of the in-memory bean. The
// property this test protects is "the file was not overwritten," which only
// a fresh read off disk can show.
func TestUpdateCmdSetAloneRejectsStaleIfMatch(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	oldIfMatch := updateIfMatch
	updateIfMatch = "deadbeefdeadbeef" // not b's real etag
	t.Cleanup(func() { updateIfMatch = oldIfMatch })

	updateSet = []string{"release=0-4-1"}

	err := updateCmd.RunE(updateCmd, []string{b.ID})
	if err == nil {
		t.Fatal("expected an etag-mismatch error for a stale --if-match, got nil")
	}

	onDisk, readErr := readBeanFromDisk(t, b)
	if readErr != nil {
		t.Fatalf("reading bean from disk: %v", readErr)
	}
	if _, ok := onDisk.Extra["release"]; ok {
		t.Errorf("file on disk should be unchanged after a rejected stale --if-match, got Extra = %#v", onDisk.Extra)
	}
}

// readBeanFromDisk re-parses b's file directly off disk, bypassing Core's
// in-memory cache entirely -- see TestUpdateCmdSetAloneRejectsStaleIfMatch
// for why that distinction matters.
func readBeanFromDisk(t *testing.T, b *bean.Bean) (*bean.Bean, error) {
	t.Helper()
	data, err := os.ReadFile(core.FullPath(b))
	if err != nil {
		return nil, err
	}
	return bean.Parse(bytes.NewReader(data))
}

// Sanity counterpart to the regression above: the correct current --if-match
// still lets a --set-only update through.
func TestUpdateCmdSetAloneAcceptsCorrectIfMatch(t *testing.T) {
	b := setupUpdateTest(t)
	resetUpdateFlags(t)

	oldIfMatch := updateIfMatch
	updateIfMatch = b.ETag()
	t.Cleanup(func() { updateIfMatch = oldIfMatch })

	updateSet = []string{"release=0-4-1"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() error = %v (correct --if-match should be accepted)", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Extra["release"] != "0-4-1" {
		t.Errorf("Extra[release] = %v, want %q", got.Extra["release"], "0-4-1")
	}
}

// B01 regression, require_if_match:true variant: the extra-key write used to
// pass ifMatch=nil, which under require_if_match:true fails outright even
// though the caller never asked for optimistic locking to be skipped.
func TestUpdateCmdSetSucceedsUnderRequireIfMatch(t *testing.T) {
	tmpDir := t.TempDir()
	beansDir := tmpDir + "/.beans"
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}
	testCfg := config.Default()
	testCfg.Beans.RequireIfMatch = true
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}
	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	b := &bean.Bean{ID: "beans-test1", Slug: bean.Slugify("A test bean"), Title: "A test bean", Status: "todo", Type: "task"}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}

	resetUpdateFlags(t)
	oldIfMatch := updateIfMatch
	updateIfMatch = b.ETag()
	t.Cleanup(func() { updateIfMatch = oldIfMatch })

	updateSet = []string{"release=0-4-1"}

	if err := updateCmd.RunE(updateCmd, []string{b.ID}); err != nil {
		t.Fatalf("updateCmd.RunE() error = %v (a correct --if-match should satisfy require_if_match:true)", err)
	}

	got, err := core.Get(b.ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Extra["release"] != "0-4-1" {
		t.Errorf("Extra[release] = %v, want %q", got.Extra["release"], "0-4-1")
	}
}
