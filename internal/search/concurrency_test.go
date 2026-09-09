//go:build !windows

package search

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestOpen_FallsBackWhenLockContended forces the AC-07 contention case
// instead of hoping a goroutine race reproduces it: the lock file is
// acquired directly, exactly as a second beans process holding it would,
// before Open is ever called. Open must not block, error, or hang -- it
// must return a working, private in-memory index for the caller that
// couldn't acquire the persisted one.
func TestOpen_FallsBackWhenLockContended(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lockPath := filepath.Join(dir, ".lock")
	holder, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer holder.Close()
	if err := syscall.Flock(int(holder.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("Flock() error = %v", err)
	}
	defer syscall.Flock(int(holder.Fd()), syscall.LOCK_UN)

	// Open must not block while the lock is contended (AC-07):
	// openWithTimeout fails this test itself, by name, if it has not
	// returned within 5s, instead of trusting the package's own
	// -test.timeout to notice a regression to a blocking flock call.
	idx := openWithTimeout(t, 5*time.Second, Open, dir)
	defer idx.Close()

	if idx.persistent {
		t.Fatal("Open() reported a persistent index despite the lock being held by another holder")
	}

	// The fallback index must still work end to end.
	if err := idx.IndexBean(beanWith("aaa1", "Fallback Works", "x")); err != nil {
		t.Fatalf("IndexBean() on fallback index error = %v", err)
	}
	results, err := idx.Search("Fallback", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "aaa1" {
		t.Fatalf("Search() on fallback index = %v, want [aaa1]", results)
	}

	// The contended directory must not have been corrupted or written to by
	// the process that couldn't acquire the lock.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != ".lock" {
		t.Fatalf("contended index directory contents = %v, want only the lock file untouched", entries)
	}
}

// TestOpen_SecondCallerAcquiresAfterFirstReleases proves the lock is really
// released (not merely never taken): once the first holder releases, a
// second Open on the same directory gets the real persistent index.
func TestOpen_SecondCallerAcquiresAfterFirstReleases(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")

	// openWithTimeout guards this Open() the same way: on an uncontended
	// directory it should return almost instantly, so a regression to a
	// blocking flock call still fails this test by name rather than
	// hanging until the package's own -test.timeout.
	idx1 := openWithTimeout(t, 5*time.Second, Open, dir)
	if !idx1.persistent {
		t.Fatal("Open() #1 on an uncontended directory should have acquired the persistent index")
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	idx2 := openWithTimeout(t, 5*time.Second, Open, dir)
	defer idx2.Close()
	if !idx2.persistent {
		t.Fatal("Open() #2 after the first holder released should have acquired the persistent index")
	}
}
