package commands

import (
	"regexp"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

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

func TestMilestoneOrderRespectsManualOrderKey(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = config.Default()

	older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	beans := []*bean.Bean{
		// Created later but manually ordered first via the "order" key.
		{ID: "m2", Type: "milestone", Title: "v2.0", Status: "todo", Order: "a", CreatedAt: &newer},
		{ID: "t2", Type: "task", Title: "Task 2", Status: "todo", Parent: "m2"},
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", Order: "z", CreatedAt: &older},
		{ID: "t1", Type: "task", Title: "Task 1", Status: "todo", Parent: "m1"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Milestones) != 2 {
		t.Fatalf("expected 2 milestones, got %d", len(result.Milestones))
	}
	if result.Milestones[0].Milestone.ID != "m2" || result.Milestones[1].Milestone.ID != "m1" {
		t.Errorf("milestone order = [%q, %q], want [m2, m1] (manual order overrides created_at)",
			result.Milestones[0].Milestone.ID, result.Milestones[1].Milestone.ID)
	}
}

func TestMilestoneOrderRespectsBlockingDependency(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()

	cfg = config.Default()

	// IDs deliberately sort the "wrong" way (aa- before zz-) so the ID
	// fallback alone cannot make this test pass -- only Blocking/BlockedBy
	// awareness can put the blocker first.
	beans := []*bean.Bean{
		{ID: "aa-blocked", Type: "milestone", Title: "Blocked", Status: "todo", BlockedBy: []string{"zz-blocker"}},
		{ID: "t2", Type: "task", Title: "Task 2", Status: "todo", Parent: "aa-blocked"},
		{ID: "zz-blocker", Type: "milestone", Title: "Blocker", Status: "todo", Blocking: []string{"aa-blocked"}},
		{ID: "t1", Type: "task", Title: "Task 1", Status: "todo", Parent: "zz-blocker"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Milestones) != 2 {
		t.Fatalf("expected 2 milestones, got %d", len(result.Milestones))
	}
	if result.Milestones[0].Milestone.ID != "zz-blocker" || result.Milestones[1].Milestone.ID != "aa-blocked" {
		t.Errorf("milestone order = [%q, %q], want [zz-blocker, aa-blocked] (blocker sorts first)",
			result.Milestones[0].Milestone.ID, result.Milestones[1].Milestone.ID)
	}
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
		got := collectLeafDescendants("feat1", children, false, nil)
		if len(got) != 2 {
			t.Fatalf("got %d leafs, want 2 (t1, t2)", len(got))
		}
		ids := map[string]bool{got[0].ID: true, got[1].ID: true}
		if !ids["t1"] || !ids["t2"] {
			t.Errorf("got ids %v, want t1 and t2", ids)
		}
	})

	t.Run("includes done when requested", func(t *testing.T) {
		got := collectLeafDescendants("feat1", children, true, nil)
		if len(got) != 3 {
			t.Fatalf("got %d leafs, want 3 (t1, t2, t3)", len(got))
		}
	})

	t.Run("no children returns empty, not nil panic", func(t *testing.T) {
		got := collectLeafDescendants("nonexistent", children, false, nil)
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
		got := collectLeafDescendants("featA", cyclic, false, nil)
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

// beans-n8zw D01: a Feature is a container IFF it has >=1 leaf descendant
// (respecting includeDone). Childless features must render as flat leaf
// lines instead of vanishing from the roadmap.

// Ta: orphan feature, 0 children, status todo -> flat leaf in
// Unscheduled.Other, never as a featureGroup, never dropped.
func TestOrphanChildlessFeatureAppearsAsFlatLeaf(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	beans := []*bean.Bean{
		{ID: "f1", Type: "feature", Title: "Lonely", Status: "todo"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if result.Unscheduled == nil {
		t.Fatal("expected Unscheduled to be non-nil")
	}
	if len(result.Unscheduled.Features) != 0 {
		t.Errorf("got %d unscheduled featureGroups, want 0 (childless feature must not become a container)", len(result.Unscheduled.Features))
	}
	if len(result.Unscheduled.Other) != 1 || result.Unscheduled.Other[0].ID != "f1" {
		t.Errorf("Unscheduled.Other = %v, want [f1]", result.Unscheduled.Other)
	}
}

// Tb: feature under epic, 0 children -> flat leaf in epic.Items, not in
// epic.Features.
func TestChildlessFeatureUnderEpicAppearsInEpicItems(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
		{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Parent: "m1"},
		{ID: "f1", Type: "feature", Title: "Lonely", Status: "todo", Parent: "e1"},
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
	if len(epic.Features) != 0 {
		t.Errorf("got %d epic feature groups, want 0", len(epic.Features))
	}
	if len(epic.Items) != 1 || epic.Items[0].ID != "f1" {
		t.Errorf("epic.Items = %v, want [f1]", epic.Items)
	}
}

// Tc: feature directly under milestone, 0 children -> flat leaf in
// milestone.Other.
func TestChildlessFeatureDirectUnderMilestoneAppearsInOther(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
		{ID: "f1", Type: "feature", Title: "Lonely", Status: "todo", Parent: "m1"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if len(result.Milestones) != 1 {
		t.Fatalf("got %d milestones, want 1", len(result.Milestones))
	}
	ms := result.Milestones[0]
	if len(ms.Features) != 0 {
		t.Errorf("got %d milestone feature groups, want 0", len(ms.Features))
	}
	if len(ms.Other) != 1 || ms.Other[0].ID != "f1" {
		t.Errorf("milestone.Other = %v, want [f1]", ms.Other)
	}
}

// Td (regression guard): a feature with >=1 live child still renders as a
// featureGroup container, even when a childless sibling feature is present
// in the same epic -- the two code paths must not cross-contaminate.
func TestFeatureWithChildRemainsContainerAlongsideChildlessSibling(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
		{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Parent: "m1"},
		{ID: "f1", Type: "feature", Title: "Childless", Status: "todo", Parent: "e1"},
		{ID: "f2", Type: "feature", Title: "Has a child", Status: "todo", Parent: "e1"},
		{ID: "t1", Type: "task", Title: "OIDC login", Status: "todo", Parent: "f2"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	epic := result.Milestones[0].Epics[0]
	if len(epic.Items) != 1 || epic.Items[0].ID != "f1" {
		t.Errorf("epic.Items = %v, want [f1] (childless feature flattened)", epic.Items)
	}
	if len(epic.Features) != 1 || epic.Features[0].Feature.ID != "f2" {
		t.Fatalf("epic.Features = %+v, want exactly [f2]", epic.Features)
	}
	if len(epic.Features[0].Items) != 1 || epic.Features[0].Items[0].ID != "t1" {
		t.Errorf("epic.Features[0].Items = %v, want [t1]", epic.Features[0].Items)
	}
}

// Te (edge case): a childless feature with an archive status is dropped by
// the normal archive-status filter, exactly like any other leaf.
func TestChildlessCompletedFeatureDroppedByArchiveFilter(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	beans := []*bean.Bean{
		{ID: "f1", Type: "feature", Title: "Done and lonely", Status: "completed"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if result.Unscheduled != nil {
		t.Errorf("expected Unscheduled to be nil (completed childless feature dropped), got %+v", result.Unscheduled)
	}
}

// beans-36fa: a childless orphan epic (type epic, status open, no parent)
// must render as a flat leaf row in the Unscheduled group, analogous to
// childless orphan features (beans-n8zw).
func TestOrphanChildlessEpicAppearsAsFlatLeaf(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	beans := []*bean.Bean{
		{ID: "e1", Type: "epic", Title: "Lonely Epic", Status: "todo"},
	}

	result := buildRoadmap(beans, false, nil, nil)

	if result.Unscheduled == nil {
		t.Fatal("expected Unscheduled to be non-nil")
	}
	if len(result.Unscheduled.Epics) != 0 {
		t.Errorf("got %d unscheduled epicGroups, want 0 (childless epic must not become a container)", len(result.Unscheduled.Epics))
	}
	if len(result.Unscheduled.Other) != 1 || result.Unscheduled.Other[0].ID != "e1" {
		t.Errorf("Unscheduled.Other = %v, want [e1]", result.Unscheduled.Other)
	}
}


// --- T5: roadmapOutput TTY switch -------------------------------------------
//
// roadmapOutputFixture is a standalone roadmapData literal (T4 pattern) --
// NOT prettyFixture(), whose want-literal in TestRenderRoadmapPrettyAt80 is
// the frozen DESIGN.md block and must not be perturbed by an unrelated test.

func roadmapOutputFixture() *roadmapData {
	now := time.Now()
	m := &bean.Bean{ID: "beans-m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now, Path: "m1--v10.md"}
	e := &bean.Bean{ID: "beans-e1", Type: "epic", Title: "Auth", Status: "todo", Path: "e1--auth.md"}
	t1 := &bean.Bean{ID: "beans-t1", Type: "task", Title: "Login", Status: "todo", Priority: "high", Path: "t1--login.md"}
	return &roadmapData{
		Milestones: []milestoneGroup{
			{
				Milestone: m,
				Epics: []epicGroup{
					{Epic: e, Items: []*bean.Bean{t1}},
				},
			},
		},
	}
}

// SC-501: the TTY flag alone must decide which renderer roadmapOutput picks
// -- pipe gets the markdown artifact (badges + links), TTY gets the plain
// table (no badges, no markdown link syntax, glyph tree).
func TestRoadmapOutputSwitchesOnTTY(t *testing.T) {
	data := roadmapOutputFixture()

	t.Run("pipe (non-tty) renders markdown", func(t *testing.T) {
		got := roadmapOutput(data, false, 0, true, "", false)
		if !strings.HasPrefix(got, "# Roadmap") {
			t.Errorf("got prefix %q, want %q", got[:min(len(got), 20)], "# Roadmap")
		}
		if !strings.Contains(got, "img.shields.io") {
			t.Error("expected markdown output to contain an img.shields.io badge")
		}
	})

	t.Run("tty renders plain-text table", func(t *testing.T) {
		got := roadmapOutput(data, true, 80, true, "", false)
		if !strings.HasPrefix(got, "Roadmap") {
			t.Errorf("got prefix %q, want %q", got[:min(len(got), 20)], "Roadmap")
		}
		if strings.Contains(got, "img.shields.io") {
			t.Error("TTY output must not contain shields.io badges (EARS-4)")
		}
		if strings.Contains(got, "](") {
			t.Error("TTY output must not contain markdown link syntax (EARS-4)")
		}
		if !strings.Contains(got, "■ Milestone") {
			t.Error("expected TTY output to contain the milestone glyph line")
		}
	})
}

// TestRoadmapOutputSwitchesOnTTYWithRoot verifies that roadmapOutput correctly
// dispatches Root-scoped data (from buildScopedRoadmap) to both renderers,
// not just the Milestones-based data tested in TestRoadmapOutputSwitchesOnTTY.
// This proves the dispatcher itself handles the Root case, alongside the
// renderer-specific tests (TestRenderRoadmapMarkdownRootEpic, etc.) that test
// each renderer in isolation.
func TestRoadmapOutputSwitchesOnTTYWithRoot(t *testing.T) {
	epic := &bean.Bean{ID: "beans-epic1", Type: "epic", Title: "Auth", Status: "todo", Path: "epic1--auth.md"}
	item := &bean.Bean{ID: "beans-task1", Type: "task", Title: "Login", Status: "todo", Path: "task1--login.md"}
	data := &roadmapData{
		Root: &rootGroup{
			Epic: &epicGroup{Epic: epic, Items: []*bean.Bean{item}},
		},
	}

	t.Run("pipe (non-tty) renders markdown with Root", func(t *testing.T) {
		got := roadmapOutput(data, false, 0, true, "", false)
		if !strings.HasPrefix(got, "# Roadmap") {
			t.Errorf("got prefix %q, want %q", got[:min(len(got), 20)], "# Roadmap")
		}
		if !strings.Contains(got, "### Epic:") {
			t.Errorf("expected epic heading (### Epic:), got %q", got)
		}
		if !strings.Contains(got, "img.shields.io") {
			t.Error("expected markdown output to contain an img.shields.io badge")
		}
		if !strings.Contains(got, "Auth") {
			t.Errorf("expected epic title Auth in output, got %q", got)
		}
		if !strings.Contains(got, "Login") {
			t.Errorf("expected item title Login in output, got %q", got)
		}
	})

	t.Run("tty renders plain-text table with Root", func(t *testing.T) {
		got := roadmapOutput(data, true, 80, true, "", false)
		if !strings.HasPrefix(got, "Roadmap") {
			t.Errorf("got prefix %q, want %q", got[:min(len(got), 20)], "Roadmap")
		}
		if !strings.Contains(got, "▸ Epic") {
			t.Errorf("expected epic glyph line (▸ Epic), got %q", got)
		}
		if strings.Contains(got, "img.shields.io") {
			t.Error("TTY output must not contain shields.io badges")
		}
		if strings.Contains(got, "](") {
			t.Error("TTY output must not contain markdown link syntax")
		}
		if strings.Contains(got, "■ Milestone") {
			t.Error("root-scoped output must not contain milestone framing")
		}
		if !strings.Contains(got, "Auth") {
			t.Errorf("expected epic title Auth in output, got %q", got)
		}
		if !strings.Contains(got, "Login") {
			t.Errorf("expected item title Login in output, got %q", got)
		}
	})
}

// SC-502 / EARS-5: the non-TTY path of roadmapOutput must be a pure pass
// -through to renderRoadmapMarkdown -- character-for-character, with the
// exact same arguments. This is the regression guard for the Q07/D02
// byte-identity constraint at the function level.
// Table-driven over both `links` values (reviewer B01): a prior version of
// this test only ever passed links=true, so a mutation hardcoding the
// markdown branch's `links` argument to `true` left the whole suite green --
// the `false` direction was never exercised. `--no-links` is a real user
// flag (roadmapNoLinks, roadmap.go); if roadmapOutput ever stopped
// forwarding it, the piped/redirected markdown output would silently ignore
// it. Both directions must be observed for the wiring itself to be pinned.
func TestRoadmapMarkdownByteIdentical(t *testing.T) {
	data := roadmapOutputFixture()

	tests := []struct {
		name  string
		links bool
	}{
		{name: "links enabled", links: true},
		{name: "links disabled (--no-links)", links: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roadmapOutput(data, false, 0, tt.links, "some/prefix", false)
			want := renderRoadmapMarkdown(data, tt.links, "some/prefix", false)
			if got != want {
				t.Errorf("roadmapOutput(non-tty, links=%v, false) diverged from renderRoadmapMarkdown:\ngot:  %q\nwant: %q", tt.links, got, want)
			}
			// links=false must actually change the rendered output relative
			// to links=true for this fixture (has a milestone with a bean
			// ref) -- otherwise the two subtests could both pass vacuously
			// against a `links` argument that was never wired through at all.
			withLinks := renderRoadmapMarkdown(data, true, "some/prefix", false)
			withoutLinks := renderRoadmapMarkdown(data, false, "some/prefix", false)
			if withLinks == withoutLinks {
				t.Fatal("fixture invariant broken: renderRoadmapMarkdown output identical for links=true/false -- test cannot distinguish the two branches")
			}
		})
	}
}

// Mutation guard for the "fallback bei unbestimmbarer Breite" line (D08):
// roadmapOutput must run cols through roadmapClampWidth, not use it raw. A
// cols=0 caller (no terminal detected) must land on the 80-column floor, not
// a 0-width render.
func TestRoadmapOutputZeroColsFallsBackTo80(t *testing.T) {
	data := roadmapOutputFixture()
	got := roadmapOutput(data, true, 0, true, "", false)

	lines := strings.SplitN(got, "\n", 3)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), got)
	}
	sepWidth := utf8.RuneCountInString(lines[1])
	if sepWidth != 80 {
		t.Errorf("separator width = %d, want 80 (roadmapClampWidth(0) floor, D08)", sepWidth)
	}
}

func TestBuildScopedRoadmapMilestoneRoot(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	now := time.Now()
	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now},
		{ID: "e1", Type: "epic", Title: "Auth", Status: "todo", Parent: "m1"},
		{ID: "t1", Type: "task", Title: "Login", Status: "todo", Parent: "e1"},
		{ID: "m2", Type: "milestone", Title: "v2.0", Status: "todo", CreatedAt: &now},
		{ID: "t2", Type: "task", Title: "Other milestone task", Status: "todo", Parent: "m2"},
		{ID: "e3", Type: "epic", Title: "Unscheduled", Status: "todo"},
	}

	root := beans[0] // m1
	result := buildScopedRoadmap(beans, false, root)

	if len(result.Milestones) != 1 {
		t.Fatalf("got %d milestones, want 1", len(result.Milestones))
	}
	if result.Milestones[0].Milestone.ID != "m1" {
		t.Errorf("got milestone %s, want m1", result.Milestones[0].Milestone.ID)
	}
	if result.Unscheduled != nil {
		t.Errorf("expected Unscheduled to be nil when scoped to a milestone, got %+v", result.Unscheduled)
	}
	if result.Root != nil {
		t.Errorf("expected Root to be nil for milestone scope, got %+v", result.Root)
	}
}

func TestBuildScopedRoadmapMilestoneRootWithNoVisibleChildren(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	now := time.Now()
	root := &bean.Bean{ID: "m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now}
	beans := []*bean.Bean{root}

	result := buildScopedRoadmap(beans, false, root)

	if len(result.Milestones) != 1 {
		t.Fatalf("got %d milestones, want 1 (empty container still rendered)", len(result.Milestones))
	}
	if result.Milestones[0].Milestone.ID != "m1" {
		t.Errorf("got milestone %s, want m1", result.Milestones[0].Milestone.ID)
	}
	if len(result.Milestones[0].Epics) != 0 || len(result.Milestones[0].Features) != 0 || len(result.Milestones[0].Other) != 0 {
		t.Errorf("expected empty milestone group, got %+v", result.Milestones[0])
	}
}

func TestBuildScopedRoadmapEpicRoot(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	root := &bean.Bean{ID: "e1", Type: "epic", Title: "Auth", Status: "todo"}
	beans := []*bean.Bean{
		root,
		{ID: "t1", Type: "task", Title: "Login", Status: "todo", Parent: "e1"},
		{ID: "f1", Type: "feature", Title: "SSO", Status: "todo", Parent: "e1"},
		{ID: "t2", Type: "task", Title: "OIDC", Status: "todo", Parent: "f1"},
		{ID: "m1", Type: "milestone", Title: "Other", Status: "todo"},
	}

	result := buildScopedRoadmap(beans, false, root)

	if result.Root == nil || result.Root.Epic == nil {
		t.Fatalf("expected Root.Epic to be set, got %+v", result.Root)
	}
	if result.Root.Epic.Epic.ID != "e1" {
		t.Errorf("got epic %s, want e1", result.Root.Epic.Epic.ID)
	}
	if len(result.Root.Epic.Items) != 1 || result.Root.Epic.Items[0].ID != "t1" {
		t.Errorf("Root.Epic.Items = %v, want [t1]", result.Root.Epic.Items)
	}
	if len(result.Root.Epic.Features) != 1 || result.Root.Epic.Features[0].Feature.ID != "f1" {
		t.Fatalf("Root.Epic.Features = %+v, want feature f1", result.Root.Epic.Features)
	}
	if len(result.Milestones) != 0 {
		t.Errorf("expected no Milestones for epic scope, got %d", len(result.Milestones))
	}
}

func TestBuildScopedRoadmapFeatureRoot(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	root := &bean.Bean{ID: "f1", Type: "feature", Title: "SSO", Status: "todo"}
	beans := []*bean.Bean{
		root,
		{ID: "t1", Type: "task", Title: "OIDC", Status: "todo", Parent: "f1"},
	}

	result := buildScopedRoadmap(beans, false, root)

	if result.Root == nil || result.Root.Feature == nil {
		t.Fatalf("expected Root.Feature to be set, got %+v", result.Root)
	}
	if result.Root.Feature.Feature.ID != "f1" {
		t.Errorf("got feature %s, want f1", result.Root.Feature.Feature.ID)
	}
	if len(result.Root.Feature.Items) != 1 || result.Root.Feature.Items[0].ID != "t1" {
		t.Errorf("Root.Feature.Items = %v, want [t1]", result.Root.Feature.Items)
	}
}

func TestBuildScopedRoadmapEpicRootWithNoVisibleChildren(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	root := &bean.Bean{ID: "e1", Type: "epic", Title: "Empty epic", Status: "todo"}
	beans := []*bean.Bean{root}

	result := buildScopedRoadmap(beans, false, root)

	if result.Root == nil || result.Root.Epic == nil {
		t.Fatalf("expected Root.Epic to still be set for an empty epic, got %+v", result.Root)
	}
	if result.Root.Epic.Epic.ID != "e1" {
		t.Errorf("got epic %s, want e1", result.Root.Epic.Epic.ID)
	}
	if len(result.Root.Epic.Items) != 0 || len(result.Root.Epic.Features) != 0 {
		t.Errorf("expected an empty epic group, got %+v", result.Root.Epic)
	}
}

func TestBuildScopedRoadmapFeatureRootWithNoVisibleChildren(t *testing.T) {
	oldCfg := cfg
	defer func() { cfg = oldCfg }()
	cfg = config.Default()

	root := &bean.Bean{ID: "f1", Type: "feature", Title: "Empty feature", Status: "todo"}
	beans := []*bean.Bean{root}

	result := buildScopedRoadmap(beans, false, root)

	if result.Root == nil || result.Root.Feature == nil {
		t.Fatalf("expected Root.Feature to still be set for an empty feature, got %+v", result.Root)
	}
	if result.Root.Feature.Feature.ID != "f1" {
		t.Errorf("got feature %s, want f1", result.Root.Feature.Feature.ID)
	}
	if len(result.Root.Feature.Items) != 0 {
		t.Errorf("expected an empty feature group, got %+v", result.Root.Feature)
	}
}

func TestValidateRoadmapRootType(t *testing.T) {
	tests := []struct {
		beanType string
		wantErr  bool
	}{
		{"milestone", false},
		{"epic", false},
		{"feature", false},
		{"task", true},
		{"bug", true},
	}
	for _, tt := range tests {
		t.Run(tt.beanType, func(t *testing.T) {
			b := &bean.Bean{ID: "x1", Type: tt.beanType}
			err := validateRoadmapRootType(b)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRoadmapRootType(%s) error = %v, wantErr %v", tt.beanType, err, tt.wantErr)
			}
		})
	}
}

// TestValidateRoadmapRootTypeWithNoContainerTypes covers a profile like
// "todo" that defines no container types at all: the old message rendered
// as "roadmap root must be one of , got task (x-1)" -- an empty list before
// "got" instead of a sensible explanation. The error must instead say the
// project has no container types, without touching the normal-case message
// TestRoadmapCmdRejectsNonContainerRootType depends on.
func TestValidateRoadmapRootTypeWithNoContainerTypes(t *testing.T) {
	leafRank := config.LeafRank
	prev := cfg
	cfg = &config.Config{TypesExclusive: true, Types: []config.TypeOverride{
		{Name: "task", Rank: &leafRank},
	}}
	defer func() { cfg = prev }()

	err := validateRoadmapRootType(&bean.Bean{ID: "x-1", Type: "task"})
	if err == nil {
		t.Fatal("expected an error when the project defines no container types")
	}
	if !strings.Contains(err.Error(), "no container types") {
		t.Errorf("expected error to say the project defines no container types, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "must be one of ,") {
		t.Errorf("error still renders the degenerate empty list, got %q", err.Error())
	}
}

func TestRenderRoadmapMarkdownRootEpic(t *testing.T) {
	e := &bean.Bean{ID: "beans-e1", Type: "epic", Title: "Auth", Status: "todo", Path: "e1--auth.md"}
	item1 := &bean.Bean{ID: "beans-t1", Type: "task", Title: "Login", Status: "todo", Path: "t1--login.md"}
	data := &roadmapData{
		Root: &rootGroup{
			Epic: &epicGroup{Epic: e, Items: []*bean.Bean{item1}},
		},
	}

	got := renderRoadmapMarkdown(data, true, "", false)

	if !strings.HasPrefix(got, "# Roadmap") {
		t.Errorf("got prefix %q, want %q", got[:min(len(got), 20)], "# Roadmap")
	}
	if !strings.Contains(got, "### Epic: Auth") {
		t.Errorf("expected epic heading, got %q", got)
	}
	if !strings.Contains(got, "Login") {
		t.Errorf("expected item Login to be rendered, got %q", got)
	}
	if strings.Contains(got, "## Milestone") {
		t.Errorf("root-scoped output must not contain a Milestone heading, got %q", got)
	}
	if strings.Contains(got, "No Milestone") {
		t.Errorf("root-scoped output must not contain the Unscheduled heading, got %q", got)
	}
}

func TestRenderRoadmapMarkdownRootFeature(t *testing.T) {
	f := &bean.Bean{ID: "beans-f1", Type: "feature", Title: "SSO", Status: "todo", Path: "f1--sso.md"}
	item1 := &bean.Bean{ID: "beans-t1", Type: "task", Title: "OIDC", Status: "todo", Path: "t1--oidc.md"}
	data := &roadmapData{
		Root: &rootGroup{
			Feature: &featureGroup{Feature: f, Items: []*bean.Bean{item1}},
		},
	}

	got := renderRoadmapMarkdown(data, true, "", false)

	if !strings.Contains(got, "#### Feature: SSO") {
		t.Errorf("expected feature heading, got %q", got)
	}
	if !strings.Contains(got, "OIDC") {
		t.Errorf("expected item OIDC to be rendered, got %q", got)
	}
}

func TestRoadmapPlacesAConfiguredRank2TypeInTheEpicSlot(t *testing.T) {
	rank := 2
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{{Name: "package", Rank: &rank}}}
	defer func() { cfg = prev }()

	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Status: "todo", Title: "Release"},
		{ID: "p1", Type: "package", Status: "todo", Title: "A package", Parent: "m1"},
		{ID: "t1", Type: "task", Status: "todo", Title: "A task", Parent: "p1"},
	}

	data := buildRoadmap(beans, false, nil, nil)

	if len(data.Milestones) != 1 {
		t.Fatalf("got %d milestone groups, want 1", len(data.Milestones))
	}
	group := data.Milestones[0]
	if len(group.Epics) != 1 {
		t.Fatalf("got %d rank-2 groups, want 1 — a configured rank-2 type belongs in the epic slot", len(group.Epics))
	}
	if group.Epics[0].Epic.ID != "p1" {
		t.Errorf("rank-2 slot holds %q, want \"p1\"", group.Epics[0].Epic.ID)
	}
	if len(group.Epics[0].Items) != 1 {
		t.Errorf("got %d leaves under the package, want 1", len(group.Epics[0].Items))
	}
}

func TestRoadmapAcceptsAConfiguredRank1TypeAsScopeRoot(t *testing.T) {
	rank := 1
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{{Name: "release", Rank: &rank}}}
	defer func() { cfg = prev }()

	root := &bean.Bean{ID: "r1", Type: "release", Status: "todo", Title: "v1"}
	if err := validateRoadmapRootType(root); err != nil {
		t.Errorf("validateRoadmapRootType(release) = %v, want nil", err)
	}
}

func TestRoadmapRejectsALeafAsScopeRoot(t *testing.T) {
	prev := cfg
	cfg = &config.Config{}
	defer func() { cfg = prev }()

	if err := validateRoadmapRootType(&bean.Bean{ID: "t1", Type: "task"}); err == nil {
		t.Error("a leaf type must not be accepted as a roadmap root")
	}
}

func TestHiddenContainerAndItsSubtreeStayOutOfTheRoadmap(t *testing.T) {
	rank := 1
	visible := false
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "bucket", Rank: &rank, Roadmap: &visible},
	}}
	defer func() { cfg = prev }()

	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Status: "todo", Title: "Release"},
		{ID: "t1", Type: "task", Status: "todo", Title: "Planned", Parent: "m1"},
		{ID: "b1", Type: "bucket", Status: "todo", Title: "Parking lot"},
		{ID: "t2", Type: "task", Status: "todo", Title: "Someday", Parent: "b1"},
	}

	data := buildRoadmap(beans, false, nil, nil)

	for _, g := range data.Milestones {
		if g.Milestone.ID == "b1" {
			t.Fatal("a hidden container must not render as its own section")
		}
	}
	if data.Unscheduled != nil {
		for _, b := range data.Unscheduled.Other {
			if b.ID == "t2" {
				t.Fatal("a leaf under a hidden container must not resurface as unscheduled")
			}
		}
	}
	if len(data.Milestones) != 1 || data.Milestones[0].Milestone.ID != "m1" {
		t.Errorf("the visible milestone must still render")
	}
}

