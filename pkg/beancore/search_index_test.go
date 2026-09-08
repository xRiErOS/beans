package beancore

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/xRiErOS/beans/pkg/bean"
)

// TestIndexDir_OutsideStoreRoot proves AC-04: the persisted index location is
// never inside the store root, so writeGitignore's ".conversations/"-only
// exclusion can never leave it git-tracked.
func TestIndexDir_OutsideStoreRoot(t *testing.T) {
	core, beansDir := setupTestCore(t)
	defer core.Close()

	dir, err := core.indexDir()
	if err != nil {
		t.Fatalf("indexDir() error = %v", err)
	}

	absRoot, err := filepath.Abs(beansDir)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	rel, err := filepath.Rel(absRoot, dir)
	if err != nil {
		t.Fatalf("Rel() error = %v", err)
	}
	if rel == "." || !strings.HasPrefix(rel, "..") {
		t.Fatalf("indexDir() = %q is inside the store root %q (rel = %q), want outside", dir, absRoot, rel)
	}
}

// TestIndexDir_DistinctPerStore proves two different stores never collide on
// the same persisted index directory (which would let one project's search
// results leak into another's).
func TestIndexDir_DistinctPerStore(t *testing.T) {
	core1, _ := setupTestCore(t)
	defer core1.Close()
	core2, _ := setupTestCore(t)
	defer core2.Close()

	dir1, err := core1.indexDir()
	if err != nil {
		t.Fatalf("indexDir() #1 error = %v", err)
	}
	dir2, err := core2.indexDir()
	if err != nil {
		t.Fatalf("indexDir() #2 error = %v", err)
	}
	if dir1 == dir2 {
		t.Fatalf("two distinct stores resolved to the same index directory %q", dir1)
	}
}

// TestSearchIndexField_OnlyTouchedByKnownSeams is the structural half of
// AC-06 ("no command other than search depends on the index"): it parses
// every non-test source file in this package -- not just core.go, so a
// dependency introduced from another file or a free function in the
// package would still be caught -- and asserts the Core.searchIndex field
// is referenced only inside the known, already-audited seams: the lazy
// initializer, Search itself, the incremental maintenance hooks on
// Create/Update/Delete, the resync on reload, and Close. Any other
// function touching it (in particular anything backing
// list/roadmap/milestones/progress/next/show/graph) fails this test before
// it ever reaches a behavioral difference.
func TestSearchIndexField_OnlyTouchedByKnownSeams(t *testing.T) {
	allowed := map[string]bool{
		"ensureSearchIndexLocked": true,
		"Search":                  true,
		"loadFromDisk":            true,
		"Create":                  true,
		"Update":                  true,
		"Delete":                  true,
		"Close":                   true,
		// The widened, whole-package scan also caught fsnotify-driven
		// incremental maintenance in watcher.go/worktree_watcher.go for
		// long-running processes (beans-serve, the TUI): the same
		// best-effort, nil-gated IndexBean/DeleteBean pattern as
		// Create/Update/Delete, just triggered by a filesystem event
		// instead of a direct API call. Not a correctness dependency for
		// any non-search command: errors are logged, never surfaced.
		"handleChanges":            true,
		"loadWorktreeBeansInitial": true,
		"handleWorktreeChanges":    true,
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	fset := token.NewFileSet()
	var offenders []string
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%q) error = %v", name, err)
		}
		scanned++

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			fnName := fn.Name.Name
			touches := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if ok && sel.Sel.Name == "searchIndex" {
					touches = true
				}
				return true
			})
			if touches && !allowed[fnName] {
				offenders = append(offenders, name+":"+fnName)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("scanned zero source files -- test is not exercising the package")
	}

	if len(offenders) > 0 {
		t.Fatalf("functions outside the audited search-index seams reference c.searchIndex: %v", offenders)
	}
}

