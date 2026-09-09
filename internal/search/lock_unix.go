//go:build !windows

package search

import (
	"os"
	"path/filepath"
	"syscall"
)

// indexLock guards write access to a persisted index directory across
// processes. It wraps flock(2), an advisory lock tied to the open file
// descriptor: the kernel releases it automatically when this process
// exits or the descriptor is closed, including on a crash or any exit path
// that never reaches Close() -- unlike a PID file, which needs explicit or
// heuristic cleanup and can be fooled by PID reuse. The same struct backs
// both an exclusive holder (tryAcquireIndexLock) and a shared holder
// (tryAcquireSharedIndexLockUnix, see persist.go's tryAcquireSharedIndexLock
// var): release's LOCK_UN is mode-agnostic.
type indexLock struct {
	f *os.File
}

// tryAcquireIndexLock attempts a non-blocking exclusive lock on
// "<dir>/.lock". It never blocks: if another process already holds the
// lock in any mode, it returns (nil, false, nil) immediately so the caller
// can degrade to an in-memory index (AC-04, beans-6y60 AC-07). Its
// signature is fixed by lock_windows.go's stub of the same name, which
// persist.go calls identically on every platform: do not add parameters
// here without adding the matching parameter there too.
func tryAcquireIndexLock(dir string) (*indexLock, bool, error) {
	return tryAcquireIndexLockMode(dir, syscall.LOCK_EX)
}

// tryAcquireSharedIndexLockUnix attempts a non-blocking shared lock on
// "<dir>/.lock". Multiple processes may hold a shared lock concurrently
// (POSIX flock semantics): a second, third, etc. shared requester is
// granted immediately rather than degrading (AC-03). It is wired into
// persist.go's portable tryAcquireSharedIndexLock var by this file's init,
// so Windows (which never compiles this file) keeps that var's default
// alias to tryAcquireIndexLock -- i.e. always "not acquired" -- unchanged
// (AC-05).
func tryAcquireSharedIndexLockUnix(dir string) (*indexLock, bool, error) {
	return tryAcquireIndexLockMode(dir, syscall.LOCK_SH)
}

func init() {
	tryAcquireSharedIndexLock = tryAcquireSharedIndexLockUnix
}

func tryAcquireIndexLockMode(dir string, flockMode int) (*indexLock, bool, error) {
	path := filepath.Join(dir, ".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), flockMode|syscall.LOCK_NB); err != nil {
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
