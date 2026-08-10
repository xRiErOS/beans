package bean

import (
	"testing"
	"time"
)

func TestSortRoadmapContainers(t *testing.T) {
	statusNames := []string{"draft", "todo", "in-progress", "completed"}
	priorityNames := []string{"critical", "high", "normal", "low", "deferred"}

	t.Run("status stays the outermost level", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "A", Status: "completed"},
			{ID: "2", Title: "B", Status: "todo"},
			{ID: "3", Title: "C", Status: "draft"},
		}

		SortRoadmapContainers(beans, statusNames, priorityNames)

		if beans[0].Status != "draft" || beans[1].Status != "todo" || beans[2].Status != "completed" {
			t.Errorf("statuses = [%q, %q, %q], want [draft, todo, completed]", beans[0].Status, beans[1].Status, beans[2].Status)
		}
	})

	t.Run("manual order respected within same status", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "Second", Status: "todo", Order: "m"},
			{ID: "2", Title: "First", Status: "todo", Order: "a"},
		}

		SortRoadmapContainers(beans, statusNames, priorityNames)

		if beans[0].Title != "First" || beans[1].Title != "Second" {
			t.Errorf("order = [%q, %q], want [First, Second]", beans[0].Title, beans[1].Title)
		}
	})

	t.Run("blocker sorts before blocked even against order and created_at", func(t *testing.T) {
		older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		newer := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		beans := []*Bean{
			// "Blocked" has the earlier order key and older created_at, but is
			// blocked by "Blocker" -- dependency must win over both.
			{ID: "blocked", Title: "Blocked", Status: "todo", Order: "a", CreatedAt: &older, BlockedBy: []string{"blocker"}},
			{ID: "blocker", Title: "Blocker", Status: "todo", Order: "z", CreatedAt: &newer, Blocking: []string{"blocked"}},
		}

		SortRoadmapContainers(beans, statusNames, priorityNames)

		if beans[0].ID != "blocker" || beans[1].ID != "blocked" {
			t.Errorf("order = [%q, %q], want [blocker, blocked]", beans[0].ID, beans[1].ID)
		}
	})

	t.Run("dependency chain A blocks B blocks C resolves to A, B, C", func(t *testing.T) {
		beans := []*Bean{
			{ID: "c", Title: "C", Status: "todo", BlockedBy: []string{"b"}},
			{ID: "a", Title: "A", Status: "todo", Blocking: []string{"b"}},
			{ID: "b", Title: "B", Status: "todo", BlockedBy: []string{"a"}, Blocking: []string{"c"}},
		}

		SortRoadmapContainers(beans, statusNames, priorityNames)

		got := []string{beans[0].ID, beans[1].ID, beans[2].ID}
		want := []string{"a", "b", "c"}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("order = %v, want %v", got, want)
				break
			}
		}
	})

	t.Run("cycle does not hang and is broken deterministically", func(t *testing.T) {
		beans := []*Bean{
			{ID: "a", Title: "A", Status: "todo", Order: "b", Blocking: []string{"b"}, BlockedBy: []string{"b"}},
			{ID: "b", Title: "B", Status: "todo", Order: "a", Blocking: []string{"a"}, BlockedBy: []string{"a"}},
		}

		done := make(chan struct{})
		go func() {
			SortRoadmapContainers(beans, statusNames, priorityNames)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("SortRoadmapContainers hung on a dependency cycle")
		}

		if len(beans) != 2 {
			t.Fatalf("expected 2 beans to survive the cycle, got %d", len(beans))
		}
		// Falls back to the order key since the dependency can't be resolved.
		if beans[0].ID != "b" || beans[1].ID != "a" {
			t.Errorf("order = [%q, %q], want [b, a] (tie-break by Order)", beans[0].ID, beans[1].ID)
		}
	})

	t.Run("edge to a bean outside the slice is ignored", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "Only", Status: "todo", BlockedBy: []string{"not-in-slice"}},
		}

		SortRoadmapContainers(beans, statusNames, priorityNames)

		if len(beans) != 1 || beans[0].ID != "1" {
			t.Fatalf("expected the single bean to survive untouched, got %+v", beans)
		}
	})

	t.Run("falls back to priority then created_at without an order key", func(t *testing.T) {
		older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		newer := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		beans := []*Bean{
			{ID: "1", Title: "Newer Normal", Status: "todo", Priority: "normal", CreatedAt: &newer},
			{ID: "2", Title: "High", Status: "todo", Priority: "high", CreatedAt: &newer},
			{ID: "3", Title: "Older Normal", Status: "todo", Priority: "normal", CreatedAt: &older},
		}

		SortRoadmapContainers(beans, statusNames, priorityNames)

		got := []string{beans[0].Title, beans[1].Title, beans[2].Title}
		want := []string{"High", "Older Normal", "Newer Normal"}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("order = %v, want %v", got, want)
				break
			}
		}
	})
}

func TestSortRoadmapLeaves(t *testing.T) {
	statusNames := []string{"draft", "todo", "in-progress", "completed"}
	priorityNames := []string{"critical", "high", "normal", "low", "deferred"}
	typeNames := []string{"bug", "feature", "task"}

	t.Run("type stays the outermost level, ahead of status", func(t *testing.T) {
		beans := []*Bean{
			{ID: "1", Title: "Task in todo", Status: "todo", Type: "task"},
			{ID: "2", Title: "Bug in completed", Status: "completed", Type: "bug"},
			{ID: "3", Title: "Bug in todo", Status: "todo", Type: "bug"},
		}

		SortRoadmapLeaves(beans, statusNames, priorityNames, typeNames)

		if beans[0].Type != "bug" || beans[1].Type != "bug" || beans[2].Type != "task" {
			t.Fatalf("types = [%q, %q, %q], want [bug, bug, task]", beans[0].Type, beans[1].Type, beans[2].Type)
		}
		// Within the "bug" bucket, status still orders todo before completed.
		if beans[0].Status != "todo" || beans[1].Status != "completed" {
			t.Errorf("bug statuses = [%q, %q], want [todo, completed]", beans[0].Status, beans[1].Status)
		}
	})
}