// TestSearch_CrossProcessFreshness proves AC-02/SC-02: a bean file written
// directly to disk -- bypassing the Core instance doing the searching, as a
// second beans process or an editor would -- is found by the next Search
// once that instance reloads. The two Core instances share the same
// persisted index directory (derived purely from the store root), and the
// first releases its lock via Close() before the second opens it, so the
// second genuinely opens the persisted index the first one wrote.
func TestSearch_CrossProcessFreshness(t *testing.T) {
	coreA, beansDir := setupTestCore(t)
	if err := coreA.Create(&bean.Bean{ID: "aaa1", Slug: "a", Title: "From Process A", Body: "x"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := coreA.Search("Process"); err != nil {
		t.Fatalf("Search() (process A) error = %v", err)
	}
	if err := coreA.Close(); err != nil {
		t.Fatalf("Close() (process A) error = %v", err)
	}

	// A second process/editor writes a bean file directly -- coreA (and any
	// index it built) never sees this write.
	content := "---\ntitle: From External Editor\nstatus: todo\ntype: task\n---\n\nexternal body\n"
	if err := os.WriteFile(filepath.Join(beansDir, "ext1--external.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	coreB := New(beansDir, coreA.Config())
	coreB.SetWarnWriter(nil)
	if err := coreB.Load(); err != nil {
		t.Fatalf("Load() (process B) error = %v", err)
	}
	defer coreB.Close()

	results, err := coreB.Search("External")
	if err != nil {
		t.Fatalf("Search() (process B) error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "ext1" {
		t.Fatalf("Search() (process B) = %v, want [ext1]: external edit not reflected", results)
	}
}

// TestSearch_ConcurrentCoresDegradeGracefully forces AC-07/SC-04's
// concurrency, rather than hoping for it: two Core instances over the same
// store -- and therefore the same persisted index directory -- run Search
// at the same time via a barrier, without either closing first. Both must
// return correct results with no hang, crash, or error; at most one of them
// can be holding the persisted index's lock, so at least one is proven to
// be running on the in-memory fallback.
func TestSearch_ConcurrentCoresDegradeGracefully(t *testing.T) {
	coreA, beansDir := setupTestCore(t)
	defer coreA.Close()
	if err := coreA.Create(&bean.Bean{ID: "aaa1", Slug: "a", Title: "Concurrent Alpha", Body: "x"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	coreB := New(beansDir, coreA.Config())
	coreB.SetWarnWriter(nil)
	if err := coreB.Load(); err != nil {
		t.Fatalf("Load() (process B) error = %v", err)
	}
	defer coreB.Close()

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make([][]*bean.Bean, 2)
	errs := make([]error, 2)
	for i, c := range []*Core{coreA, coreB} {
		wg.Add(1)
		go func(i int, c *Core) {
			defer wg.Done()
			<-start
			results[i], errs[i] = c.Search("Concurrent")
		}(i, c)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("Search() core %d error = %v", i, err)
		}
		if len(results[i]) != 1 || results[i][0].ID != "aaa1" {
			t.Fatalf("Search() core %d = %v, want [aaa1]", i, results[i])
		}
	}
}

// TestSearch_SurvivesDeletedPersistedIndex proves AC-05/SC-03 at the Core
// level: deleting the persisted index directory out from under a running
// process still yields a correct search, at the cost of one rebuild, and
// does not affect non-search reads like All()/Get().
func TestSearch_SurvivesDeletedPersistedIndex(t *testing.T) {
	core, _ := setupTestCore(t)
	defer core.Close()
	if err := core.Create(&bean.Bean{ID: "aaa1", Slug: "a", Title: "Findable", Body: "x"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := core.Search("Findable"); err != nil {
		t.Fatalf("Search() #1 error = %v", err)
	}

	dir, err := core.indexDir()
	if err != nil {
		t.Fatalf("indexDir() error = %v", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	if all := core.All(); len(all) != 1 {
		t.Fatalf("All() = %d beans, want 1: unaffected by index deletion", len(all))
	}
	if _, err := core.Get("aaa1"); err != nil {
		t.Fatalf("Get() error = %v: unaffected by index deletion", err)
	}

	// The in-process index handle is still the one built before deletion;
	// deletion under a live process is expected to only matter on the next
	// open (a fresh Core, simulating the next invocation).
	core2 := New(core.Root(), core.Config())
	core2.SetWarnWriter(nil)
	if err := core2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	defer core2.Close()
	results, err := core2.Search("Findable")
	if err != nil {
		t.Fatalf("Search() after deletion error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "aaa1" {
		t.Fatalf("Search() after deletion = %v, want [aaa1]", results)
	}
}