// TestHiddenContainerHidesEveryRankBeneathIt covers the full container depth
// the rank scheme allows: a hidden rank-1 bucket with a rank-2 child that
// itself has a rank-3 child with a leaf, plus a rank-3 child parented
// directly under the hidden rank-1 (skipping rank 2). A visible-typed
// container nested under a hidden one must vanish too -- markSubtree walks
// every descendant regardless of its own type -- and none of it may resurface
// in any of the unscheduled buckets.
func TestHiddenContainerHidesEveryRankBeneathIt(t *testing.T) {
	rank := 1
	visible := false
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "bucket", Rank: &rank, Roadmap: &visible},
	}}
	defer func() { cfg = prev }()

	beans := []*bean.Bean{
		{ID: "b1", Type: "bucket", Status: "todo", Title: "Parking lot"},
		{ID: "e1", Type: "epic", Status: "todo", Title: "Nested epic", Parent: "b1"},
		{ID: "f1", Type: "feature", Status: "todo", Title: "Nested feature", Parent: "e1"},
		{ID: "t1", Type: "task", Status: "todo", Title: "Deep leaf", Parent: "f1"},
		{ID: "f2", Type: "feature", Status: "todo", Title: "Direct feature", Parent: "b1"},
		{ID: "t2", Type: "task", Status: "todo", Title: "Leaf under direct feature", Parent: "f2"},
	}

	data := buildRoadmap(beans, false, nil, nil)

	if data.Unscheduled != nil {
		for _, eg := range data.Unscheduled.Epics {
			if eg.Epic.ID == "e1" {
				t.Fatal("a rank-2 container nested under a hidden rank-1 container must not resurface as unscheduled")
			}
		}
		for _, fg := range data.Unscheduled.Features {
			if fg.Feature.ID == "f1" || fg.Feature.ID == "f2" {
				t.Fatalf("a rank-3 container under a hidden rank-1 container must not resurface as unscheduled, got %q", fg.Feature.ID)
			}
		}
		for _, b := range data.Unscheduled.Other {
			if b.ID == "t1" || b.ID == "t2" {
				t.Fatalf("a leaf at any depth under a hidden rank-1 container must not resurface as unscheduled, got %q", b.ID)
			}
		}
	}
	if len(data.Milestones) != 0 {
		t.Errorf("got %d milestone groups, want 0 — the hidden bucket is the only rank-1 bean", len(data.Milestones))
	}
}

