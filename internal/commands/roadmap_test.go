package commands

import (
	"testing"
	"time"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/config"
)

// mockConfig implements the StatusNames interface for testing.
type mockConfig struct {
	statuses []string
	archive  map[string]bool
}

func (m *mockConfig) StatusNames() []string {
	return m.statuses
}

func (m *mockConfig) IsArchiveStatus(s string) bool {
	return m.archive[s]
}

func TestBuildRoadmap(t *testing.T) {
	// Save and restore global cfg
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	// Statuses are now hardcoded
	cfg = config.Default()

	now := time.Now()

	tests := []struct {
		name                  string
		beans                 []*bean.Bean
		includeDone           bool
		wantMilestones        int
		wantUnscheduledEpics  int
		wantUnscheduledOther  int
	}{
		{
			name:           "empty beans",
			beans:          []*bean.Bean{},
			wantMilestones: 0,
		},
		{
			name: "milestone with epic and items",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Parent: "m1"},
				{ID: "t1", Type: "task", Title: "Login", Status: "todo", Parent: "e1"},
			},
			wantMilestones: 1,
		},
		{
			name: "milestone with direct children (no epic)",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Docs", Status: "todo", Parent: "m1"},
			},
			wantMilestones: 1,
		},
		{
			name: "unscheduled epic",
			beans: []*bean.Bean{
				{ID: "e1", Type: "epic", Title: "Future", Status: "todo"},
				{ID: "t1", Type: "task", Title: "Nice to have", Status: "todo", Parent: "e1"},
			},
			wantMilestones:       0,
			wantUnscheduledEpics: 1,
		},
		{
			name: "done items excluded by default",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Done task", Status: "completed", Parent: "m1"},
			},
			includeDone:    false,
			wantMilestones: 0, // milestone has no visible children
		},
		{
			name: "done items included when requested",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Done task", Status: "completed", Parent: "m1"},
			},
			includeDone:    true,
			wantMilestones: 1,
		},
		{
			name: "orphan bean appears in unscheduled other",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "t1", Type: "task", Title: "Orphan", Status: "todo"}, // no parent link
			},
			wantMilestones:       0, // milestone has no children
			wantUnscheduledOther: 1, // orphan appears in unscheduled
		},
		{
			name: "leaf nested under feature under epic under milestone is not lost",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Parent: "m1"},
				{ID: "f1", Type: "feature", Title: "SSO", Status: "todo", Parent: "e1"},
				{ID: "t1", Type: "task", Title: "OIDC login", Status: "todo", Parent: "f1"},
			},
			wantMilestones: 1,
		},
		{
			name: "milestone with direct feature child and no epic is not dropped",
			beans: []*bean.Bean{
				{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
				{ID: "f1", Type: "feature", Title: "SSO", Status: "todo", Parent: "m1"},
				{ID: "t1", Type: "task", Title: "OIDC login", Status: "todo", Parent: "f1"},
			},
			wantMilestones: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildRoadmap(tt.beans, tt.includeDone, nil, nil)

			if got := len(result.Milestones); got != tt.wantMilestones {
				t.Errorf("got %d milestones, want %d", got, tt.wantMilestones)
			}

			gotUnscheduledEpics := 0
			gotUnscheduledOther := 0
			if result.Unscheduled != nil {
				gotUnscheduledEpics = len(result.Unscheduled.Epics)
				gotUnscheduledOther = len(result.Unscheduled.Other)
			}
			if gotUnscheduledEpics != tt.wantUnscheduledEpics {
				t.Errorf("got %d unscheduled epics, want %d", gotUnscheduledEpics, tt.wantUnscheduledEpics)
			}
			if gotUnscheduledOther != tt.wantUnscheduledOther {
				t.Errorf("got %d unscheduled other, want %d", gotUnscheduledOther, tt.wantUnscheduledOther)
			}
		})
	}
}

