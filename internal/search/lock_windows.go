//go:build windows

package search

// indexLock is unused on Windows: see tryAcquireIndexLock.
type indexLock struct{}

// tryAcquireIndexLock always reports "not acquired" on Windows. This package
// depends only on the standard library, and there is no portable, dependency-
// free equivalent of flock(2) available there; rather than take a real lock
// with different (and untested) crash-release semantics, every process
// degrades to the in-memory index (AC-07's sanctioned fallback). Persistence
// (AC-01's warm-search speedup) is therefore unavailable on Windows; every
// correctness acceptance criterion (AC-02..AC-07) still holds because the
// in-memory index is rebuilt fresh, from disk, every time.
func tryAcquireIndexLock(dir string) (*indexLock, bool, error) {
	return nil, false, nil
}

// release is a no-op on Windows: tryAcquireIndexLock never hands out a lock.
func (l *indexLock) release() {}