// TestHiddenContainerNestedUnderVisibleParentVanishesEntirely covers Ruling 1
// of the fix round (D15): hiding removes, it does not reclassify. A
// hidden-type rank-2 container underneath a *visible* rank-1 milestone must
// vanish together with its whole subtree -- it must not render as one of the
// milestone's Epics, and its leaf must not fold up into the milestone's flat
// "Other" list either (that fold-up is exactly the "unassigned items"
// failure mode D15 names).
func TestHiddenContainerNestedUnderVisibleParentVanishesEntirely(t *testing.T) {
	rank := 2
	visible := false
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "bucket", Rank: &rank, Roadmap: &visible},
	}}
	defer func() { cfg = prev }()

	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Status: "todo", Title: "Release"},
		{ID: "t0", Type: "task", Status: "todo", Title: "Planned", Parent: "m1"},
		{ID: "b1", Type: "bucket", Status: "todo", Title: "Parking lot", Parent: "m1"},
		{ID: "t1", Type: "task", Status: "todo", Title: "Someday", Parent: "b1"},
	}

	data := buildRoadmap(beans, false, nil, nil)

	if len(data.Milestones) != 1 || data.Milestones[0].Milestone.ID != "m1" {
		t.Fatalf("the visible milestone must still render, got %+v", data.Milestones)
	}
	group := data.Milestones[0]
	for _, eg := range group.Epics {
		if eg.Epic.ID == "b1" {
			t.Fatal("a hidden rank-2 container under a visible milestone must not render as one of its Epics")
		}
	}
	for _, o := range group.Other {
		if o.ID == "t1" || o.ID == "b1" {
			t.Fatalf("a hidden container's leaf must not fold up into the parent's flat Other list, got %q", o.ID)
		}
	}
	if data.Unscheduled != nil {
		for _, o := range data.Unscheduled.Other {
			if o.ID == "t1" || o.ID == "b1" {
				t.Fatalf("a hidden container's leaf must not resurface as unscheduled either, got %q", o.ID)
			}
		}
	}
}

