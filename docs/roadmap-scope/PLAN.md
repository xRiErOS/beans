---
type: Plan
title: "PLAN — roadmap-scope"
description: "Implementation plan for scoping beans roadmap to a single milestone, epic, or feature root"
tags:
  - tpic
  - roadmap-scope
timestamp: 2026-08-11T11:58:39Z
---

# roadmap-scope Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an optional positional bean ID argument to `beans roadmap` (`beans roadmap <id>`) that scopes the output to a single milestone, epic, or feature subtree instead of the whole repository.

**Architecture:** A new pure data-layer function `buildScopedRoadmap` builds a `roadmapData` restricted to one root bean, reusing the existing `buildRoadmap`/`buildEpicGroup`/`buildFeatureGroup` machinery unchanged. A new `roadmapData.Root` field carries the epic/feature case; the milestone case reuses the existing `Milestones` slice with one entry. Both renderers (`roadmap.tmpl` for Markdown, `roadmap_pretty.go` for TTY) get a small `Root`-aware branch. The CLI layer resolves and type-checks the positional ID via the existing `core.Get`, and rejects it combined with `--status`/`--no-status`.

**Tech Stack:** Go, Cobra (CLI), `text/template` (Markdown rendering), Go's standard `testing` package (table-driven tests, following existing conventions in `internal/commands/roadmap_test.go` and `internal/commands/order_test.go`).

## Global Constraints

- Package layering: `internal/commands` may depend on `pkg/bean`, `pkg/beancore`, `internal/graph` (via `beangraph`) — no new dependency direction is introduced by this plan (`.claude/rules/backend.md`).
- Bean sorting: any newly-sorted list must go through the existing `bean.SortRoadmapContainers`/`bean.SortRoadmapLeaves` helpers already used by `buildRoadmap` — this plan does not introduce new sorting, it only restricts which beans reach the existing sorters.
- No `TBD`/placeholder code; every step below is a complete, runnable change.
- Full regression check before the final commit: `mise test` (root `Justfile`/`mise` wraps `go test ./...`, per `.claude/rules/tools.md`).

---

### Task 1: Data layer — `rootGroup`, `buildScopedRoadmap`, `validateRoadmapRootType`

**Files:**
- Modify: `internal/commands/roadmap.go:34-37` (add `Root` field to `roadmapData`)
- Modify: `internal/commands/roadmap.go:135-142` (extract `childrenIndex` helper)
- Modify: `internal/commands/roadmap.go` (add `rootGroup` type, `buildScopedRoadmap`, `validateRoadmapRootType`, `childrenIndex` — placed after `buildRoadmap`, i.e. after current line 284)
- Test: `internal/commands/roadmap_test.go`

**Interfaces:**
- Consumes: `buildRoadmap` (existing, unchanged signature `buildRoadmap(allBeans []*bean.Bean, includeDone bool, statusFilter, noStatusFilter []string) *roadmapData`), `buildEpicGroup(epic *bean.Bean, children map[string][]*bean.Bean, includeDone bool) epicGroup`, `buildFeatureGroup(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool) featureGroup` — all pre-existing in `roadmap.go`.
- Produces: `type rootGroup struct { Epic *epicGroup; Feature *featureGroup }` (exactly one populated); `roadmapData.Root *rootGroup` (new first field, `json:"root,omitempty"`); `func buildScopedRoadmap(allBeans []*bean.Bean, includeDone bool, root *bean.Bean) *roadmapData`; `func validateRoadmapRootType(b *bean.Bean) error`; `func childrenIndex(allBeans []*bean.Bean) map[string][]*bean.Bean`. Task 4 calls `buildScopedRoadmap` and `validateRoadmapRootType` directly.

- [ ] **Step 1: Write the failing tests**

