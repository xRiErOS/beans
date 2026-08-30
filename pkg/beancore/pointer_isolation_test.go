package beancore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// liveBean returns the pointer the core actually stores, which is what the
// isolation tests compare against. Reading its fields outside the lock is
// exactly the race these tests guard, and is only safe here because the
// tests are single-goroutine.
func liveBean(tb testing.TB, c *Core, id string) *bean.Bean {
	tb.Helper()
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.beans[id]
	if !ok {
		tb.Fatalf("bean %s is not in the store", id)
	}
	return b
}

// TestGetReturnsIndependentCopy pins the fix for beans-x4ex: Get() must hand
// out a snapshot, not the live struct kept in the map.
func TestGetReturnsIndependentCopy(t *testing.T) {
	core := benchCore(t, 3)
	live := liveBean(t, core, "beans-b0000")

	got, err := core.Get("beans-b0000")
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got == live {
		t.Fatal("Get() returned the live bean pointer, want a copy")
	}

	got.Title = "mutated"
	got.Tags[0] = "mutated"
	got.Extra["release"] = "mutated"
	got.Blocking = append(got.Blocking, "beans-b0001")

	if live.Title == "mutated" {
		t.Error("mutating the Get() result's Title reached the stored bean")
	}
	if live.Tags[0] == "mutated" {
		t.Error("mutating the Get() result's Tags reached the stored bean")
	}
	if live.Extra["release"] == "mutated" {
		t.Error("mutating the Get() result's Extra reached the stored bean")
	}
	if len(live.Blocking) != 0 {
		t.Errorf("appending to the Get() result's Blocking reached the stored bean: %v", live.Blocking)
	}
	if got.ETag() != live.ETag() {
		t.Errorf("Get() ETag = %q, want %q", got.ETag(), live.ETag())
	}
}

// TestGetWithShortIDReturnsIndependentCopy covers the second return path in
// Get(): the prefix-prepended match.
func TestGetWithShortIDReturnsIndependentCopy(t *testing.T) {
	beansDir := filepath.Join(t.TempDir(), BeansDir)
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("creating test .beans dir: %v", err)
	}
	core := New(beansDir, config.DefaultWithPrefix("beans-"))
	core.SetWarnWriter(nil)
	if err := core.Load(); err != nil {
		t.Fatalf("core.Load() error = %v", err)
	}
	b := &bean.Bean{
		ID:     "beans-short1",
		Slug:   "short-bean",
		Title:  "short bean",
		Status: "todo",
		Type:   "task",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}
	live := liveBean(t, core, "beans-short1")

	got, err := core.Get("short1")
	if err != nil {
		t.Fatalf("core.Get(\"short1\") error = %v", err)
	}
	if got == live {
		t.Fatal("Get() with short ID returned the live bean pointer, want a copy")
	}

	got.Title = "mutated"
	if live.Title == "mutated" {
		t.Error("mutating the short-ID Get() result reached the stored bean")
	}
}

// TestUpdateDoesNotStoreTheCallerPointer pins the fix for beans-8p3h: Update()
// must store a copy of the caller's bean, not the caller's pointer itself.
func TestUpdateDoesNotStoreTheCallerPointer(t *testing.T) {
	core := benchCore(t, 1)
	id := "beans-b0000"

	b, err := core.Get(id)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	b.Title = "updated"
	if err := core.Update(b, nil); err != nil {
		t.Fatalf("core.Update() error = %v", err)
	}

	live := liveBean(t, core, id)
	if live == b {
		t.Fatal("Update() stored the caller's pointer, want a copy")
	}
	if live.Title != "updated" {
		t.Errorf("live.Title = %q, want %q", live.Title, "updated")
	}

	b.Title = "after"
	if live.Title != "updated" {
		t.Errorf("mutating the caller's bean after Update() reached the store: live.Title = %q", live.Title)
	}
}

