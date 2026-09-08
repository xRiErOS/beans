package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xRiErOS/beans/pkg/bean"
)

func mustOpen(t *testing.T, dir string) *Index {
	t.Helper()
	idx, err := Open(dir)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", dir, err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

func beanWith(id, title, body string) *bean.Bean {
	return &bean.Bean{ID: id, Slug: id, Title: title, Body: body}
}

// TestOpen_PersistsAcrossReopen proves the index is actually written to disk
// (AC-01's precondition): data indexed by one Index survives Close and a
// fresh Open of the same directory, without resupplying the beans.
func TestOpen_PersistsAcrossReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")

	idx1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := idx1.IndexBean(beanWith("aaa1", "Persisted Title", "content")); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	idx2 := mustOpen(t, dir)
	results, err := idx2.Search("Persisted", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "aaa1" {
		t.Fatalf("Search() after reopen = %v, want [aaa1]", results)
	}
}

// TestOpen_AbsentIndexRebuildsTransparently proves AC-05/SC-03: deleting the
// persisted index directory and reopening produces a working, empty index
// rather than an error.
func TestOpen_AbsentIndexRebuildsTransparently(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := idx1.IndexBean(beanWith("aaa1", "Gone Soon", "x")); err != nil {
		t.Fatalf("IndexBean() error = %v", err)
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	idx2 := mustOpen(t, dir)
	if !idx2.persistent {
		t.Fatal("Open() after deletion fell back to in-memory instead of rebuilding the persisted index")
	}
	results, err := idx2.Search("Gone", 0)
	if err != nil {
		t.Fatalf("Search() after deletion error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search() after deletion = %v, want none (index was rebuilt empty)", results)
	}
	// And the rebuilt index is fully functional going forward.
	if err := idx2.IndexBean(beanWith("bbb2", "Rebuilt", "x")); err != nil {
		t.Fatalf("IndexBean() on rebuilt index error = %v", err)
	}
	results, err = idx2.Search("Rebuilt", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "bbb2" {
		t.Fatalf("Search() = %v, want [bbb2]", results)
	}
}

// TestOpen_CorruptIndexRebuildsTransparently proves the corrupt-on-disk half
// of AC-05: a directory whose contents are not a valid Bleve index (e.g. a
// stray file from a prior crash) heals on the next Open instead of making
// every future search fail.
func TestOpen_CorruptIndexRebuildsTransparently(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index_meta.json"), []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	idx := mustOpen(t, dir)
	if !idx.persistent {
		t.Fatal("Open() on a corrupt directory fell back to in-memory instead of rebuilding the persisted index")
	}
	if err := idx.IndexBean(beanWith("aaa1", "Healed", "x")); err != nil {
		t.Fatalf("IndexBean() on healed index error = %v", err)
	}
	results, err := idx.Search("Healed", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "aaa1" {
		t.Fatalf("Search() = %v, want [aaa1]", results)
	}
}

// TestSync_SkipsBeansWithUnchangedETag proves the core of AC-01/AC-02: a
// bean whose ETag matches the sidecar is left untouched by Sync. A doc is
// hand-planted directly through the underlying Bleve index (bypassing
// Sync/IndexBean) with content that no bean in the Sync call would ever
// produce; if Sync reindexed it anyway despite an unchanged ETag, the
// hand-planted content would be overwritten.
func TestSync_SkipsBeansWithUnchangedETag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx := mustOpen(t, dir)

	b := beanWith("aaa1", "Stable Title", "stable body")
	if err := idx.Sync([]*bean.Bean{b}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}

	// Plant a sentinel directly, bypassing Sync/IndexBean.
	if err := idx.index.Index("aaa1", beanDocument{ID: "aaa1", Title: "SENTINEL-UNTOUCHED"}); err != nil {
		t.Fatalf("planting sentinel error = %v", err)
	}

	// Same bean, same ETag: Sync must not touch "aaa1".
	if err := idx.Sync([]*bean.Bean{b}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx.Search("SENTINEL-UNTOUCHED", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "aaa1" {
		t.Fatalf("Sync() reindexed a bean whose ETag had not changed; sentinel lost, got %v", results)
	}
}

// TestSync_ReindexesChangedETag is the mirror of the above: when the ETag
// differs (content actually changed), Sync must reindex it.
func TestSync_ReindexesChangedETag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx := mustOpen(t, dir)

	b := beanWith("aaa1", "Original Title", "original body")
	if err := idx.Sync([]*bean.Bean{b}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}

	changed := beanWith("aaa1", "Changed Title", "changed body")
	if err := idx.Sync([]*bean.Bean{changed}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx.Search("Changed", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "aaa1" {
		t.Fatalf("Sync() did not reindex a bean whose ETag changed, got %v", results)
	}
}

// TestSync_RemovesDeletedBeans proves Sync retracts beans no longer present.
func TestSync_RemovesDeletedBeans(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx := mustOpen(t, dir)

	a := beanWith("aaa1", "Keep Me", "x")
	b := beanWith("bbb2", "Delete Me", "x")
	if err := idx.Sync([]*bean.Bean{a, b}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}

	if err := idx.Sync([]*bean.Bean{a}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx.Search("Delete", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search() = %v, want none: deleted bean was not retracted", results)
	}
	if _, stillPresent := idx.etags["bbb2"]; stillPresent {
		t.Fatal("etags sidecar still tracks a bean Sync removed")
	}
}

// TestSync_UsesETagNotUpdatedAt proves AC-03: two bean values with the same
// ETag (identical persisted content) but different UpdatedAt timestamps --
// as a metadata-only touch that doesn't change file bytes would produce --
// must be treated as unchanged. A production regression that keyed
// staleness off UpdatedAt instead of ETag would reindex the second one,
// overwriting the sentinel this test plants. It uses SetContentETag, the
// exact mechanism Bean.loadBean uses when parsing a file from disk (bean.go
// ETag() returns b.contentETag whenever it is set, never re-rendering from
// the in-memory struct) -- so two Bean values built from identical file
// bytes carry an identical ETag even when other fields, like UpdatedAt as a
// metadata-only touch would leave it, differ.
func TestSync_UsesETagNotUpdatedAt(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx := mustOpen(t, dir)

	const fileBytes = "identical file bytes as loaded from disk"
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	b1 := beanWith("aaa1", "Same Content", "same body")
	b1.CreatedAt, b1.UpdatedAt = &t1, &t1
	b1.SetContentETag([]byte(fileBytes))

	if err := idx.Sync([]*bean.Bean{b1}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}
	if err := idx.index.Index("aaa1", beanDocument{ID: "aaa1", Title: "SENTINEL-UNTOUCHED"}); err != nil {
		t.Fatalf("planting sentinel error = %v", err)
	}

	// Same on-disk bytes (so the same ETag), but a later UpdatedAt -- the
	// shape of a metadata-only touch (e.g. `touch`, or a reload after a
	// filesystem mtime bump with no byte change), not a content edit.
	t2 := t1.Add(time.Hour)
	b2 := beanWith("aaa1", "Same Content", "same body")
	b2.CreatedAt, b2.UpdatedAt = &t1, &t2
	b2.SetContentETag([]byte(fileBytes))
	if b1.ETag() != b2.ETag() {
		t.Fatalf("test precondition failed: expected identical ETag for identical file bytes, got %q vs %q", b1.ETag(), b2.ETag())
	}

	if err := idx.Sync([]*bean.Bean{b2}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx.Search("SENTINEL-UNTOUCHED", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Sync() reindexed on an UpdatedAt-only change despite an unchanged ETag, sentinel lost, got %v", results)
	}
}

// TestSync_InMemoryIndexRebuildsColdFromScratch documents the fallback
// index's cold-start behavior: a fresh in-memory index has an empty
// idx.etags (see NewIndex), so its first Sync in a new process indexes
// everything, same as IndexBeans.
func TestSync_InMemoryIndexRebuildsColdFromScratch(t *testing.T) {
	idx := setupTestIndex(t)
	b := beanWith("aaa1", "Title", "body")
	if err := idx.Sync([]*bean.Bean{b}); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	results, err := idx.Search("Title", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search() = %v, want [aaa1]", results)
	}
}

// TestSync_InMemoryIndexRetractsDeletedBeans closes the fallback-path gap a
// review found: the in-memory index used under AC-07's contention fallback
// must retract a bean no longer present on a later Sync (a reload while
// contended is a reachable sequence), exactly like the persistent path.
func TestSync_InMemoryIndexRetractsDeletedBeans(t *testing.T) {
	idx := setupTestIndex(t)
	a := beanWith("aaa1", "Keep Me", "x")
	b := beanWith("bbb2", "Delete Me", "x")
	if err := idx.Sync([]*bean.Bean{a, b}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}

	if err := idx.Sync([]*bean.Bean{a}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx.Search("Delete", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search() = %v, want none: the in-memory fallback did not retract a deleted bean", results)
	}
}

// TestSync_SidecarPersistsAcrossReopen proves the sidecar write at the end
// of Sync actually lands on disk and is read back: without it, a second
// process (a fresh Index over the same directory) would start with an empty
// idx.etags and reindex everything it sees, silently losing AC-01's warm
// path. A hand-planted sentinel on the reopened index would then be
// overwritten by that spurious reindex; this test asserts it survives.
func TestSync_SidecarPersistsAcrossReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")

	b := beanWith("aaa1", "Persisted Sync", "content")
	idx1, err := Open(dir)
	if err != nil {
		t.Fatalf("Open() #1 error = %v", err)
	}
	if err := idx1.Sync([]*bean.Bean{b}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}
	if err := idx1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	idx2 := mustOpen(t, dir)
	if err := idx2.index.Index("aaa1", beanDocument{ID: "aaa1", Title: "SENTINEL-UNTOUCHED"}); err != nil {
		t.Fatalf("planting sentinel error = %v", err)
	}

	// Same bean, unchanged since before Close: a correctly loaded sidecar
	// means Sync recognizes it as current and leaves the sentinel alone.
	if err := idx2.Sync([]*bean.Bean{b}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx2.Search("SENTINEL-UNTOUCHED", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Sync() on reopen reindexed despite an unchanged bean; the ETag sidecar was not persisted/loaded, sentinel lost, got %v", results)
	}
}

// TestSync_ReindexesOnPathChangeEvenWithUnchangedETag proves the fix for the
// rename hole in AC-02: Slug/Path are excluded from the rendered front
// matter (`yaml:"-"` in bean.Bean), so a rename alone -- same bytes, new
// filename, as `beans rename` or a `git checkout` that only renames a file
// produces -- never changes ETag. staleKey folds Path in specifically so
// this case still reindexes.
func TestSync_ReindexesOnPathChangeEvenWithUnchangedETag(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx := mustOpen(t, dir)

	const fileBytes = "identical file bytes, only the filename changes"
	before := beanWith("aaa1", "Same Title", "same body")
	before.Path = "old-slug--aaa1.md"
	before.Slug = "old-slug"
	before.SetContentETag([]byte(fileBytes))

	if err := idx.Sync([]*bean.Bean{before}); err != nil {
		t.Fatalf("Sync() #1 error = %v", err)
	}

	after := beanWith("aaa1", "Same Title", "same body")
	after.Path = "new-slug--aaa1.md"
	after.Slug = "new-slug"
	after.SetContentETag([]byte(fileBytes))
	if before.ETag() != after.ETag() {
		t.Fatalf("test precondition failed: rename should not change ETag, got %q vs %q", before.ETag(), after.ETag())
	}

	if err := idx.Sync([]*bean.Bean{after}); err != nil {
		t.Fatalf("Sync() #2 error = %v", err)
	}

	results, err := idx.Search("new-slug", 0)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0] != "aaa1" {
		t.Fatalf("Search(new-slug) = %v, want [aaa1]: rename was not reflected despite an unchanged ETag", results)
	}
}