// TestScopedRoadmapHidingRespectsScopeBoundary pins Ruling 2 (fix round 2)
// plus its round-3 correction: the scoped-by-ID bypass exempts exactly the
// named root, nothing more and nothing less.
//
//   - Naming a hidden container directly renders its full subtree (the
//     bypass): true for a rank-1 root (restores the coverage
//     TestScopedRoadmapBypassesHidingForTheNamedRoot carried before it was
//     narrowed to rank 2 only) and for a rank-2 root.
//   - Naming a visible container that sits BELOW a hidden ancestor also
//     renders its full subtree: a direct-by-ID lookup is not the aggregate
//     view the visibility flag governs, and an ancestor's opt-out must not
//     reach down past the scoped root. This is the fix-round-4 bug: seeding
//     hidden from a bean whose ID merely differs from root reaches ancestors
//     of root too, and marks everything root and its descendants ancestrally
//     shared.
//   - Naming a visible container that CONTAINS a hidden container still
//     suppresses that inner hidden container and its subtree -- the bypass
//     is not total, D15 still applies to anything nested inside the scope.
//
// One fixture carries all four cases: v1 (hidden rank-1 vault) -> e2
// (visible epic) -> t2 (leaf), and m1 (visible milestone) -> b1 (hidden
// rank-2 bucket) -> t1 (leaf).
func TestScopedRoadmapHidingRespectsScopeBoundary(t *testing.T) {
	rank1, rank2 := 1, 2
	hidden := false
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "bucket", Rank: &rank2, Roadmap: &hidden},
		{Name: "vault", Rank: &rank1, Roadmap: &hidden},
	}}
	defer func() { cfg = prev }()

	m1 := &bean.Bean{ID: "m1", Type: "milestone", Status: "todo", Title: "Release"}
	b1 := &bean.Bean{ID: "b1", Type: "bucket", Status: "todo", Title: "Parking lot", Parent: "m1"}
	t1 := &bean.Bean{ID: "t1", Type: "task", Status: "todo", Title: "Someday", Parent: "b1"}
	v1 := &bean.Bean{ID: "v1", Type: "vault", Status: "todo", Title: "Attic"}
	e2 := &bean.Bean{ID: "e2", Type: "epic", Status: "todo", Title: "Someday feature", Parent: "v1"}
	t2 := &bean.Bean{ID: "t2", Type: "task", Status: "todo", Title: "Someday task", Parent: "e2"}
	beans := []*bean.Bean{m1, b1, t1, v1, e2, t2}

	t.Run("scoped to a visible container that contains a hidden one, the hidden subtree is suppressed", func(t *testing.T) {
		data := buildScopedRoadmap(beans, false, m1)
		if len(data.Milestones) != 1 || data.Milestones[0].Milestone.ID != "m1" {
			t.Fatalf("expected the named root m1 to render, got %+v", data.Milestones)
		}
		group := data.Milestones[0]
		for _, eg := range group.Epics {
			if eg.Epic.ID == "b1" {
				t.Fatal("a hidden rank-2 container below the scoped root must not render as one of its Epics")
			}
		}
		for _, o := range group.Other {
			if o.ID == "b1" || o.ID == "t1" {
				t.Fatalf("a hidden container's subtree below the scoped root must not fold into Other, got %q", o.ID)
			}
		}
	})

	t.Run("scoped directly to a hidden rank-2 container, it renders in full", func(t *testing.T) {
		data := buildScopedRoadmap(beans, false, b1)
		if data.Root == nil || data.Root.Epic == nil || data.Root.Epic.Epic.ID != "b1" {
			t.Fatalf("a scoped hidden root must still render as its own container, got %+v", data.Root)
		}
		found := false
		for _, o := range data.Root.Epic.Items {
			if o.ID == "t1" {
				found = true
			}
		}
		if !found {
			t.Error("a scoped hidden root must render its own full content -- the bypass covers the named root")
		}
	})

	t.Run("scoped directly to a hidden rank-1 container, it renders in full", func(t *testing.T) {
		data := buildScopedRoadmap(beans, false, v1)
		if len(data.Milestones) != 1 || data.Milestones[0].Milestone.ID != "v1" {
			t.Fatalf("a scoped hidden rank-1 root must still render as its own container, got %+v", data.Milestones)
		}
		group := data.Milestones[0]
		if len(group.Epics) != 1 || group.Epics[0].Epic.ID != "e2" {
			t.Fatalf("a scoped hidden rank-1 root must render its own visible child, got %+v", group.Epics)
		}
		found := false
		for _, item := range group.Epics[0].Items {
			if item.ID == "t2" {
				found = true
			}
		}
		if !found {
			t.Error("a scoped hidden rank-1 root must render its whole subtree -- the bypass covers the named root")
		}
	})

	t.Run("scoped to a visible container below a hidden ancestor, it renders in full", func(t *testing.T) {
		data := buildScopedRoadmap(beans, false, e2)
		if data.Root == nil || data.Root.Epic == nil || data.Root.Epic.Epic.ID != "e2" {
			t.Fatalf("expected the named root e2 to render, got %+v", data.Root)
		}
		found := false
		for _, item := range data.Root.Epic.Items {
			if item.ID == "t2" {
				found = true
			}
		}
		if !found {
			t.Error("a visible root below a hidden ancestor must render its full subtree -- the ancestor's opt-out must not reach past the scoped root")
		}
	})
}