func TestFirstParagraph(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		want  string
	}{
		{
			name: "empty body",
			body: "",
			want: "",
		},
		{
			name: "single line",
			body: "This is a description.",
			want: "This is a description.",
		},
		{
			name: "multiple paragraphs",
			body: "First paragraph.\n\nSecond paragraph.",
			want: "First paragraph.",
		},
		{
			name: "multiline first paragraph",
			body: "Line one\nLine two\n\nSecond para.",
			want: "Line one Line two",
		},
		{
			name: "skips headers at start",
			body: "## Checklist\n- item one",
			want: "- item one",
		},
		{
			name: "truncates long text",
			body: "This is a very long paragraph that exceeds two hundred characters and needs to be truncated so it does not take up too much space in the roadmap output. Lorem ipsum dolor sit amet consectetur adipiscing elit.",
			want: "This is a very long paragraph that exceeds two hundred characters and needs to be truncated so it does not take up too much space in the roadmap output. Lorem ipsum dolor sit amet consectetur adipi...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstParagraph(tt.body)
			if got != tt.want {
				t.Errorf("firstParagraph() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderBeanRef(t *testing.T) {
	tests := []struct {
		name       string
		bean       *bean.Bean
		asLink     bool
		linkPrefix string
		want       string
	}{
		{
			name:   "no link - just ID",
			bean:   &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink: false,
			want:   "(abc)",
		},
		{
			name:       "link without prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: "",
			want:       "([abc](abc--milestone.md))",
		},
		{
			name:       "link with prefix",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: "https://example.com/beans/",
			want:       "([abc](https://example.com/beans/abc--milestone.md))",
		},
		{
			name:       "link with prefix without trailing slash",
			bean:       &bean.Bean{ID: "abc", Path: "abc--milestone.md"},
			asLink:     true,
			linkPrefix: ".beans",
			want:       "([abc](.beans/abc--milestone.md))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderBeanRef(tt.bean, tt.asLink, tt.linkPrefix)
			if got != tt.want {
				t.Errorf("renderBeanRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusFiltering(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	// Statuses are now hardcoded
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "Todo Milestone", Status: "todo", CreatedAt: &now},
		{ID: "m2", Type: "milestone", Title: "In Progress Milestone", Status: "in-progress", CreatedAt: &now},
		{ID: "t1", Type: "task", Title: "Task 1", Status: "todo", Parent: "m1"},
		{ID: "t2", Type: "task", Title: "Task 2", Status: "todo", Parent: "m2"},
	}

	t.Run("filter by status", func(t *testing.T) {
		result := buildRoadmap(beans, false, []string{"todo"}, nil)
		if len(result.Milestones) != 1 {
			t.Errorf("expected 1 milestone, got %d", len(result.Milestones))
		}
		if result.Milestones[0].Milestone.Status != "todo" {
			t.Errorf("expected todo milestone, got %s", result.Milestones[0].Milestone.Status)
		}
	})

	t.Run("exclude by status", func(t *testing.T) {
		result := buildRoadmap(beans, false, nil, []string{"in-progress"})
		if len(result.Milestones) != 1 {
			t.Errorf("expected 1 milestone, got %d", len(result.Milestones))
		}
		if result.Milestones[0].Milestone.Status != "todo" {
			t.Errorf("expected todo milestone, got %s", result.Milestones[0].Milestone.Status)
		}
	})
}

func TestSplitByContainerType(t *testing.T) {
	beans := []*bean.Bean{
		{ID: "f1", Type: "feature", Title: "F1"},
		{ID: "t1", Type: "task", Title: "T1"},
		{ID: "b1", Type: "bug", Title: "B1"},
	}

	leafs, features := splitByContainerType(beans)

	if len(leafs) != 2 {
		t.Errorf("got %d leafs, want 2", len(leafs))
	}
	if len(features) != 1 {
		t.Errorf("got %d features, want 1", len(features))
	}
	if features[0].ID != "f1" {
		t.Errorf("got feature %s, want f1", features[0].ID)
	}
}

func TestCollectLeafDescendants(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	children := map[string][]*bean.Bean{
		"feat1": {
			{ID: "t1", Type: "task", Title: "Direct leaf", Status: "todo", Parent: "feat1"},
			{ID: "feat2", Type: "feature", Title: "Nested feature", Status: "todo", Parent: "feat1"},
		},
		"feat2": {
			{ID: "t2", Type: "task", Title: "Nested leaf", Status: "todo", Parent: "feat2"},
			{ID: "t3", Type: "task", Title: "Done nested leaf", Status: "completed", Parent: "feat2"},
		},
	}

	t.Run("flattens through nested features, excludes done by default", func(t *testing.T) {
		got := collectLeafDescendants("feat1", children, false)
		if len(got) != 2 {
			t.Fatalf("got %d leafs, want 2 (t1, t2)", len(got))
		}
		ids := map[string]bool{got[0].ID: true, got[1].ID: true}
		if !ids["t1"] || !ids["t2"] {
			t.Errorf("got ids %v, want t1 and t2", ids)
		}
	})

	t.Run("includes done when requested", func(t *testing.T) {
		got := collectLeafDescendants("feat1", children, true)
		if len(got) != 3 {
			t.Fatalf("got %d leafs, want 3 (t1, t2, t3)", len(got))
		}
	})

	t.Run("no children returns empty, not nil panic", func(t *testing.T) {
		got := collectLeafDescendants("nonexistent", children, false)
		if len(got) != 0 {
			t.Errorf("got %d leafs, want 0", len(got))
		}
	})

	t.Run("hand-authored parent cycle does not stack-overflow", func(t *testing.T) {
		// The CLI's ValidateParent/DetectCycle reject this at write time, but
		// beans are hand-editable markdown -- a manually edited cycle must not
		// crash roadmap generation. This subtest hangs/panics without the
		// visited guard in collectLeafDescendantsVisited.
		cyclic := map[string][]*bean.Bean{
			"featA": {
				{ID: "featB", Type: "feature", Title: "B", Status: "todo", Parent: "featA"},
			},
			"featB": {
				{ID: "featA", Type: "feature", Title: "A", Status: "todo", Parent: "featB"},
				{ID: "t1", Type: "task", Title: "Reachable leaf", Status: "todo", Parent: "featB"},
			},
		}
		got := collectLeafDescendants("featA", cyclic, false)
		if len(got) != 1 || got[0].ID != "t1" {
			t.Errorf("got %v, want exactly [t1]", got)
		}
	})
}

func TestBuildMilestoneGroupResolvesFeatureNesting(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
		{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Parent: "m1"},
		{ID: "f1", Type: "feature", Title: "SSO", Status: "todo", Parent: "e1"},
		{ID: "t1", Type: "task", Title: "OIDC login", Status: "todo", Parent: "f1"},
		{ID: "b1", Type: "bug", Title: "Direct epic bug", Status: "todo", Parent: "e1"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Milestones) != 1 {
		t.Fatalf("got %d milestones, want 1", len(result.Milestones))
	}
	epics := result.Milestones[0].Epics
	if len(epics) != 1 {
		t.Fatalf("got %d epics, want 1", len(epics))
	}
	epic := epics[0]
	if len(epic.Items) != 1 || epic.Items[0].ID != "b1" {
		t.Errorf("epic.Items = %v, want [b1]", epic.Items)
	}
	if len(epic.Features) != 1 {
		t.Fatalf("got %d feature groups, want 1", len(epic.Features))
	}
	if epic.Features[0].Feature.ID != "f1" {
		t.Errorf("feature group is for %s, want f1", epic.Features[0].Feature.ID)
	}
	if len(epic.Features[0].Items) != 1 || epic.Features[0].Items[0].ID != "t1" {
		t.Errorf("feature.Items = %v, want [t1]", epic.Features[0].Items)
	}
}

func TestUnscheduledFeatureResolvesNesting(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	beans := []*bean.Bean{
		{ID: "f1", Type: "feature", Title: "Standalone feature", Status: "todo"},
		{ID: "t1", Type: "task", Title: "Leaf under orphan feature", Status: "todo", Parent: "f1"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if result.Unscheduled == nil {
		t.Fatal("expected Unscheduled to be non-nil")
	}
	if len(result.Unscheduled.Features) != 1 {
		t.Fatalf("got %d unscheduled features, want 1", len(result.Unscheduled.Features))
	}
	fg := result.Unscheduled.Features[0]
	if fg.Feature.ID != "f1" {
		t.Errorf("feature group is for %s, want f1", fg.Feature.ID)
	}
	if len(fg.Items) != 1 || fg.Items[0].ID != "t1" {
		t.Errorf("fg.Items = %v, want [t1]", fg.Items)
	}
	// The leaf must not also leak into Other (it has a parent, so it's
	// handled entirely via the feature group).
	if len(result.Unscheduled.Other) != 0 {
		t.Errorf("got %d unscheduled other, want 0", len(result.Unscheduled.Other))
	}
}

func TestUnscheduledEpicWithFeatureNesting(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	beans := []*bean.Bean{
		{ID: "e1", Type: "epic", Title: "Unscheduled epic", Status: "todo"},
		{ID: "f1", Type: "feature", Title: "Feature under unscheduled epic", Status: "todo", Parent: "e1"},
		{ID: "t1", Type: "task", Title: "Leaf", Status: "todo", Parent: "f1"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Unscheduled.Epics) != 1 {
		t.Fatalf("got %d unscheduled epics, want 1", len(result.Unscheduled.Epics))
	}
	eg := result.Unscheduled.Epics[0]
	if len(eg.Features) != 1 || eg.Features[0].Feature.ID != "f1" {
		t.Fatalf("eg.Features = %+v, want feature f1", eg.Features)
	}
	if len(eg.Features[0].Items) != 1 || eg.Features[0].Items[0].ID != "t1" {
		t.Errorf("eg.Features[0].Items = %v, want [t1]", eg.Features[0].Items)
	}
}

func TestUnscheduledNestedFeatureNotDoubleRendered(t *testing.T) {
	// A feature nested under another orphan feature (hand-edited data --
	// ValidateParent rejects this via the CLI) must be flattened into the
	// top feature's Items exactly once, never also appear as its own
	// top-level unscheduled feature entry.
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	beans := []*bean.Bean{
		{ID: "f1", Type: "feature", Title: "Top feature", Status: "todo"},
		{ID: "f2", Type: "feature", Title: "Nested feature", Status: "todo", Parent: "f1"},
		{ID: "t1", Type: "task", Title: "Leaf", Status: "todo", Parent: "f2"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Unscheduled.Features) != 1 {
		t.Fatalf("got %d unscheduled features, want 1 (f1 only, f2 must not double-render)", len(result.Unscheduled.Features))
	}
	fg := result.Unscheduled.Features[0]
	if fg.Feature.ID != "f1" {
		t.Errorf("unscheduled feature is %s, want f1", fg.Feature.ID)
	}
	if len(fg.Items) != 1 || fg.Items[0].ID != "t1" {
		t.Errorf("fg.Items = %v, want [t1]", fg.Items)
	}
}
