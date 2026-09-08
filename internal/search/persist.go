package search

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/blevesearch/bleve/v2"
)

// etagsFileName is the sidecar file, next to the Bleve index directory's own
// files, that records the ETag this index last observed for each bean ID.
// It is the staleness signal Sync diffs against (AC-03): never mtime.
const etagsFileName = "beans-etags.json"

// Open returns a search index backed by the persisted Bleve index at dir
// when this process can claim exclusive access to it, or an in-memory index
// otherwise. It never returns an error for ordinary contention or a missing/
// corrupt on-disk index -- both degrade to a working index (AC-05, AC-07);
// Open only errors when even the in-memory fallback cannot be built.
func Open(dir string) (*Index, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return NewIndex()
	}

	lock, acquired, err := tryAcquireIndexLock(dir)
	if err != nil || !acquired {
		// Contended, or the lock itself couldn't be established (e.g. a
		// read-only filesystem): fall back to a private in-memory index
		// rather than fail the caller's search (AC-07).
		return NewIndex()
	}

	idx, etags, err := openOrCreatePersistent(dir)
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

// openOrCreatePersistent opens the Bleve index at dir, creating it (and a
// fresh, empty ETag sidecar) if absent. A corrupt on-disk index is treated as
// absent: it is discarded and rebuilt, so a damaged persisted index heals
// itself on the next open (AC-05) instead of wedging every future search.
func openOrCreatePersistent(dir string) (bleve.Index, map[string]string, error) {
	idx, err := bleve.Open(dir)
	switch {
	case err == nil:
		etags, loadErr := loadETags(dir)
		if loadErr != nil {
			// Sidecar missing or unreadable but the Bleve index itself
			// opened fine: treat every bean as unseen rather than fail --
			// Sync will simply reindex everything once, same cost as a
			// fresh build (AC-05).
			etags = map[string]string{}
		}
		return idx, etags, nil
	case errors.Is(err, bleve.ErrorIndexPathDoesNotExist):
		idx, createErr := bleve.New(dir, buildIndexMapping())
		if createErr != nil {
			return nil, nil, createErr
		}
		return idx, map[string]string{}, nil
	default:
		// Corrupt or otherwise unopenable: wipe and rebuild fresh.
		if rmErr := removeContents(dir); rmErr != nil {
			return nil, nil, fmt.Errorf("clearing corrupt index at %s: %w", dir, rmErr)
		}
		idx, createErr := bleve.New(dir, buildIndexMapping())
		if createErr != nil {
			return nil, nil, createErr
		}
		return idx, map[string]string{}, nil
	}
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