// TestHiddenRank3ContainerVanishesWithItsLeaf closes a coverage gap the
// reviewer flagged: rank 3, directly opting out (not via an ancestor), had
// no test of its own alongside the existing rank-1/rank-2 cases.
func TestHiddenRank3ContainerVanishesWithItsLeaf(t *testing.T) {
	rank := 3
	visible := false
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "drawer", Rank: &rank, Roadmap: &visible},
	}}
	defer func() { cfg = prev }()

	beans := []*bean.Bean{
		{ID: "m1", Type: "milestone", Status: "todo", Title: "Release"},
		{ID: "e1", Type: "epic", Status: "todo", Title: "Auth", Parent: "m1"},
		{ID: "t2", Type: "task", Status: "todo", Title: "Planned", Parent: "e1"},
		{ID: "d1", Type: "drawer", Status: "todo", Title: "Hidden drawer", Parent: "e1"},
		{ID: "t1", Type: "task", Status: "todo", Title: "Someday", Parent: "d1"},
	}

	data := buildRoadmap(beans, false, nil, nil)

	if len(data.Milestones) != 1 {
		t.Fatalf("got %d milestone groups, want 1", len(data.Milestones))
	}
	group := data.Milestones[0]
	if len(group.Epics) != 1 || group.Epics[0].Epic.ID != "e1" {
		t.Fatalf("expected epic e1 to still render (it has a visible sibling task), got %+v", group.Epics)
	}
	eg := group.Epics[0]
	for _, fg := range eg.Features {
		if fg.Feature.ID == "d1" {
			t.Fatal("a hidden rank-3 container must not render as one of its epic's Features")
		}
	}
	foundT1, foundT2 := false, false
	for _, item := range eg.Items {
		if item.ID == "t1" {
			foundT1 = true
		}
		if item.ID == "t2" {
			foundT2 = true
		}
	}
	if foundT1 {
		t.Error("a leaf under a hidden rank-3 container must not fold into the epic's Items")
	}
	if !foundT2 {
		t.Error("the epic's own visible leaf must still render")
	}
}

