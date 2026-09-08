//go:build !windows

package search

import (
	"os"
	"path/filepath"
	"syscall"
)

// indexLock guards exclusive write access to a persisted index directory
// across processes. It wraps flock(2), an advisory lock tied to the open
// file descriptor: the kernel releases it automatically when this process
// exits or the descriptor is closed, including on a crash or any exit path
// that never reaches Close() -- unlike a PID file, which needs explicit or
// heuristic cleanup and can be fooled by PID reuse.
type indexLock struct {
	f *os.File
}

// tryAcquireIndexLock attempts a non-blocking lock on "<dir>/.lock". It never
// blocks: if another process already holds the lock, it returns (nil, false,
// nil) immediately so the caller can degrade to an in-memory index (AC-07).
func tryAcquireIndexLock(dir string) (*indexLock, bool, error) {
	path := filepath.Join(dir, ".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &indexLock{f: f}, true, nil
}

// release unlocks and closes the lock file. Safe to call on a nil lock.
func (l *indexLock) release() {
	if l == nil || l.f == nil {
		return
	}
	syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	l.f.Close()
}
