package candidates

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/config"
)

// fixture builds a small bean tree:
//
//	beans-m1 "Milestone A" (milestone)                                tags: echo, delta
//	beans-m2 "Milestone B" (milestone)                                tags: echo, delta
//	beans-e1 "Epic A"      (epic, parent beans-m1)                    tags: echo, delta
//	beans-e2 "Epic B"      (epic)                                     tags: echo, delta
//	beans-f1 "Feature A"   (feature, parent beans-e1) tags: alpha, bravo, echo
//	beans-f2 "Feature B"   (feature, parent beans-e2) tags: alpha
//	beans-t1 "Task A"      (task, parent beans-f1)    tags: bravo, charlie
//	beans-m3 "Milestone C" (milestone, parent beans-t1; a descendant of
//	                        beans-f1 whose type would otherwise be a valid
//	                        parent candidate, so it can only be kept out of
//	                        ParentCandidates by the descendant exclusion)
//	                        tags: bravo
//
// Tag counts: echo=5, delta=4, bravo=3, alpha=2, charlie=1 — five
// pairwise-distinct counts, so the expected tag order [echo, delta,
// bravo, alpha, charlie] can only be produced by a comparator that
// actually orders by count; see the rationale on TestTagCandidates.
func fixture(t *testing.T) (*beangraph.CoreResolver, *config.Config) {
	t.Helper()
	beansDir := filepath.Join(t.TempDir(), ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("creating test .beans dir: %v", err)
	}
	cfg := config.Default()
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatalf("loading core: %v", err)
	}

	beans := []*bean.Bean{
		{ID: "beans-m1", Slug: bean.Slugify("Milestone A"), Title: "Milestone A", Status: "todo", Type: "milestone", Tags: []string{"echo", "delta"}},
		{ID: "beans-m2", Slug: bean.Slugify("Milestone B"), Title: "Milestone B", Status: "todo", Type: "milestone", Tags: []string{"echo", "delta"}},
		{ID: "beans-e1", Slug: bean.Slugify("Epic A"), Title: "Epic A", Status: "todo", Type: "epic", Parent: "beans-m1", Tags: []string{"echo", "delta"}},
		{ID: "beans-e2", Slug: bean.Slugify("Epic B"), Title: "Epic B", Status: "todo", Type: "epic", Tags: []string{"echo", "delta"}},
		{ID: "beans-f1", Slug: bean.Slugify("Feature A"), Title: "Feature A", Status: "todo", Type: "feature", Parent: "beans-e1", Tags: []string{"alpha", "bravo", "echo"}},
		{ID: "beans-f2", Slug: bean.Slugify("Feature B"), Title: "Feature B", Status: "todo", Type: "feature", Parent: "beans-e2", Tags: []string{"alpha"}},
		{ID: "beans-t1", Slug: bean.Slugify("Task A"), Title: "Task A", Status: "todo", Type: "task", Parent: "beans-f1", Tags: []string{"bravo", "charlie"}},
		{ID: "beans-m3", Slug: bean.Slugify("Milestone C"), Title: "Milestone C", Status: "todo", Type: "milestone", Parent: "beans-t1", Tags: []string{"bravo"}},
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("core.Create(%s): %v", b.ID, err)
		}
	}

	return &beangraph.CoreResolver{Core: core}, cfg
}

func ids(beans []*bean.Bean) []string {
	out := make([]string, len(beans))
	for i, b := range beans {
		out[i] = b.ID
	}
	return out
}

func mustEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestParentCandidates_ExcludesDescendants pins ParentCandidates'
// content, order and exclusions: only valid parent types, sorted by type
// rank then title, excluding the edited bean's descendants.
//
// Self-exclusion is not independently observable here: under the strict
// type-rank hierarchy a bean's own type is never a valid parent type for
// itself, so the selected-bean skip never changes the result and cannot
// be pinned by a passing/failing assertion.
func TestParentCandidates_ExcludesDescendants(t *testing.T) {
	resolver, cfg := fixture(t)

	got, err := ParentCandidates(context.Background(), resolver, cfg, []string{"beans-f1"}, []string{"feature"})
	if err != nil {
		t.Fatalf("ParentCandidates: %v", err)
	}

	mustEqual(t, ids(got), []string{"beans-m1", "beans-m2", "beans-e1", "beans-e2"})

	for _, excluded := range []string{"beans-f1", "beans-t1", "beans-f2", "beans-m3"} {
		for _, id := range ids(got) {
			if id == excluded {
				t.Fatalf("expected %s to be excluded from parent candidates, got %v", excluded, ids(got))
			}
		}
	}
}

func TestBlockingCandidates_IncludesDescendants(t *testing.T) {
	resolver, cfg := fixture(t)

	got, err := BlockingCandidates(context.Background(), resolver, cfg, "beans-f1")
	if err != nil {
		t.Fatalf("BlockingCandidates: %v", err)
	}

	mustEqual(t, ids(got), []string{"beans-m1", "beans-m2", "beans-m3", "beans-e1", "beans-e2", "beans-f2", "beans-t1"})
}