// TestScopedRoadmapRank3RootBypassesItsOwnHiddenType mirrors the rank-1/
// rank-2 scoped-bypass coverage for rank 3, per the reviewer's note.
func TestScopedRoadmapRank3RootBypassesItsOwnHiddenType(t *testing.T) {
	rank := 3
	visible := false
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "drawer", Rank: &rank, Roadmap: &visible},
	}}
	defer func() { cfg = prev }()

	root := &bean.Bean{ID: "d1", Type: "drawer", Status: "todo", Title: "Hidden drawer"}
	beans := []*bean.Bean{
		root,
		{ID: "t1", Type: "task", Status: "todo", Title: "Someday", Parent: "d1"},
	}

	data := buildScopedRoadmap(beans, false, root)

	if data.Root == nil || data.Root.Feature == nil || data.Root.Feature.Feature.ID != "d1" {
		t.Fatalf("a scoped hidden rank-3 root must still render as its own container, got %+v", data.Root)
	}
	found := false
	for _, item := range data.Root.Feature.Items {
		if item.ID == "t1" {
			found = true
		}
	}
	if !found {
		t.Error("a scoped hidden rank-3 root must render its own full content")
	}
}

func TestTypeBadgeUsesTheConfiguredColour(t *testing.T) {
	rank := 2
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "chore", Rank: &rank, Color: "peach"},
	}}
	defer func() { cfg = prev }()

	got := typeBadge(&bean.Bean{ID: "c1", Type: "chore"})

	if strings.Contains(got, "gray") {
		t.Errorf("badge fell back to gray: %s", got)
	}
	if !strings.Contains(got, "chore-") {
		t.Errorf("badge does not name the type: %s", got)
	}
}