Add to `internal/commands/roadmap_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run `go test ./internal/commands/... -run 'TestBuildScopedRoadmap|TestValidateRoadmapRootType' -v` — expected to FAIL to compile, since `buildScopedRoadmap`, `validateRoadmapRootType`, and `roadmapData.Root` are undefined.

- [ ] **Step 3: Add the `Root` field to `roadmapData`**

In `internal/commands/roadmap.go`, replace:

```go
// roadmapData holds the structured roadmap for JSON output.
type roadmapData struct {
	Milestones  []milestoneGroup  `json:"milestones"`
	Unscheduled *unscheduledGroup `json:"unscheduled,omitempty"`
}
```

with:

```go
// roadmapData holds the structured roadmap for JSON output.
type roadmapData struct {
	// Root is set instead of Milestones/Unscheduled when the roadmap is
	// scoped to a single epic or feature (buildScopedRoadmap). A
	// milestone-scoped roadmap reuses Milestones with a single entry
	// instead, since that requires no new rendering path.
	Root        *rootGroup        `json:"root,omitempty"`
	Milestones  []milestoneGroup  `json:"milestones"`
	Unscheduled *unscheduledGroup `json:"unscheduled,omitempty"`
}

// rootGroup holds the scoped roadmap when rooted at an epic or feature.
// Exactly one of Epic/Feature is set.
type rootGroup struct {
	Epic    *epicGroup    `json:"epic,omitempty"`
	Feature *featureGroup `json:"feature,omitempty"`
}
```

- [ ] **Step 4: Extract the `childrenIndex` helper**

In `internal/commands/roadmap.go`, inside `buildRoadmap`, replace:

```go
	// Build children index: parent ID -> children
	// This maps each bean ID to the beans that have it as a parent
	children := make(map[string][]*bean.Bean)
	for _, b := range allBeans {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b)
		}
	}
```

with:

```go
	children := childrenIndex(allBeans)
```

Then add the extracted helper directly below `buildRoadmap` (after its closing `}`, before `buildMilestoneGroup`):

```go
// childrenIndex maps each bean ID to the beans that have it as a parent.
func childrenIndex(allBeans []*bean.Bean) map[string][]*bean.Bean {
	children := make(map[string][]*bean.Bean)
	for _, b := range allBeans {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b)
		}
	}
	return children
}
```

- [ ] **Step 5: Add `buildScopedRoadmap` and `validateRoadmapRootType`**

Add directly below the `childrenIndex` helper from Step 4:

```go
// buildScopedRoadmap builds a roadmapData scoped to a single milestone,
// epic, or feature root. Callers must have already validated root's type
// via validateRoadmapRootType; any other type panics, since that would be a
// caller bug, not user input.
func buildScopedRoadmap(allBeans []*bean.Bean, includeDone bool, root *bean.Bean) *roadmapData {
	switch root.Type {
	case "milestone":
		data := buildRoadmap(allBeans, includeDone, nil, nil)
		for _, mg := range data.Milestones {
			if mg.Milestone.ID == root.ID {
				return &roadmapData{Milestones: []milestoneGroup{mg}}
			}
		}
		// buildRoadmap drops milestones with zero visible children -- the
		// root was still found and matched by type/ID, so render it as an
		// empty container rather than silently returning nothing.
		return &roadmapData{Milestones: []milestoneGroup{{Milestone: root}}}
	case "epic":
		eg := buildEpicGroup(root, childrenIndex(allBeans), includeDone)
		return &roadmapData{Root: &rootGroup{Epic: &eg}}
	case "feature":
		fg := buildFeatureGroup(root, childrenIndex(allBeans), includeDone)
		return &roadmapData{Root: &rootGroup{Feature: &fg}}
	default:
		panic("buildScopedRoadmap: unsupported root type " + root.Type)
	}
}