// TestCreateDoesNotStoreTheCallerPointer is the same shape around Create().
func TestCreateDoesNotStoreTheCallerPointer(t *testing.T) {
	core := benchCore(t, 0)
	b := &bean.Bean{
		ID:     "beans-create1",
		Slug:   "create-bean",
		Title:  "create bean",
		Status: "todo",
		Type:   "task",
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}

	live := liveBean(t, core, b.ID)
	if live == b {
		t.Fatal("Create() stored the caller's pointer, want a copy")
	}

	b.Title = "after"
	if live.Title != "create bean" {
		t.Errorf("mutating the caller's bean after Create() reached the store: live.Title = %q", live.Title)
	}
}

// TestFindIncomingLinksReturnsCopies covers the reverse-index path used by
// BeanChildren and BeanBlockedBy.
func TestFindIncomingLinksReturnsCopies(t *testing.T) {
	core := benchCore(t, 0)
	parent := &bean.Bean{ID: "beans-parent1", Slug: "parent", Title: "parent", Status: "todo", Type: "feature"}
	if err := core.Create(parent); err != nil {
		t.Fatalf("core.Create(parent) error = %v", err)
	}
	child := &bean.Bean{ID: "beans-child1", Slug: "child", Title: "child", Status: "todo", Type: "task", Parent: parent.ID}
	if err := core.Create(child); err != nil {
		t.Fatalf("core.Create(child) error = %v", err)
	}

	liveChild := liveBean(t, core, child.ID)

	links := core.FindIncomingLinks(parent.ID)
	found := false
	for _, l := range links {
		if l.FromBean.ID == child.ID {
			found = true
			if l.FromBean == liveChild {
				t.Fatal("FindIncomingLinks() returned the live bean pointer, want a copy")
			}
			l.FromBean.Title = "mutated"
		}
	}
	if !found {
		t.Fatalf("FindIncomingLinks(%s) did not return %s", parent.ID, child.ID)
	}
	if liveChild.Title == "mutated" {
		t.Error("mutating a FindIncomingLinks() result reached the stored bean")
	}
}

// TestFindBlockersReturnCopies covers FindActiveBlockers and FindAllBlockers.
func TestFindBlockersReturnCopies(t *testing.T) {
	core := benchCore(t, 0)
	blocker := &bean.Bean{ID: "beans-blocker1", Slug: "blocker", Title: "blocker", Status: "todo", Type: "task"}
	if err := core.Create(blocker); err != nil {
		t.Fatalf("core.Create(blocker) error = %v", err)
	}
	a := &bean.Bean{ID: "beans-a1", Slug: "a", Title: "a", Status: "todo", Type: "task", BlockedBy: []string{blocker.ID}}
	if err := core.Create(a); err != nil {
		t.Fatalf("core.Create(a) error = %v", err)
	}

	liveBlocker := liveBean(t, core, blocker.ID)

	for _, got := range core.FindActiveBlockers(a.ID) {
		if got == liveBlocker {
			t.Fatal("FindActiveBlockers() returned the live bean pointer, want a copy")
		}
	}
	for _, got := range core.FindAllBlockers(a.ID) {
		if got == liveBlocker {
			t.Fatal("FindAllBlockers() returned the live bean pointer, want a copy")
		}
	}
}

// TestLoadAndUnarchiveReturnsCopy covers both return branches of
// LoadAndUnarchive: the archived-then-restored path exercised here.
func TestLoadAndUnarchiveReturnsCopy(t *testing.T) {
	core := benchCore(t, 0)
	b := &bean.Bean{ID: "beans-arch1", Slug: "arch", Title: "arch", Status: "todo", Type: "task"}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create() error = %v", err)
	}
	if err := core.Archive(b.ID); err != nil {
		t.Fatalf("core.Archive() error = %v", err)
	}

	got, err := core.LoadAndUnarchive(b.ID)
	if err != nil {
		t.Fatalf("core.LoadAndUnarchive() error = %v", err)
	}
	live := liveBean(t, core, b.ID)
	if got == live {
		t.Fatal("LoadAndUnarchive() returned the live bean pointer, want a copy")
	}
}

// BenchmarkCoreGet measures the cost of the clone Get() now takes on every
// call, in proportion to which the step-4 existence-only call sites were
// dropped in favor of NormalizeID.
func BenchmarkCoreGet(b *testing.B) {
	core := benchCore(b, 1000)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := core.Get("beans-b0500"); err != nil {
			b.Fatalf("core.Get() error = %v", err)
		}
	}
}
