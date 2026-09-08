package search

import (
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

// TestSearch_ConcurrentWithIndexBean forces AC-01's concurrency instead of
// hoping the race detector trips over accidental interleaving: it starts
// several goroutines against the SAME *Index, releases them through a
// barrier, and has one goroutine repeatedly call IndexBean while the others
// repeatedly call Search. This is the real production window
// (pkg/beancore/core.go's Core.Search captures the *Index reference under
// c.mu, then calls idx.Search after releasing it, while Create/Update/
// Delete call idx.IndexBean under c.mu on that same *Index -- c.mu is what
// serializes writers there, so this test mirrors it with exactly one writer
// goroutine, not several: idx.etags (index.go:106) is a plain, unsynchronized
// map and concurrent writers to it would crash on "concurrent map writes"
// regardless of Search, which is a different bug than the one this leaf
// measures) reproduced without the Core wrapper, so
// `go test -race ./internal/search/...` has actual concurrent substrate on
// this shared *Index instead of running the existing sequential tests under
// instrumentation for nothing.
//
// Before this test, `grep -nE 'go func|sync.WaitGroup' internal/search/*_test.go`
// found no hits (see beans-rciy's Problem section) -- this is the fix.
func TestSearch_ConcurrentWithIndexBean(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "idx")
	idx := mustOpen(t, dir)

	const searchers = 4
	const iterations = 50

	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 1+searchers)

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		for i := range iterations {
			id := "writer-" + strconv.Itoa(i)
			if err := idx.IndexBean(beanWith(id, "Concurrent Writer", "payload")); err != nil {
				errs <- err
				return
			}
		}
	}()
	for range searchers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for range iterations {
				if _, err := idx.Search("Concurrent", 0); err != nil {
					errs <- err
					return
				}
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent Search/IndexBean on shared *Index error = %v", err)
	}

	// The index must still be fully usable and internally consistent after
	// the concurrent phase: the writer's last bean is findable.
	results, err := idx.Search("Concurrent", 0)
	if err != nil {
		t.Fatalf("Search() after concurrent phase error = %v", err)
	}
	id := "writer-" + strconv.Itoa(iterations-1)
	found := false
	for _, r := range results {
		if r == id {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Search() after concurrent phase = %v, missing %q", results, id)
	}
}