// validateRoadmapRootType returns an error if b is not a valid roadmap scope
// root (milestone, epic, or feature).
func validateRoadmapRootType(b *bean.Bean) error {
	switch b.Type {
	case "milestone", "epic", "feature":
		return nil
	default:
		return fmt.Errorf("roadmap root must be a milestone, epic, or feature, got %s (%s)", b.Type, b.ID)
	}
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run `go test ./internal/commands/... -run 'TestBuildScopedRoadmap|TestValidateRoadmapRootType' -v` — expected to PASS.

- [ ] **Step 7: Run the full roadmap test file to check for regressions**

Run `go test ./internal/commands/... -run 'TestBuildRoadmap|TestBuildScopedRoadmap|TestValidateRoadmapRootType|TestStatusFiltering|TestMilestoneOrder' -v` — expected to PASS; the `childrenIndex` extraction must not change any existing `buildRoadmap` test outcome.

- [ ] **Step 8: Commit**

```bash
git add internal/commands/roadmap.go internal/commands/roadmap_test.go
git commit -m "feat(roadmap): add buildScopedRoadmap for milestone/epic/feature root"
```

---

### Task 2: Markdown template — render `.Root`

**Files:**
- Modify: `internal/commands/roadmap.tmpl:30-72`
- Test: `internal/commands/roadmap_test.go`

**Interfaces:**
- Consumes: `roadmapData.Root`, `rootGroup{Epic, Feature}` (Task 1); pre-existing named templates `epicGroup`, `featureGroup` (`roadmap.tmpl:5-28`); pre-existing `renderRoadmapMarkdown(data *roadmapData, links bool, linkPrefix string) string` (`roadmap.go:498-515`, signature unchanged).
- Produces: no new Go symbols — `renderRoadmapMarkdown` now also handles `data.Root != nil`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/commands/roadmap_test.go`:

```go
func TestRenderRoadmapMarkdownRootEpic(t *testing.T) {
	e := &bean.Bean{ID: "beans-e1", Type: "epic", Title: "Auth", Status: "todo", Path: "e1--auth.md"}
	t1 := &bean.Bean{ID: "beans-t1", Type: "task", Title: "Login", Status: "todo", Path: "t1--login.md"}
	data := &roadmapData{
		Root: &rootGroup{
			Epic: &epicGroup{Epic: e, Items: []*bean.Bean{t1}},
		},
	}

	got := renderRoadmapMarkdown(data, true, "")

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
	t1 := &bean.Bean{ID: "beans-t1", Type: "task", Title: "OIDC", Status: "todo", Path: "t1--oidc.md"}
	data := &roadmapData{
		Root: &rootGroup{
			Feature: &featureGroup{Feature: f, Items: []*bean.Bean{t1}},
		},
	}

	got := renderRoadmapMarkdown(data, true, "")

	if !strings.Contains(got, "#### Feature: SSO") {
		t.Errorf("expected feature heading, got %q", got)
	}
	if !strings.Contains(got, "OIDC") {
		t.Errorf("expected item OIDC to be rendered, got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run `go test ./internal/commands/... -run 'TestRenderRoadmapMarkdownRoot' -v` — expected to FAIL: with `data.Root` set and `Milestones` empty, today's template renders just `# Roadmap` and nothing else, so the `Contains "### Epic: Auth"` assertion fails.

- [ ] **Step 3: Add the `.Root` branch to the template**

Replace `internal/commands/roadmap.tmpl` lines 30-72 (from `# Roadmap` to the final `{{- end}}`) with:

```
# Roadmap
{{- if .Root}}
{{- if .Root.Epic}}
{{template "epicGroup" .Root.Epic}}
{{- end}}
{{- if .Root.Feature}}
{{template "featureGroup" .Root.Feature}}
{{- end}}
{{- else}}
{{range .Milestones}}
## Milestone: {{.Milestone.Title}} {{beanRef .Milestone}}
{{with firstParagraph .Milestone.Body}}
> {{.}}
{{end}}
{{range .Epics -}}
{{template "epicGroup" .}}
{{- end}}
{{- range .Features -}}
{{template "featureGroup" .}}
{{- end}}
{{- if .Other}}
{{- if or (len .Epics) (len .Features)}}
### Miscellaneous
{{end}}

{{range .Other -}}
{{template "beanLine" .}}
{{- end}}
{{- end}}
{{- end}}
{{- if .Unscheduled}}
{{- if len $.Milestones}}
## No Milestone
{{end}}
{{- range .Unscheduled.Epics -}}
{{template "epicGroup" .}}
{{- end}}
{{- range .Unscheduled.Features -}}
{{template "featureGroup" .}}
{{- end}}
{{- if .Unscheduled.Other}}
{{- if or (len .Unscheduled.Epics) (len .Unscheduled.Features)}}
### Miscellaneous
{{end}}

{{range .Unscheduled.Other -}}
{{template "beanLine" .}}
{{- end}}
{{- end}}
{{- end}}
{{- end}}
```

(Everything from `{{range .Milestones}}` to the second-to-last `{{- end}}` is unchanged, just moved inside the new `{{- else}}` branch; the final `{{- end}}` closes the new `{{- if .Root}}`.)

- [ ] **Step 4: Run tests to verify they pass**

Run `go test ./internal/commands/... -run 'TestRenderRoadmapMarkdownRoot' -v` — expected to PASS. If a whitespace mismatch trips an assertion, adjust the `{{-`/`-}}` trim markers around the failing line only — do not change the unrelated `{{- else}}` branch body.

- [ ] **Step 5: Run the full markdown-rendering regression tests**

Run `go test ./internal/commands/... -run 'TestRoadmapMarkdownByteIdentical|TestRoadmapOutputSwitchesOnTTY|TestRenderRoadmapMarkdown' -v` — expected to PASS; the unscoped (`.Root == nil`) rendering path must be byte-identical to before this change.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/roadmap.tmpl internal/commands/roadmap_test.go
git commit -m "feat(roadmap): render .Root in the Markdown template"
```

---

### Task 3: TTY (pretty) renderer — render `.Root`

**Files:**
- Modify: `internal/commands/roadmap_pretty.go:154-160` (`renderRoadmapPretty`)
- Test: `internal/commands/roadmap_pretty_test.go`

**Interfaces:**
- Consumes: `roadmapData.Root`, `rootGroup{Epic, Feature}` (Task 1); pre-existing `renderRoadmapEpicGroup(sb *strings.Builder, eg epicGroup, indent int, width int)` and `renderRoadmapFeatureGroup(sb *strings.Builder, fg featureGroup, indent int, width int)` (`roadmap_pretty.go:204-225`, signatures unchanged).
- Produces: no new Go symbols — `renderRoadmapPretty` now also handles `data.Root != nil`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/commands/roadmap_pretty_test.go`:

```go
func TestRenderRoadmapPrettyRootEpic(t *testing.T) {
	epic := &bean.Bean{ID: "beans-eeee", Title: "Auth", Type: "epic", Status: "todo"}
	leaf := &bean.Bean{ID: "beans-tttt", Title: "Login", Type: "task", Status: "todo", Parent: "beans-eeee"}

	data := &roadmapData{
		Root: &rootGroup{
			Epic: &epicGroup{Epic: epic, Items: []*bean.Bean{leaf}},
		},
	}
	got := renderRoadmapPretty(data, 80)
	lines := strings.Split(got, "\n")
	// [0] Roadmap, [1] separator, [2] blank, [3] the epic row, [4] its leaf.
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines, got %d:\n%s", len(lines), got)
	}
	epicLine, leafLine := lines[3], lines[4]

	if !strings.HasPrefix(epicLine, "▸ Epic") {
		t.Errorf("epic line prefix = %q, want prefix %q", epicLine, "▸ Epic")
	}
	if !strings.Contains(epicLine, "Auth") {
		t.Errorf("epic line missing title: %q", epicLine)
	}
	if !strings.HasPrefix(leafLine, "  - task") {
		t.Errorf("leaf line prefix = %q, want prefix %q", leafLine, "  - task")
	}
	if !strings.Contains(leafLine, "Login") {
		t.Errorf("leaf line missing title: %q", leafLine)
	}
	if strings.Contains(got, "No Milestone") {
		t.Errorf("root-scoped output must not contain the Unscheduled heading: %q", got)
	}
	if strings.Contains(got, "■ Milestone") {
		t.Errorf("root-scoped output must not contain a Milestone row: %q", got)
	}
}

func TestRenderRoadmapPrettyRootFeature(t *testing.T) {
	feat := &bean.Bean{ID: "beans-ffff", Title: "SSO", Type: "feature", Status: "todo", Priority: "high"}
	leaf := &bean.Bean{ID: "beans-tttt", Title: "OIDC", Type: "task", Status: "todo", Parent: "beans-ffff"}

	data := &roadmapData{
		Root: &rootGroup{
			Feature: &featureGroup{Feature: feat, Items: []*bean.Bean{leaf}},
		},
	}
	got := renderRoadmapPretty(data, 80)
	lines := strings.Split(got, "\n")
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 lines, got %d:\n%s", len(lines), got)
	}
	featureLine, leafLine := lines[3], lines[4]

	if !strings.HasPrefix(featureLine, "▪ Feature") {
		t.Errorf("feature line prefix = %q, want prefix %q", featureLine, "▪ Feature")
	}
	if !strings.Contains(featureLine, "SSO") {
		t.Errorf("feature line missing title: %q", featureLine)
	}
	if !strings.Contains(featureLine, "high") {
		t.Errorf("feature row must show priority (D15): %q", featureLine)
	}
	if !strings.HasPrefix(leafLine, "  - task") {
		t.Errorf("leaf line prefix = %q, want prefix %q", leafLine, "  - task")
	}
	if !strings.Contains(leafLine, "OIDC") {
		t.Errorf("leaf line missing title: %q", leafLine)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run `go test ./internal/commands/... -run 'TestRenderRoadmapPrettyRoot' -v` — expected to FAIL: with `data.Root` set and `Milestones`/`Unscheduled` empty, today's `renderRoadmapPretty` returns just the header and separator, so `lines` has fewer than 5 entries and the test fails at the `len(lines) < 5` fatal check.

- [ ] **Step 3: Add the `.Root` branch to `renderRoadmapPretty`**

In `internal/commands/roadmap_pretty.go`, replace:

```go
func renderRoadmapPretty(data *roadmapData, width int) string {
	var sb strings.Builder
	sb.WriteString("Roadmap\n")
	sb.WriteString(strings.Repeat("═", width))
	sb.WriteString("\n")

	for _, mg := range data.Milestones {
```

with:

```go
func renderRoadmapPretty(data *roadmapData, width int) string {
	var sb strings.Builder
	sb.WriteString("Roadmap\n")
	sb.WriteString(strings.Repeat("═", width))
	sb.WriteString("\n")

	if data.Root != nil {
		sb.WriteString("\n")
		if data.Root.Epic != nil {
			renderRoadmapEpicGroup(&sb, *data.Root.Epic, 0, width)
		}
		if data.Root.Feature != nil {
			renderRoadmapFeatureGroup(&sb, *data.Root.Feature, 0, width)
		}
		return sb.String()
	}

	for _, mg := range data.Milestones {
```

- [ ] **Step 4: Run tests to verify they pass**

Run `go test ./internal/commands/... -run 'TestRenderRoadmapPrettyRoot' -v` — expected to PASS.

- [ ] **Step 5: Run the full pretty-renderer regression tests**

Run `go test ./internal/commands/... -run 'TestRenderRoadmapPretty|TestRoadmapLine|TestRoadmapClampWidth' -v` — expected to PASS; the unscoped (`.Root == nil`) rendering path must be unchanged.

- [ ] **Step 6: Commit**

```bash
git add internal/commands/roadmap_pretty.go internal/commands/roadmap_pretty_test.go
git commit -m "feat(roadmap): render .Root in the TTY renderer"
```

---

### Task 4: CLI wiring — positional root ID on `beans roadmap`

**Files:**
- Modify: `internal/commands/roadmap.go:68-111` (`roadmapCmd` struct literal and `RunE`)
- Create: `internal/commands/roadmap_cmd_test.go`

**Interfaces:**
- Consumes: `buildScopedRoadmap(allBeans []*bean.Bean, includeDone bool, root *bean.Bean) *roadmapData`, `validateRoadmapRootType(b *bean.Bean) error` (Task 1); `core.Get(id string) (*bean.Bean, error)` (pre-existing, `pkg/beancore/core.go:309`); `buildRoadmap` (pre-existing, unchanged).
- Produces: `roadmapCmd.Use = "roadmap [id]"`, `roadmapCmd.Args = cobra.MaximumNArgs(1)`; `RunE` now branches on `len(args)`. No new exported Go symbols outside the test file's helpers `setupRoadmapCmdTest`, `resetRoadmapFlags` (test-only).

- [ ] **Step 1: Write the failing tests**

Create `internal/commands/roadmap_cmd_test.go`:

```go
package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
)

// setupRoadmapCmdTest installs a throwaway core+config into the package
// globals roadmapCmd.RunE reads and returns the core for bean creation.
func setupRoadmapCmdTest(t *testing.T) *beancore.Core {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.Default()
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	return testCore
}

// resetRoadmapFlags clears the roadmap* package globals these wiring tests
// touch, so a flag set in one test can't leak into the next.
func resetRoadmapFlags(t *testing.T) {
	t.Helper()
	oldStatus, oldNoStatus, oldJSON := roadmapStatus, roadmapNoStatus, roadmapJSON
	roadmapStatus, roadmapNoStatus, roadmapJSON = nil, nil, false
	t.Cleanup(func() {
		roadmapStatus, roadmapNoStatus, roadmapJSON = oldStatus, oldNoStatus, oldJSON
		roadmapCmd.SetOut(nil)
	})
}

func TestRoadmapCmdJSONScopesToEpicRoot(t *testing.T) {
	testCore := setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)

	epic := &bean.Bean{ID: "beans-epic1", Slug: bean.Slugify("Auth"), Title: "Auth", Status: "todo", Type: "epic"}
	if err := testCore.Create(epic); err != nil {
		t.Fatalf("core.Create(epic) error = %v", err)
	}
	task := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("Login"), Title: "Login", Status: "todo", Type: "task", Parent: epic.ID}
	if err := testCore.Create(task); err != nil {
		t.Fatalf("core.Create(task) error = %v", err)
	}
	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("v1"), Title: "v1", Status: "todo", Type: "milestone"}
	if err := testCore.Create(milestone); err != nil {
		t.Fatalf("core.Create(milestone) error = %v", err)
	}

	out := new(bytes.Buffer)
	roadmapCmd.SetOut(out)
	roadmapJSON = true

	if err := roadmapCmd.RunE(roadmapCmd, []string{epic.ID}); err != nil {
		t.Fatalf("roadmapCmd.RunE() error = %v", err)
	}

	var got roadmapData
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if got.Root == nil || got.Root.Epic == nil {
		t.Fatalf("expected Root.Epic to be set, got %+v", got.Root)
	}
	if got.Root.Epic.Epic.ID != epic.ID {
		t.Errorf("got epic %s, want %s", got.Root.Epic.Epic.ID, epic.ID)
	}
	if len(got.Milestones) != 0 {
		t.Errorf("expected no Milestones in scoped output, got %d", len(got.Milestones))
	}
}

func TestRoadmapCmdJSONScopesToMilestoneRoot(t *testing.T) {
	testCore := setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("v1"), Title: "v1", Status: "todo", Type: "milestone"}
	if err := testCore.Create(milestone); err != nil {
		t.Fatalf("core.Create(milestone) error = %v", err)
	}
	task := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("Docs"), Title: "Docs", Status: "todo", Type: "task", Parent: milestone.ID}
	if err := testCore.Create(task); err != nil {
		t.Fatalf("core.Create(task) error = %v", err)
	}
	otherMilestone := &bean.Bean{ID: "beans-mile2", Slug: bean.Slugify("v2"), Title: "v2", Status: "todo", Type: "milestone"}
	if err := testCore.Create(otherMilestone); err != nil {
		t.Fatalf("core.Create(otherMilestone) error = %v", err)
	}

	out := new(bytes.Buffer)
	roadmapCmd.SetOut(out)
	roadmapJSON = true

	if err := roadmapCmd.RunE(roadmapCmd, []string{milestone.ID}); err != nil {
		t.Fatalf("roadmapCmd.RunE() error = %v", err)
	}

	var got roadmapData
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if len(got.Milestones) != 1 || got.Milestones[0].Milestone.ID != milestone.ID {
		t.Fatalf("got Milestones = %+v, want exactly [%s]", got.Milestones, milestone.ID)
	}
	if got.Unscheduled != nil {
		t.Errorf("expected Unscheduled to be nil for a milestone scope, got %+v", got.Unscheduled)
	}
}

func TestRoadmapCmdRejectsNonContainerRootType(t *testing.T) {
	testCore := setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)

	task := &bean.Bean{ID: "beans-task1", Slug: bean.Slugify("Just a task"), Title: "Just a task", Status: "todo", Type: "task"}
	if err := testCore.Create(task); err != nil {
		t.Fatalf("core.Create(task) error = %v", err)
	}

	err := roadmapCmd.RunE(roadmapCmd, []string{task.ID})
	if err == nil {
		t.Fatal("expected an error for a task-typed root")
	}
	if !strings.Contains(err.Error(), "milestone, epic, or feature") {
		t.Errorf("expected error to name the allowed types, got %q", err.Error())
	}
}

func TestRoadmapCmdRejectsUnknownID(t *testing.T) {
	setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)

	err := roadmapCmd.RunE(roadmapCmd, []string{"beans-doesnotexist"})
	if err == nil {
		t.Fatal("expected an error for an unknown bean ID")
	}
	if !strings.Contains(err.Error(), "unknown bean") {
		t.Errorf("expected error to mention 'unknown bean', got %q", err.Error())
	}
}

