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
//
// Phase 2's OpenRead call carries its OWN timeout (openWithTimeout), well
// under the child's 20s hold deadline: a prior version of this test relied
// on the test binary's -test.timeout flag instead, which meant "regressed
// to blocking forever" only failed if the *runner* happened to pass a
// timeout shorter than the child's hold -- under a bare `go test` (no
// -timeout override, 10-minute default) the same regression would have
// gone green, because the child's own 20s deadline would fire and release
// the lock before the runner's timeout ever did. See beans-dfdw's
// completion report for the measured before/after.
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
	// process and wait for it to prove it is actually holding the shared
	// lock, not merely running.
	holder := startSharedLockHolder(t, dir)
	defer holder.stop()

	// Phase 2: THIS process -- a second, independent process from the
	// child's perspective -- opens the same index for read while the
	// child's shared lock is still held. It must take the warm, on-disk
	// path (AC-01, AC-03), not degrade to an in-memory rebuild, and it
	// must not block: openWithTimeout fails the test itself if OpenRead
	// has not returned within 5s, rather than trusting the test runner's
	// own -timeout flag to notice a regression to blocking/exclusive
	// behavior.
	reader := openWithTimeout(t, 5*time.Second, OpenRead, dir)
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

// TestOpen_ExclusiveOpenDegradesWhileSharedLockHeld guards the shared half
// of AC-04: "an exclusive lock request that cannot be granted immediately
// -- index held shared OR exclusive by another process -- degrades to
// in-memory rather than blocking or erroring". The exclusive-vs-exclusive
// case was already covered before beans-dfdw by
// TestOpen_LockLoserSkipsSidecarWrite (unmodified, SC-03); exclusive-vs-
// shared was impossible before this leaf (there was no shared mode to
// contend against) and is new. A correct-by-inspection LOCK_EX|LOCK_NB
// against a held LOCK_SH returning EWOULDBLOCK is not the same as an
// executed test: this measures it, and would catch a later regression to
// a blocking flock call (forbidden by beans-dfdw's Non-Goals) that the
// existing exclusive-vs-exclusive test cannot reach.
func TestOpen_ExclusiveOpenDegradesWhileSharedLockHeld(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")

	seed, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() (seed) error = %v", err)
	}
	if !seed.persistent {
		t.Fatal("Open() on a fresh, uncontended directory should have acquired the persistent index")
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("Close() (seed) error = %v", err)
	}

	holder := startSharedLockHolder(t, dir)
	defer holder.stop()

	// A write caller's Open must degrade, not block, while the child's
	// shared lock is held (AC-04). openWithTimeout enforces its own
	// bound so a regression to a blocking flock call fails this test
	// directly instead of hanging until the child's unrelated 20s
	// deadline releases the lock.
	writer := openWithTimeout(t, 5*time.Second, Open, dir)
	defer writer.Close()

	if writer.persistent {
		t.Fatal("Open() (write intent) acquired the persistent index while a shared-lock holder was active; want degrade to in-memory (AC-04)")
	}
}

// openWithTimeout calls open(dir) on a separate goroutine and fails t if it
// has not returned within timeout, so a regression that makes open block
// indefinitely (rather than the documented "never blocks" contract) is
// caught by the test itself, independent of whatever -test.timeout the
// runner happens to pass. The call is intentionally left running if it
// times out (nothing can safely cancel a blocked flock/bleve.Open call);
// that goroutine leaks for the remainder of the test binary's process,
// which is acceptable for the one-shot failure this exists to catch.
func openWithTimeout(t *testing.T, timeout time.Duration, open func(string) (*Index, error), dir string) *Index {
	t.Helper()

	type result struct {
		idx *Index
		err error
	}
	done := make(chan result, 1)
	go func() {
		idx, err := open(dir)
		done <- result{idx, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("open() error = %v", r.err)
		}
		return r.idx
	case <-time.After(timeout):
		t.Fatalf("open() did not return within %s -- regressed to blocking instead of degrading", timeout)
		return nil
	}
}

// sharedLockHolder is a real child process holding OpenRead's shared lock
// on a directory, started by startSharedLockHolder and released by stop.
type sharedLockHolder struct {
	t           *testing.T
	stopPath    string
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	done        chan error
	childOutput *strings.Builder
}

// startSharedLockHolder starts TestHelperChildHoldsSharedLock as a real,
// separate OS process against dir and blocks until it has signaled -- by
// writing its ready file only after OpenRead() returned a persistent index
// -- that it is actually holding the shared lock, not merely running.
func startSharedLockHolder(t *testing.T, dir string) *sharedLockHolder {
	t.Helper()

	signalDir := t.TempDir()
	readyPath := filepath.Join(signalDir, "ready")
	stopPath := filepath.Join(signalDir, "stop")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

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
		cancel()
		t.Fatalf("starting child process error = %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	h := &sharedLockHolder{
		t:           t,
		stopPath:    stopPath,
		cmd:         cmd,
		cancel:      cancel,
		done:        done,
		childOutput: &childOutput,
	}

	waitDeadline := time.Now().Add(15 * time.Second)
	for {
		if _, statErr := os.Stat(readyPath); statErr == nil {
			return h
		}
		select {
		case err := <-done:
			t.Fatalf("child process exited before signaling ready: %v\n--- child output ---\n%s", err, childOutput.String())
		default:
		}
		if time.Now().After(waitDeadline) {
			t.Fatalf("timed out waiting for child to signal it holds the shared lock\n--- child output ---\n%s", childOutput.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// stop signals the child to release the shared lock and exit, and waits
// for it to do so.
func (h *sharedLockHolder) stop() {
	h.t.Helper()
	defer h.cancel()
	_ = os.WriteFile(h.stopPath, []byte("stop"), 0o644)
	select {
	case <-h.done:
	case <-time.After(10 * time.Second):
		h.t.Error("child process did not exit after stop signal")
		h.cancel()
		<-h.done
	}
}

// TestHelperChildHoldsSharedLock is not a test in its own right: outside
// the reentrant invocation from startSharedLockHolder (guarded by
// BEANS_SEARCH_CHILD_DIR) it skips immediately. Under the reentrant
// invocation it opens dir for read, proves it got the persistent index,
// signals readiness by writing readyPath, then holds the shared lock open
// (via idx, deferred to Close) until stopPath appears or a generous
// deadline elapses -- so the parent test's measurement is against a
// genuinely held shared lock, not a race.
func TestHelperChildHoldsSharedLock(t *testing.T) {
	dir := os.Getenv("BEANS_SEARCH_CHILD_DIR")
	readyPath := os.Getenv("BEANS_SEARCH_CHILD_READY")
	stopPath := os.Getenv("BEANS_SEARCH_CHILD_STOP")
	if dir == "" {
		t.Skip("only runs as a reentrant child process spawned by startSharedLockHolder")
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
