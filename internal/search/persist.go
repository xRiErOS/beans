package search

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve/v2"
)

// etagsFileName is the sidecar file, next to the Bleve index directory's own
// files, that records the ETag this index last observed for each bean ID.
// It is the staleness signal Sync diffs against (AC-03): never mtime.
const etagsFileName = "beans-etags.json"

// lockMode selects which of tryAcquireIndexLock (exclusive) or
// tryAcquireSharedIndexLock (shared) open uses to acquire dir's on-disk
// lock. It is a persist.go-level concept, not part of tryAcquireIndexLock's
// signature: that signature is fixed by lock_windows.go's stub of the same
// name and must stay identical -- dir string in, (*indexLock, bool, error)
// out -- on every platform, so the two lock modes are two distinct
// functions instead of one function taking a mode parameter.
type lockMode int

const (
	lockModeExclusive lockMode = iota
	lockModeShared
)

// tryAcquireSharedIndexLock defaults to tryAcquireIndexLock's exclusive
// behavior -- i.e. on Windows, where lock_unix.go never compiles, a shared
// request degrades exactly like an exclusive one always has (AC-05: every
// lock acquisition continues to report not-acquired there, unchanged).
// lock_unix.go's init overrides this with a true shared-mode flock on
// platforms that support it.
var tryAcquireSharedIndexLock = tryAcquireIndexLock

// Open returns a search index backed by the persisted Bleve index at dir
// when this process can claim exclusive access to it, or an in-memory index
// otherwise. It never returns an error for ordinary contention or a missing/
// corrupt on-disk index -- both degrade to a working index (AC-05, AC-07);
// Open only errors when even the in-memory fallback cannot be built.
//
// Open is the write-caller entry point (sync/index-update): it acquires the
// on-disk lock exclusively and, unchanged from before, never blocks -- an
// exclusive request that cannot be granted immediately degrades to
// in-memory rather than blocking or erroring (beans-6y60 AC-07).
//
// Use OpenRead for a read caller (search/list queries): it acquires the
// lock in shared mode instead, so concurrent readers -- including two
// long-lived processes such as beans serve and beans-tui -- can each get
// the warm, persisted index instead of forcing every reader but the first
// into the cold in-memory rebuild (AC-03). OpenRead's persisted index, when
// obtained, is backed by Bleve's read-only mode (see openOrCreatePersistent):
// writing to it -- IndexBean, DeleteBean, or Sync -- blocks forever rather
// than erroring, because Bleve's background persister never starts for a
// read-only-opened index. A caller MUST NOT call OpenRead if it will reuse
// the returned *Index for writes later in its lifetime (e.g. a long-lived
// process that also handles bean Create/Update/Delete through the same
// cached index): use Open for that caller instead, even though it is
// conceptually "mostly reading" -- see beans-dfdw's completion report for
// why pkg/beancore's only current call site keeps write intent for exactly
// this reason.
func Open(dir string) (*Index, error) {
	return open(dir, lockModeExclusive)
}

// OpenRead is Open's read-caller counterpart: see Open's doc comment,
// especially the warning about never reusing the result for writes.
func OpenRead(dir string) (*Index, error) {
	return open(dir, lockModeShared)
}

func open(dir string, mode lockMode) (*Index, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return NewIndex()
	}

	acquire := tryAcquireIndexLock
	if mode == lockModeShared {
		acquire = tryAcquireSharedIndexLock
	}
	lock, acquired, err := acquire(dir)
	if err != nil || !acquired {
		// Contended, or the lock itself couldn't be established (e.g. a
		// read-only filesystem): fall back to a private in-memory index
		// rather than fail the caller's search (AC-07).
		return NewIndex()
	}

	idx, etags, err := openOrCreatePersistent(dir, mode)
	if err != nil {
		lock.release()
		return NewIndex()
	}

	return &Index{
		index:      idx,
		persistent: true,
		dir:        dir,
		etags:      etags,
		lock:       lock,
	}, nil
}

