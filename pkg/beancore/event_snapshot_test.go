package beancore

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestArchiveEventCarriesACopy pins the fix for beans-35rr on the Archive
// path. Archive() fans out synchronously (no fsnotify involved), so the
// event is deterministically ready as soon as Archive() returns.
func TestArchiveEventCarriesACopy(t *testing.T) {
	core, _ := setupTestCore(t)
	b := createTestBean(t, core, "arch-ev1", "archive event", "todo")

	ch, unsub := core.Subscribe()
	defer unsub()

	if err := core.Archive(b.ID); err != nil {
		t.Fatalf("core.Archive() error = %v", err)
	}

	select {
	case batch := <-ch:
		found := false
		for _, e := range batch {
			if e.BeanID == b.ID && e.Type == EventUpdated {
				found = true
				live := liveBean(t, core, b.ID)
				if e.Bean == live {
					t.Fatal("EventUpdated payload is the live bean pointer, want a copy")
				}
				e.Bean.Title = "mutated"
				if live.Title == "mutated" {
					t.Error("mutating the event payload reached the stored bean")
				}
			}
		}
		if !found {
			t.Fatalf("expected EventUpdated for %s, got: %+v", b.ID, batch)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for archive event")
	}
}

// TestWatcherEventCarriesACopy is the shape of TestSubscribe
// (pkg/beancore/core_test.go): a file watcher event must also carry a copy,
// both for the create and for a subsequent update.
func TestWatcherEventCarriesACopy(t *testing.T) {
	core, beansDir := setupTestCore(t)

	if err := core.StartWatching(); err != nil {
		t.Fatalf("StartWatching() error = %v", err)
	}
	defer core.Unwatch()

	ch, unsub := core.Subscribe()
	defer unsub()

	time.Sleep(50 * time.Millisecond)

	beanPath := filepath.Join(beansDir, "new1--new.md")
	content := "---\ntitle: New Bean\nstatus: todo\n---\n"
	if err := os.WriteFile(beanPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	select {
	case events := <-ch:
		found := false
		for _, e := range events {
			if e.Type == EventCreated && e.BeanID == "new1" {
				found = true
				if e.Bean == nil {
					t.Fatal("EventCreated should include Bean")
				}
				live := liveBean(t, core, "new1")
				if e.Bean == live {
					t.Fatal("EventCreated payload is the live bean pointer, want a copy")
				}
			}
		}
		if !found {
			t.Fatalf("expected EventCreated for new1, got: %+v", events)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for created event")
	}

	updated := "---\ntitle: New Bean Updated\nstatus: todo\n---\n"
	if err := os.WriteFile(beanPath, []byte(updated), 0644); err != nil {
		t.Fatalf("failed to overwrite test file: %v", err)
	}

	select {
	case events := <-ch:
		found := false
		for _, e := range events {
			if e.Type == EventUpdated && e.BeanID == "new1" {
				found = true
				live := liveBean(t, core, "new1")
				if e.Bean == live {
					t.Fatal("EventUpdated payload is the live bean pointer, want a copy")
				}
			}
		}
		if !found {
			t.Fatalf("expected EventUpdated for new1, got: %+v", events)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for updated event")
	}
}

// TestWorktreeWatcherEventCarriesACopy is the shape of
// pkg/beancore/worktree_watcher_test.go's "delete in worktree reverts to
// main-repo version": both the revert-to-main emit and the worktree
// updated/created emits must carry copies.
func TestWorktreeWatcherEventCarriesACopy(t *testing.T) {
	core, _ := setupTestCore(t)

	createTestBean(t, core, "wt-ev-1", "Original Title", "todo")

	wtDir := t.TempDir()
	wtBeansDir := filepath.Join(wtDir, BeansDir)
	if err := os.MkdirAll(wtBeansDir, 0755); err != nil {
		t.Fatalf("creating worktree .beans dir: %v", err)
	}

	content := "---\ntitle: Modified in Worktree\nstatus: in-progress\ntype: task\n---\n"
	beanPath := filepath.Join(wtBeansDir, "wt-ev-1--original-title.md")
	if err := os.WriteFile(beanPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing worktree bean file: %v", err)
	}

	if err := core.WatchWorktreeBeans(wtDir); err != nil {
		t.Fatalf("WatchWorktreeBeans() error = %v", err)
	}
	defer core.UnwatchWorktreeBeans(wtDir)

	time.Sleep(100 * time.Millisecond)

	events, unsub := core.Subscribe()
	defer unsub()

	// Overwrite the worktree file to trigger the worktree updated emit.
	updated := "---\ntitle: Modified Again\nstatus: in-progress\ntype: task\n---\n"
	if err := os.WriteFile(beanPath, []byte(updated), 0644); err != nil {
		t.Fatalf("overwriting worktree bean file: %v", err)
	}

	select {
	case batch := <-events:
		found := false
		for _, ev := range batch {
			if ev.BeanID == "wt-ev-1" {
				found = true
				live := liveBean(t, core, "wt-ev-1")
				if ev.Bean == live {
					t.Fatal("worktree update event payload is the live bean pointer, want a copy")
				}
			}
		}
		if !found {
			t.Fatal("expected an event for wt-ev-1 after worktree overwrite")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for worktree update event")
	}

	// Remove the worktree file to exercise the revert-to-main emit.
	if err := os.Remove(beanPath); err != nil {
		t.Fatalf("removing worktree bean file: %v", err)
	}

	select {
	case batch := <-events:
		found := false
		for _, ev := range batch {
			if ev.BeanID == "wt-ev-1" && ev.Type == EventUpdated {
				found = true
				live := liveBean(t, core, "wt-ev-1")
				if ev.Bean == live {
					t.Fatal("revert-to-main event payload is the live bean pointer, want a copy")
				}
				if ev.Bean.Title != "Original Title" {
					t.Errorf("reverted Title = %q, want %q", ev.Bean.Title, "Original Title")
				}
			}
		}
		if !found {
			t.Fatal("expected EventUpdated reverting to main-repo version")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for revert event")
	}
}

// TestEventPayloadsAreRaceFreeAgainstInPlaceWrites is the transposition of
// TestAllIsRaceFreeAgainstInPlaceWrites (all_snapshot_test.go): a subscriber
// reads event payloads after the lock is released while writers mutate the
// stored beans in place under the write lock. Meaningful under -race
// (`just test-race`).
func TestEventPayloadsAreRaceFreeAgainstInPlaceWrites(t *testing.T) {
	core := benchCore(t, 20)
	ch, unsub := core.Subscribe()

	var work sync.WaitGroup
	work.Add(2)

	go func() {
		defer work.Done()
		for i := range 10 {
			id := fmt.Sprintf("beans-b%04d", i)
			if err := core.Archive(id); err != nil {
				return
			}
		}
	}()

	go func() {
		defer work.Done()
		for range 200 {
			b, err := core.Get("beans-b0010")
			if err != nil {
				return
			}
			copied := b.Clone()
			copied.Blocking = []string{"beans-b0011"}
			if err := core.Update(copied, nil); err != nil {
				return
			}
			if _, err := core.RemoveLinksTo("beans-b0011"); err != nil {
				return
			}
		}
	}()

	var drain sync.WaitGroup
	drain.Add(1)
	go func() {
		defer drain.Done()
		for events := range ch {
			for _, e := range events {
				if e.Bean != nil {
					_ = fmt.Sprintf("%s %s %s %v %v %v", e.Bean.ID, e.Bean.Title, e.Bean.Status, e.Bean.Tags, e.Bean.Blocking, e.Bean.Extra)
				}
			}
		}
	}()

	work.Wait()
	unsub()
	drain.Wait()
}