func TestTypeBadgeStaysEmptyForATypelessBean(t *testing.T) {
	if got := typeBadge(&bean.Bean{ID: "x1"}); got != "" {
		t.Errorf("typeBadge() = %q, want empty", got)
	}
}

// badgeURLPattern pins the exact shape shields.io needs: a bare 6-digit hex
// (no leading '#') or the literal fallback "gray" after the type-dash. A
// stray '#' would make the URL fail to match and thus fail the test, where
// the badge itself would just silently not render.
var badgeURLPattern = regexp.MustCompile(`^!\[([a-z]+)\]\(https://img\.shields\.io/badge/([a-z]+)-([0-9a-f]{6}|gray)\?style=flat-square\)$`)

func TestTypeBadgeURLOmitsTheLeadingHash(t *testing.T) {
	rank := 2
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "chore", Rank: &rank, Color: "peach"},
	}}
	defer func() { cfg = prev }()

	got := typeBadge(&bean.Bean{ID: "c1", Type: "chore"})

	m := badgeURLPattern.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("typeBadge() = %q, does not match expected badge URL shape %s", got, badgeURLPattern)
	}
	if m[3] == "gray" {
		t.Errorf("typeBadge() = %q, resolved colour fell back to gray", got)
	}
	if strings.Contains(got, "#") {
		t.Errorf("typeBadge() = %q, badge URL carries a stray '#'", got)
	}
}