func TestRoadmapCmdRejectsStatusFlagWithRootID(t *testing.T) {
	testCore := setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)

	milestone := &bean.Bean{ID: "beans-mile1", Slug: bean.Slugify("v1"), Title: "v1", Status: "todo", Type: "milestone"}
	if err := testCore.Create(milestone); err != nil {
		t.Fatalf("core.Create(milestone) error = %v", err)
	}
	roadmapStatus = []string{"todo"}

	err := roadmapCmd.RunE(roadmapCmd, []string{milestone.ID})
	if err == nil {
		t.Fatal("expected an error combining a root ID with --status")
	}
	if !strings.Contains(err.Error(), "--status") {
		t.Errorf("expected error to mention --status, got %q", err.Error())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run `go test ./internal/commands/... -run 'TestRoadmapCmd' -v` — expected to FAIL: `roadmapCmd.RunE` does not yet accept/branch on a positional argument, so the epic/milestone-scoping assertions fail and the reject-tests get no error (`err == nil`).

- [ ] **Step 3: Wire the positional argument into `roadmapCmd`**

In `internal/commands/roadmap.go`, replace:

```go
var roadmapCmd = &cobra.Command{
	Use:   "roadmap",
	Short: "Generate a Markdown roadmap from milestones and epics",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Query all beans via GraphQL resolver
		resolver := &beangraph.CoreResolver{Core: core}
		allBeans, err := resolver.Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		// Build the roadmap
		data := buildRoadmap(allBeans, roadmapIncludeDone, roadmapStatus, roadmapNoStatus)
```

with:

```go
var roadmapCmd = &cobra.Command{
	Use:   "roadmap [id]",
	Short: "Generate a Markdown roadmap from milestones and epics",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Query all beans via GraphQL resolver
		resolver := &beangraph.CoreResolver{Core: core}
		allBeans, err := resolver.Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		var data *roadmapData
		if len(args) == 1 {
			if len(roadmapStatus) > 0 || len(roadmapNoStatus) > 0 {
				return fmt.Errorf("--status/--no-status cannot be combined with a roadmap root ID")
			}
			root, err := core.Get(args[0])
			if err != nil {
				return fmt.Errorf("unknown bean: %s", args[0])
			}
			if err := validateRoadmapRootType(root); err != nil {
				return err
			}
			data = buildScopedRoadmap(allBeans, roadmapIncludeDone, root)
		} else {
			data = buildRoadmap(allBeans, roadmapIncludeDone, roadmapStatus, roadmapNoStatus)
		}
```

- [ ] **Step 4: Run tests to verify they pass**

Run `go test ./internal/commands/... -run 'TestRoadmapCmd' -v` — expected to PASS.

- [ ] **Step 5: Run the entire `internal/commands` package**

Run `go test ./internal/commands/...` — expected to PASS, no regressions in `order_test.go`, `show_test.go`, or any other command test sharing the `core`/`cfg` package globals.

- [ ] **Step 6: Run the full test suite**

Run `mise test` — expected to PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/commands/roadmap.go internal/commands/roadmap_cmd_test.go
git commit -m "feat(roadmap): scope beans roadmap to a milestone/epic/feature ID"
```
