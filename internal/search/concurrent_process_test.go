//go:build !windows

package search

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestOpen_LockLoserSkipsSidecarWrite forces AC-02's cross-process case
// instead of hoping two independently started processes overlap (see
// beans-rciy's Finding F-19: eight rounds of "start both together" measured
// zero observable overlap). It holds the persisted index's flock itself --
// exactly as a second beans process holding it would, and exactly as
// TestOpen_FallsBackWhenLockContended already does for the single-process
// case -- and observes that held state BEFORE running the measurement: a
// real child PROCESS (not goroutine; flock is a cross-process primitive
// in-process contention cannot exercise) that performs an actual indexing
// operation (IndexBean) while the lock is held. The sidecar file's mtime and
// size, read before and after the child runs, are the objective measurement
// that the lock loser wrote nothing to the shared persistent state -- not a
// property the child could truthfully self-report, since a corrupted
// process could still claim success. A second phase then releases the lock
// and reopens persistently to prove the persistent path resumes writing
// afterward -- the flip side of AC-02, not assumed from the first phase.
func TestOpen_LockLoserSkipsSidecarWrite(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	sidecarPath := filepath.Join(dir, etagsFileName)

	// Phase 0: establish the persisted index and its sidecar uncontended, so
	// there is a baseline mtime/size to compare against. Must Close before
	// taking the external flock below, or that flock would contend against
	// ourselves.
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

	before, err := os.Stat(sidecarPath)
	if err != nil {
		t.Fatalf("Stat(%q) before contention error = %v", sidecarPath, err)
	}

	// Phase 1: hold the lock externally, exactly like
	// TestOpen_FallsBackWhenLockContended, and observe the held state before
	// running the child -- the contention is forced, not hoped for.
	lockPath := filepath.Join(dir, ".lock")
	holder, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile(%q) error = %v", lockPath, err)
	}
	defer holder.Close()
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("Flock() error = %v", err)
	}
	held := true
	release := func() {
		if !held {
			return
		}
		syscall.Flock(int(holder.Fd()), syscall.LOCK_UN)
		held = false
	}
	defer release()

	// The lock is now demonstrably held by this process (the flock call
	// above returned nil, i.e. succeeded, before the child is ever started)
	// -- this is the "observed before the measurement call" requirement.

	runChild(t, dir)

	// Phase 2: the lock loser must not have touched the shared sidecar.
	duringLock, err := os.Stat(sidecarPath)
	if err != nil {
		t.Fatalf("Stat(%q) after contended child error = %v", sidecarPath, err)
	}
	if duringLock.ModTime() != before.ModTime() || duringLock.Size() != before.Size() {
		t.Fatalf("sidecar changed while the lock was held: before mtime=%v size=%d, after mtime=%v size=%d",
			before.ModTime(), before.Size(), duringLock.ModTime(), duringLock.Size())
	}

	// Phase 3: release, and prove the persistent path writes again.
	release()

	resumed, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() (resumed) error = %v", err)
	}
	if !resumed.persistent {
		t.Fatal("Open() after the external lock was released should have acquired the persistent index")
	}
	if err := resumed.IndexBean(beanWith("resumed1", "Resumed", "x")); err != nil {
		t.Fatalf("IndexBean() (resumed) error = %v", err)
	}
	if err := resumed.Close(); err != nil {
		t.Fatalf("Close() (resumed) error = %v", err)
	}

	after, err := os.Stat(sidecarPath)
	if err != nil {
		t.Fatalf("Stat(%q) after release error = %v", sidecarPath, err)
	}
	if after.ModTime() == before.ModTime() && after.Size() == before.Size() {
		t.Fatalf("sidecar unchanged after the lock was released and a new bean indexed: mtime=%v size=%d",
			after.ModTime(), after.Size())
	}
}

// runChild launches a second, real OS process (re-entering this same test
// binary) that opens dir -- which the caller must already hold the flock
// on -- and performs an indexing operation on whatever Open returns. Its
// own assertions (via TestHelperChildIndexUnderLock) fail the child process
// if it is ever handed a persistent index while the lock is held; this
// function additionally fails the parent test if the child process itself
// fails or hangs.
func runChild(t *testing.T, dir string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperChildIndexUnderLock$", "-test.v", "-test.timeout=25s")
	cmd.Env = append(os.Environ(),
		"BEANS_SEARCH_CHILD_DIR="+dir,
	)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("child process timed out\n--- child output ---\n%s", output)
	}
	if err != nil {
		t.Fatalf("child process failed: %v\n--- child output ---\n%s", err, output)
	}

	// A zero exit status alone is vacuous: a child that never found its
	// reentry env var (SKIP) or whose -test.run matched nothing ("no tests
	// to run") also exits 0 and would make this a B54-class silent pass.
	// Demand the one line that proves the reentrant indexing assertions in
	// TestHelperChildIndexUnderLock actually ran and passed.
	if !strings.Contains(string(output), "--- PASS: TestHelperChildIndexUnderLock") {
		t.Fatalf("child process exited 0 without proving TestHelperChildIndexUnderLock ran (SKIP or no-match would also exit 0)\n--- child output ---\n%s", output)
	}
	if strings.Contains(string(output), "--- SKIP: TestHelperChildIndexUnderLock") || strings.Contains(string(output), "no tests to run") {
		t.Fatalf("child process skipped instead of running the reentrant assertions\n--- child output ---\n%s", output)
	}
}

// TestHelperChildIndexUnderLock is not a test in its own right: outside the
// reentrant invocation from runChild (guarded by BEANS_SEARCH_CHILD_DIR) it
// skips immediately, so a normal `go test ./internal/search/...` run shows
// it as SKIP. Under the reentrant invocation it opens dir -- the directory
// the parent holds the persisted index's flock on -- and performs a real
// indexing operation on the result. It must observe an in-memory fallback
// index (AC-02's premise; a persistent index here would mean the flock
// check in Open did not do its job) and must be able to search what it just
// indexed on that fallback index.
func TestHelperChildIndexUnderLock(t *testing.T) {
	dir := os.Getenv("BEANS_SEARCH_CHILD_DIR")
	if dir == "" {
		t.Skip("only runs as a reentrant child process spawned by TestOpen_LockLoserSkipsSidecarWrite")
	}

	idx, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer idx.Close()

	if idx.persistent {
		t.Fatal("Open() returned a persistent index while the parent held the lock")
	}

	if err := idx.IndexBean(beanWith("child1", "Child Indexed While Contended", "x")); err != nil {
		t.Fatalf("IndexBean() on the fallback index error = %v", err)
	}
	results, err := idx.Search("Child", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "child1" {
		t.Fatalf("Search() = %v, want [child1]", results)
	}
}
