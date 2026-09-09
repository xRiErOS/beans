//go:build !windows

package search

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestOpen_ConcurrentReadersShareOnDiskIndex is beans-dfdw's SC-01: it
// starts a real, long-lived child process that opens the persisted index
// for read (OpenRead) and holds that shared lock open -- mirroring
// TestHelperChildIndexUnderLock's real-process pattern, since flock(2) is a
// cross-process primitive a goroutine cannot exercise -- and then, in this
// (second) process, opens the same index for read while the child's shared
// lock is still held. The assertion is on idx.persistent, not merely on
// search results being correct: a correct search result is obtainable from
// either the warm on-disk index or a freshly rebuilt in-memory one, so only
// checking persistent distinguishes "took the warm on-disk path" (AC-03,
// AC-06) from "silently fell back and still happened to work".
func TestOpen_ConcurrentReadersShareOnDiskIndex(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")

	// Phase 0: seed the persisted index uncontended, exactly like
	// TestOpen_LockLoserSkipsSidecarWrite, so there is something on disk
	// for both readers to see.
	seed, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() (seed) error = %v", err)
	}
	if !seed.persistent {
		t.Fatal("Open() on a fresh, uncontended directory should have acquired the persistent index")
	}
	if err := seed.IndexBean(beanWith("seed1", "Seed", "x")); err != nil {
		t.Fatalf("IndexBean() (seed) error = %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("Close() (seed) error = %v", err)
	}

	// Phase 1: start the long-lived shared-lock holder as a real child
	// process and wait for it to prove -- by writing readyPath only after
	// OpenRead() returned a persistent index -- that it is actually
	// holding the shared lock, not merely running.
	signalDir := t.TempDir()
	readyPath := filepath.Join(signalDir, "ready")
	stopPath := filepath.Join(signalDir, "stop")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperChildHoldsSharedLock$", "-test.v", "-test.timeout=25s")
	cmd.Env = append(os.Environ(),
		"BEANS_SEARCH_CHILD_DIR="+dir,
		"BEANS_SEARCH_CHILD_READY="+readyPath,
		"BEANS_SEARCH_CHILD_STOP="+stopPath,
	)
	var childOutput strings.Builder
	cmd.Stdout = &childOutput
	cmd.Stderr = &childOutput
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting child process error = %v", err)
	}

	childDone := make(chan error, 1)
	go func() { childDone <- cmd.Wait() }()

	stopChild := func() {
		_ = os.WriteFile(stopPath, []byte("stop"), 0o644)
		select {
		case <-childDone:
		case <-time.After(10 * time.Second):
			t.Error("child process did not exit after stop signal")
			cancel()
			<-childDone
		}
	}
	defer stopChild()

	waitDeadline := time.Now().Add(15 * time.Second)
	for {
		if _, statErr := os.Stat(readyPath); statErr == nil {
			break
		}
		select {
		case err := <-childDone:
			t.Fatalf("child process exited before signaling ready: %v\n--- child output ---\n%s", err, childOutput.String())
		default:
		}
		if time.Now().After(waitDeadline) {
			t.Fatalf("timed out waiting for child to signal it holds the shared lock\n--- child output ---\n%s", childOutput.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	// childDone is drained only by stopChild's deferred wait below; the
	// child is now demonstrably holding OpenRead's shared lock.

	// Phase 2: THIS process -- a second, independent process from the
	// child's perspective -- opens the same index for read while the
	// child's shared lock is still held. It must take the warm, on-disk
	// path (AC-01, AC-03), not degrade to an in-memory rebuild.
	reader, err := OpenRead(dir)
	if err != nil {
		t.Fatalf("OpenRead() error = %v", err)
	}
	defer reader.Close()

	if !reader.persistent {
		t.Fatal("OpenRead() degraded to an in-memory index while only a concurrent shared-lock reader held the lock; want the warm on-disk path (AC-03)")
	}

	results, err := reader.Search("Seed", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "seed1" {
		t.Fatalf("Search() = %v, want [seed1]", results)
	}
}

// TestHelperChildHoldsSharedLock is not a test in its own right: outside
// the reentrant invocation from TestOpen_ConcurrentReadersShareOnDiskIndex
// (guarded by BEANS_SEARCH_CHILD_DIR) it skips immediately. Under the
// reentrant invocation it opens dir for read, proves it got the persistent
// index, signals readiness by writing readyPath, then holds the shared
// lock open (via idx, deferred to Close) until stopPath appears or a
// generous deadline elapses -- so the parent test's read in Phase 2 is
// measured against a genuinely held shared lock, not a race.
func TestHelperChildHoldsSharedLock(t *testing.T) {
	dir := os.Getenv("BEANS_SEARCH_CHILD_DIR")
	readyPath := os.Getenv("BEANS_SEARCH_CHILD_READY")
	stopPath := os.Getenv("BEANS_SEARCH_CHILD_STOP")
	if dir == "" {
		t.Skip("only runs as a reentrant child process spawned by TestOpen_ConcurrentReadersShareOnDiskIndex")
	}

	idx, err := OpenRead(dir)
	if err != nil {
		t.Fatalf("OpenRead() error = %v", err)
	}
	defer idx.Close()

	if !idx.persistent {
		t.Fatal("OpenRead() did not acquire the persistent index; nothing else holds the lock yet")
	}

	if err := os.WriteFile(readyPath, []byte("ready"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", readyPath, err)
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(stopPath); statErr == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for stop signal from parent")
}