func TestStatusCandidates(t *testing.T) {
	got := StatusCandidates()
	if len(got) != len(config.DefaultStatuses) {
		t.Fatalf("got %d statuses, want %d", len(got), len(config.DefaultStatuses))
	}
	for i, s := range config.DefaultStatuses {
		if got[i].Name != s.Name {
			t.Fatalf("status %d = %q, want %q", i, got[i].Name, s.Name)
		}
	}
}

func TestTypeCandidates(t *testing.T) {
	got := TypeCandidates()
	if len(got) != len(config.DefaultTypes) {
		t.Fatalf("got %d types, want %d", len(got), len(config.DefaultTypes))
	}
	for i, ty := range config.DefaultTypes {
		if got[i].Name != ty.Name {
			t.Fatalf("type %d = %q, want %q", i, got[i].Name, ty.Name)
		}
	}
}

func TestPriorityCandidates(t *testing.T) {
	got := PriorityCandidates()
	if len(got) != len(config.DefaultPriorities) {
		t.Fatalf("got %d priorities, want %d", len(got), len(config.DefaultPriorities))
	}
	for i, p := range config.DefaultPriorities {
		if got[i].Name != p.Name {
			t.Fatalf("priority %d = %q, want %q", i, got[i].Name, p.Name)
		}
	}
}

// TestTagCandidates pins TagCandidates' content and order: every tag in
// use across all beans with an accurate usage count, sorted by count
// descending then tag ascending. The fixture gives five tags five
// pairwise-distinct counts (echo=5, delta=4, bravo=3, alpha=2,
// charlie=1), so the expected order [echo, delta, bravo, alpha,
// charlie] is neither alphabetical nor any other structurally simple
// permutation of the tag set: it is reachable only by a comparator that
// actually orders by count. A comparator that dropped the count
// comparison (falling back to pure tag order) would deterministically
// produce [alpha, bravo, charlie, delta, echo] instead, differing on a
// single run. A comparator that ordered nothing at all leaves the
// result at the mercy of Go's per-call-randomized map iteration, whose
// reachable orders (rotations of one process-wide hash-bucket layout)
// are a small, fixed subset of all 5! permutations; the 10x repeated
// -call loop requires the exact expected order on all 10 independently
// randomized iterations, which the widened, non-trivial target makes an
// accidental full pass vanishingly unlikely without claiming it is
// mathematically impossible.
func TestTagCandidates(t *testing.T) {
	resolver, _ := fixture(t)

	wantTags := []string{"echo", "delta", "bravo", "alpha", "charlie"}
	wantCounts := map[string]int{"echo": 5, "delta": 4, "bravo": 3, "alpha": 2, "charlie": 1}

	for run := range 10 {
		got, err := TagCandidates(context.Background(), resolver)
		if err != nil {
			t.Fatalf("run %d: TagCandidates: %v", run, err)
		}
		if len(got) != len(wantTags) {
			t.Fatalf("run %d: got %v, want tags %v", run, got, wantTags)
		}
		for i, tc := range got {
			if tc.Tag != wantTags[i] {
				t.Fatalf("run %d: tag %d = %q, want %q (got order %v)", run, i, tc.Tag, wantTags[i], got)
			}
			if tc.Count != wantCounts[tc.Tag] {
				t.Fatalf("run %d: tag %q count = %d, want %d", run, tc.Tag, tc.Count, wantCounts[tc.Tag])
			}
		}
	}
}

// TestParentCandidates_IntersectsValidTypesAcrossMultipleSelectedBeans
// pins intersectStrings' role in a multi-select ParentCandidates call:
// with beans-e1 (epic) and beans-f1 (feature) selected together, the
// valid parent types must be the intersection of each bean's own valid
// parent types (epic -> {milestone}, feature -> {milestone, epic}),
// leaving only milestone. Without the intersection, epic-typed beans
// (e.g. beans-e2) would leak in as candidates too. Checked in both
// argument orders since intersection is commutative.
func TestParentCandidates_IntersectsValidTypesAcrossMultipleSelectedBeans(t *testing.T) {
	resolver, cfg := fixture(t)

	for _, beanIDs := range [][]string{
		{"beans-e1", "beans-f1"},
		{"beans-f1", "beans-e1"},
	} {
		beanTypes := []string{"epic", "feature"}
		if beanIDs[0] == "beans-f1" {
			beanTypes = []string{"feature", "epic"}
		}

		got, err := ParentCandidates(context.Background(), resolver, cfg, beanIDs, beanTypes)
		if err != nil {
			t.Fatalf("ParentCandidates(%v, %v): %v", beanIDs, beanTypes, err)
		}

		mustEqual(t, ids(got), []string{"beans-m1", "beans-m2"})

		for _, id := range ids(got) {
			if id == "beans-e2" {
				t.Fatalf("beanIDs=%v: expected beans-e2 (epic, outside the type intersection) to be excluded, got %v", beanIDs, ids(got))
			}
		}
	}
}