// openOrCreatePersistent opens dir's Bleve index according to mode.
//
// For lockModeExclusive (the write caller): unchanged from before this
// mode split. Note there is no separate "absent" case: Open always
// MkdirAll's dir first, so bleve.Open on a first use sees an *existing,
// empty* directory and reports ErrorIndexMetaMissing, not
// ErrorIndexPathDoesNotExist -- verified directly against this bleve
// version (v2.5.6) rather than assumed. A first-ever open and a corrupt
// leftover therefore take the same path below: any non-nil error wipes the
// directory and creates fresh.
//
// For lockModeShared (the read caller): opens with Bleve's own "read_only"
// runtime config. This is not about our own on-disk lock (tryAcquireIndexLock
// already granted that in shared mode) -- it is required because Bleve's
// default (scorch) storage backend opens its own internal bolt file with an
// OS-level *exclusive* flock regardless of whether the caller ever writes,
// so two ordinary bleve.Open calls against the same directory from
// different processes still serialize against each other even when both
// only ever read (measured directly: the second call blocks indefinitely,
// not just until a timeout). Bolt's read_only mode requests a *shared*
// flock instead, so concurrent readers no longer block each other or
// degrade to in-memory (AC-03). The cost: a Bleve index opened this way
// rejects further writes by hanging on the first IndexBean/DeleteBean/Sync
// call rather than erroring (also measured directly), because its
// background persister/merger goroutines never start in read-only mode.
// openOrCreatePersistent therefore never creates or repairs the index in
// this mode -- that would itself be a write -- and any open error (absent,
// corrupt, or otherwise unreadable) is left for the caller to interpret as
// "degrade to in-memory" exactly like the exclusive path (AC-05, AC-07).
//
// A caller that will reuse the returned *Index for writes after this call
// returns (as pkg/beancore's ensureSearchIndexLocked does when it caches
// Core.searchIndex for a long-lived process's later Create/Update/Delete)
// MUST NOT pass lockModeShared: see Open and OpenRead's doc comments.
func openOrCreatePersistent(dir string, mode lockMode) (bleve.Index, map[string]string, error) {
	if mode == lockModeShared {
		idx, err := bleve.OpenUsing(dir, map[string]interface{}{"read_only": true})
		if err != nil {
			return nil, nil, err
		}
		etags, loadErr := loadETags(dir)
		if loadErr != nil {
			etags = map[string]string{}
		}
		return idx, etags, nil
	}

	idx, err := bleve.Open(dir)
	if err == nil {
		etags, loadErr := loadETags(dir)
		if loadErr != nil {
			// Sidecar missing or unreadable but the Bleve index itself
			// opened fine: treat every bean as unseen rather than fail --
			// Sync will simply reindex everything once, same cost as a
			// fresh build (AC-05).
			etags = map[string]string{}
		}
		return idx, etags, nil
	}

	// Absent (never created) or corrupt (leftover from a crash): wipe and
	// (re)build fresh either way (AC-05).
	if rmErr := removeContents(dir); rmErr != nil {
		return nil, nil, fmt.Errorf("clearing index at %s: %w", dir, rmErr)
	}
	fresh, createErr := bleve.New(dir, buildIndexMapping())
	if createErr != nil {
		return nil, nil, createErr
	}
	return fresh, map[string]string{}, nil
}

// removeContents deletes everything inside dir (but not dir itself, which
// still holds the just-released lock file's directory entry).
func removeContents(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == ".lock" {
			continue
		}
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func loadETags(dir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(dir, etagsFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	etags := map[string]string{}
	if err := json.Unmarshal(data, &etags); err != nil {
		return nil, err
	}
	return etags, nil
}

// saveETags writes the sidecar atomically (write-temp then rename) so a
// crash mid-write never leaves a partially-written, unparseable sidecar.
func saveETags(dir string, etags map[string]string) error {
	if dir == "" {
		// An empty dir means an in-memory index (see NewIndex): dir joined
		// with etagsFileName would resolve relative to the caller's cwd
		// instead of failing, silently polluting whatever directory the
		// process happens to run in. Reject it here so the promise is a
		// return value Sync's caller (and this guard's own test) can
		// assert on, not a filesystem location a test has to scan.
		return fmt.Errorf("saveETags: empty dir (in-memory index must not persist a sidecar)")
	}
	data, err := json.Marshal(etags)
	if err != nil {
		return err
	}
	final := filepath.Join(dir, etagsFileName)
	tmp := final + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, final)
}