func TestTypeBadgeFallsBackToGrayForATypeWithNoColour(t *testing.T) {
	rank := 4
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "task", Rank: &rank},
	}}
	defer func() { cfg = prev }()

	got := typeBadge(&bean.Bean{ID: "t1", Type: "task"})

	want := "![task](https://img.shields.io/badge/task-gray?style=flat-square)"
	if got != want {
		t.Errorf("typeBadge() = %q, want %q", got, want)
	}
}

func TestTypeBadgeFallsBackToGrayForAnUnknownType(t *testing.T) {
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{}}
	defer func() { cfg = prev }()

	got := typeBadge(&bean.Bean{ID: "u1", Type: "mystery"})

	want := "![mystery](https://img.shields.io/badge/mystery-gray?style=flat-square)"
	if got != want {
		t.Errorf("typeBadge() = %q, want %q", got, want)
	}
}

func TestTypeBadgeAcceptsARawHexColour(t *testing.T) {
	rank := 2
	prev := cfg
	cfg = &config.Config{Types: []config.TypeOverride{
		{Name: "chore", Rank: &rank, Color: "#123abc"},
	}}
	defer func() { cfg = prev }()

	got := typeBadge(&bean.Bean{ID: "c1", Type: "chore"})

	want := "![chore](https://img.shields.io/badge/chore-123abc?style=flat-square)"
	if got != want {
		t.Errorf("typeBadge() = %q, want %q", got, want)
	}
}
