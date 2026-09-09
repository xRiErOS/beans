package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/config"
)

// candidateFixture builds a small bean tree used to characterise the six
// candidate producers before (and, unchanged, after) they move into a
// shared package:
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
// actually orders by count; see the rationale on TestCollectTagsWithCounts.
func candidateFixture(t *testing.T) (*beangraph.CoreResolver, *config.Config) {
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

func parentPickerBeanIDs(t *testing.T, m parentPickerModel) []string {
	t.Helper()
	var ids []string
	for _, it := range m.list.Items() {
		if pi, ok := it.(parentItem); ok {
			ids = append(ids, pi.bean.ID)
		}
	}
	return ids
}

func blockingPickerBeanIDs(t *testing.T, m blockingPickerModel) []string {
	t.Helper()
	var ids []string
	for _, it := range m.list.Items() {
		if bi, ok := it.(blockingItem); ok {
			ids = append(ids, bi.bean.ID)
		}
	}
	return ids
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

// TestParentPickerCandidates_ExcludesDescendants pins the parent
// producer's content, order and exclusions: it must offer only valid
// parent types, sorted by type rank then title, excluding the edited
// bean's descendants. (Self-exclusion is not independently observable
// here: under the strict type-rank hierarchy a bean's own type is never a
// valid parent type for itself, so the selected-bean skip never changes
// the result and cannot be pinned by a passing/failing assertion.)
func TestParentPickerCandidates_ExcludesDescendants(t *testing.T) {
	resolver, cfg := candidateFixture(t)

	m := newParentPickerModel([]string{"beans-f1"}, "Feature A", []string{"feature"}, "", resolver, cfg, 100, 40)

	got := parentPickerBeanIDs(t, m)
	want := []string{"beans-m1", "beans-m2", "beans-e1", "beans-e2"}
	mustEqual(t, got, want)

	for _, excluded := range []string{"beans-f1", "beans-t1", "beans-f2", "beans-m3"} {
		for _, id := range got {
			if id == excluded {
				t.Fatalf("expected %s to be excluded from parent candidates, got %v", excluded, got)
			}
		}
	}
}

// TestBlockingPickerCandidates_ExcludesDescendants pins the blocking
// producer's parity with the parent producer: it excludes both the bean
// itself and its descendants.
func TestBlockingPickerCandidates_ExcludesDescendants(t *testing.T) {
	resolver, cfg := candidateFixture(t)

	m := newBlockingPickerModel("beans-f1", "Feature A", nil, resolver, cfg, 100, 40)

	got := blockingPickerBeanIDs(t, m)
	want := []string{"beans-m1", "beans-m2", "beans-e1", "beans-e2", "beans-f2"}
	mustEqual(t, got, want)

	for _, excluded := range []string{"beans-f1", "beans-t1", "beans-m3"} {
		for _, id := range got {
			if id == excluded {
				t.Fatalf("expected %s to be excluded from blocking candidates, got %v", excluded, got)
			}
		}
	}
}

// TestStatusPickerCandidates pins the status producer's content and order
// against config.DefaultStatuses.
func TestStatusPickerCandidates(t *testing.T) {
	_, cfg := candidateFixture(t)

	m := newStatusPickerModel([]string{"beans-f1"}, "Feature A", "todo", cfg, 100, 40)

	items := m.list.Items()
	if len(items) != len(config.DefaultStatuses) {
		t.Fatalf("got %d items, want %d", len(items), len(config.DefaultStatuses))
	}
	for i, want := range config.DefaultStatuses {
		si, ok := items[i].(statusItem)
		if !ok {
			t.Fatalf("item %d is not a statusItem", i)
		}
		if si.name != want.Name {
			t.Fatalf("item %d name = %q, want %q", i, si.name, want.Name)
		}
		if si.isCurrent != (want.Name == "todo") {
			t.Fatalf("item %d (%s) isCurrent = %v, want %v", i, si.name, si.isCurrent, want.Name == "todo")
		}
	}
}

// TestTypePickerCandidates pins the type producer's content and order
// against config.DefaultTypes.
func TestTypePickerCandidates(t *testing.T) {
	_, cfg := candidateFixture(t)

	m := newTypePickerModel([]string{"beans-f1"}, "Feature A", "feature", cfg, 100, 40)

	items := m.list.Items()
	if len(items) != len(config.DefaultTypes) {
		t.Fatalf("got %d items, want %d", len(items), len(config.DefaultTypes))
	}
	for i, want := range config.DefaultTypes {
		ti, ok := items[i].(typeItem)
		if !ok {
			t.Fatalf("item %d is not a typeItem", i)
		}
		if ti.name != want.Name {
			t.Fatalf("item %d name = %q, want %q", i, ti.name, want.Name)
		}
		if ti.isCurrent != (want.Name == "feature") {
			t.Fatalf("item %d (%s) isCurrent = %v, want %v", i, ti.name, ti.isCurrent, want.Name == "feature")
		}
	}
}

// TestPriorityPickerCandidates pins the priority producer's content and
// order against config.DefaultPriorities.
func TestPriorityPickerCandidates(t *testing.T) {
	_, cfg := candidateFixture(t)

	m := newPriorityPickerModel([]string{"beans-f1"}, "Feature A", "high", cfg, 100, 40)

	items := m.list.Items()
	if len(items) != len(config.DefaultPriorities) {
		t.Fatalf("got %d items, want %d", len(items), len(config.DefaultPriorities))
	}
	for i, want := range config.DefaultPriorities {
		pi, ok := items[i].(priorityItem)
		if !ok {
			t.Fatalf("item %d is not a priorityItem", i)
		}
		if pi.name != want.Name {
			t.Fatalf("item %d name = %q, want %q", i, pi.name, want.Name)
		}
		if pi.isCurrent != (want.Name == "high") {
			t.Fatalf("item %d (%s) isCurrent = %v, want %v", i, pi.name, pi.isCurrent, want.Name == "high")
		}
	}
}

// TestCollectTagsWithCounts pins the tag producer's content and order:
// every tag in use across all beans, with an accurate usage count,
// rendered in the count-desc/tag-asc order the tag picker relies on
// (internal/tui no longer sorts locally; the order comes straight from
// pkg/candidates.TagCandidates). The fixture gives five tags five
// pairwise-distinct counts (echo=5, delta=4, bravo=3, alpha=2,
// charlie=1), so the expected order [echo, delta, bravo, alpha,
// charlie] is neither alphabetical nor any other structurally simple
// permutation of the tag set — see pkg/candidates.TestTagCandidates for
// the full rationale (this is the same fixture, mirrored).
func TestCollectTagsWithCounts(t *testing.T) {
	resolver, cfg := candidateFixture(t)

	core := resolver.Core
	a := New(core, cfg)

	tags := a.collectTagsWithCounts()

	wantOrder := []string{"echo", "delta", "bravo", "alpha", "charlie"}
	wantCounts := map[string]int{"echo": 5, "delta": 4, "bravo": 3, "alpha": 2, "charlie": 1}
	if len(tags) != len(wantOrder) {
		t.Fatalf("got %v, want tags %v", tags, wantOrder)
	}
	for i, tc := range tags {
		if tc.tag != wantOrder[i] {
			t.Fatalf("tag %d = %q, want %q (got order %v)", i, tc.tag, wantOrder[i], tags)
		}
		if tc.count != wantCounts[tc.tag] {
			t.Fatalf("tag %q count = %d, want %d", tc.tag, tc.count, wantCounts[tc.tag])
		}
	}
}
