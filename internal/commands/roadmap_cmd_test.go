package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hmans/beans/internal/ui"
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
	oldDepth, oldTags := roadmapDepth, roadmapTags
	oldView, oldFormat, oldWidth := roadmapView, roadmapFormat, roadmapWidthFlag
	roadmapStatus, roadmapNoStatus, roadmapJSON = nil, nil, false
	roadmapDepth, roadmapTags = 0, false
	// roadmapView's registered flag default is "tree", not the empty string
	// its zero value would give it -- seed that here so a RunE test doesn't
	// depend on some other test file having already called
	// RegisterRoadmapCmd (whose StringVar call has that side effect).
	roadmapView, roadmapFormat, roadmapWidthFlag = "tree", "", 0
	t.Cleanup(func() {
		roadmapStatus, roadmapNoStatus, roadmapJSON = oldStatus, oldNoStatus, oldJSON
		roadmapDepth, roadmapTags = oldDepth, oldTags
		roadmapView, roadmapFormat, roadmapWidthFlag = oldView, oldFormat, oldWidth
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

// -- --format override (beans-dbph): roadmapOutput stays a pure function of
// its arguments -- these call it directly with an explicit isTTY bool, the
// same pattern roadmap_test.go already uses, rather than trying to fake a
// real terminal through RunE.

func TestRoadmapOutputFormatMarkdownOverridesTTY(t *testing.T) {
	data := roadmapOutputFixture()

	got := roadmapOutput(data, true, roadmapFormatMarkdown, 80, true, "", false, ui.FormTree, config.Default())
	if !strings.HasPrefix(got, "# Roadmap") {
		t.Errorf("got prefix %q, want %q (--format markdown must win over isTTY=true)", got[:min(len(got), 20)], "# Roadmap")
	}
	if !strings.Contains(got, "img.shields.io") {
		t.Error("expected markdown badges even though isTTY was true")
	}
}

func TestRoadmapOutputFormatTTYOverridesNonTTY(t *testing.T) {
	data := roadmapOutputFixture()

	got := roadmapOutput(data, false, roadmapFormatTTY, 80, true, "", false, ui.FormTree, config.Default())
	if !strings.HasPrefix(got, "Roadmap") {
		t.Errorf("got prefix %q, want %q (--format tty must win over isTTY=false)", got[:min(len(got), 20)], "Roadmap")
	}
	if strings.Contains(got, "img.shields.io") {
		t.Error("--format tty must not render markdown badges even though isTTY was false")
	}
}

// TestRoadmapOutputFormatAutoPreservesDetection guards against this step
// shifting the no-flag behaviour: roadmapFormatAuto (the zero value, what
// RunE passes when --format was never given) must produce byte-identical
// output to explicitly requesting the branch isTTY alone would pick -- the
// TTY branch via ui.Render (roadmap.go's TTY branch since beans-dbph Step B,
// which replaced the renderRoadmapPretty call this test used to compare
// against), the non-TTY branch via renderRoadmapMarkdown unchanged.
func TestRoadmapOutputFormatAutoPreservesDetection(t *testing.T) {
	data := roadmapOutputFixture()
	cfg := config.Default()

	if got, want := roadmapOutput(data, true, roadmapFormatAuto, 80, true, "", false, ui.FormTree, cfg),
		ui.Render(roadmapRows(data), ui.FormTree, "Roadmap", roadmapClampWidth(80), false, cfg); got != want {
		t.Errorf("roadmapFormatAuto with isTTY=true diverged from ui.Render:\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := roadmapOutput(data, false, roadmapFormatAuto, 0, true, "", false, ui.FormTree, cfg),
		renderRoadmapMarkdown(data, true, "", false); got != want {
		t.Errorf("roadmapFormatAuto with isTTY=false diverged from renderRoadmapMarkdown:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestRoadmapCmdRejectsInvalidView(t *testing.T) {
	setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)
	roadmapView = "bogus"

	err := roadmapCmd.RunE(roadmapCmd, nil)
	if err == nil {
		t.Fatal("expected an error for an invalid --view value")
	}
	if !strings.Contains(err.Error(), "tree") || !strings.Contains(err.Error(), "table") {
		t.Errorf("expected error to name both valid --view values (tree, table), got %q", err.Error())
	}
}

func TestRoadmapCmdRejectsInvalidFormat(t *testing.T) {
	setupRoadmapCmdTest(t)
	resetRoadmapFlags(t)
	roadmapFormat = "bogus"

	err := roadmapCmd.RunE(roadmapCmd, nil)
	if err == nil {
		t.Fatal("expected an error for an invalid --format value")
	}
	if !strings.Contains(err.Error(), "tty") || !strings.Contains(err.Error(), "markdown") {
		t.Errorf("expected error to name both valid --format values (tty, markdown), got %q", err.Error())
	}
}

// -- TTY branch through the shared layout engine (beans-dbph Step B):
// roadmapOutput's TTY branch now calls ui.Render(roadmapRows(data), ...)
// instead of the bespoke renderRoadmapPretty. These guard the bridge itself
// (roadmapRows) and the two ui.Form outcomes it feeds.

func TestRoadmapCmdTreeViewRendersConnectors(t *testing.T) {
	data := roadmapOutputFixture()

	got := roadmapOutput(data, true, roadmapFormatTTY, 90, true, "", false, ui.FormTree, config.Default())
	if !strings.Contains(got, "└─") {
		t.Errorf("expected tree form to draw a connector (└─) for the nested task, got %q", got)
	}
}

func TestRoadmapCmdTableViewRendersHeaderNoConnectors(t *testing.T) {
	data := roadmapOutputFixture()

	got := roadmapOutput(data, true, roadmapFormatTTY, 90, true, "", false, ui.FormTable, config.Default())
	if !strings.Contains(got, "TITLE") {
		t.Errorf("expected table form to render a column header (TITLE), got %q", got)
	}
	if strings.Contains(got, "└─") || strings.Contains(got, "├─") {
		t.Errorf("table form must not draw tree connectors, got %q", got)
	}
}

// roadmapGroupingFixture builds two milestones plus an unscheduled bucket, so
// a test can check that the milestone/unscheduled grouping survives being
// flattened into ui.FormTable rows (which drop Depth entirely -- Section is
// what carries the grouping across that flattening, per ui/columns.go).
func roadmapGroupingFixture() *roadmapData {
	now := time.Now()
	m1 := &bean.Bean{ID: "beans-m1", Type: "milestone", Title: "v1.0", Status: "todo", CreatedAt: &now, Path: "m1--v10.md"}
	m2 := &bean.Bean{ID: "beans-m2", Type: "milestone", Title: "v2.0", Status: "todo", CreatedAt: &now, Path: "m2--v20.md"}
	t1 := &bean.Bean{ID: "beans-t1", Type: "task", Title: "In v1", Status: "todo", Path: "t1--in-v1.md"}
	t2 := &bean.Bean{ID: "beans-t2", Type: "task", Title: "In v2", Status: "todo", Path: "t2--in-v2.md"}
	orphan := &bean.Bean{ID: "beans-o1", Type: "task", Title: "Orphan", Status: "todo", Path: "o1--orphan.md"}
	return &roadmapData{
		Milestones: []milestoneGroup{
			{Milestone: m1, Other: []*bean.Bean{t1}},
			{Milestone: m2, Other: []*bean.Bean{t2}},
		},
		Unscheduled: &unscheduledGroup{Other: []*bean.Bean{orphan}},
	}
}

func TestRoadmapCmdMilestoneGroupingSurvivesTableFlattening(t *testing.T) {
	data := roadmapGroupingFixture()

	got := roadmapOutput(data, true, roadmapFormatTTY, 90, true, "", false, ui.FormTable, config.Default())
	for _, want := range []string{"beans-m1", "beans-m2", "beans-o1", "No Milestone"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected table output to contain %q, got %q", want, got)
		}
	}
	// The unscheduled bucket's heading must come after both milestones --
	// otherwise "No Milestone" is floating free of the group it labels.
	m2Idx := strings.Index(got, "beans-m2")
	headingIdx := strings.Index(got, "No Milestone")
	if m2Idx < 0 || headingIdx < 0 || headingIdx < m2Idx {
		t.Errorf("expected \"No Milestone\" heading after both milestone rows, got %q", got)
	}
}

// TestRoadmapOutputNoLineExceedsWidthNarrow sweeps cols across and below
// roadmapMinWidth (80), not just comfortably above it -- a sweep that only
// tries wide values can pass without ever exercising roadmapClampWidth's
// floor, or NewColumns' short-form/tag-dropping decisions, which is exactly
// the kind of guard-never-fired false pass this plan has already had to
// remove once (see the task report). The fixture nests three levels deep
// (milestone -> epic -> feature -> task) with long titles and tags so the
// column budget is actually under pressure at these widths.
func TestRoadmapOutputNoLineExceedsWidthNarrow(t *testing.T) {
	now := time.Now()
	m := &bean.Bean{ID: "beans-m1", Type: "milestone", Title: "A moderately long milestone title", Status: "todo", CreatedAt: &now, Tags: []string{"backend", "urgent"}, Path: "m1.md"}
	e := &bean.Bean{ID: "beans-e1", Type: "epic", Title: "A moderately long epic title", Status: "todo", Tags: []string{"auth"}, Path: "e1.md"}
	f := &bean.Bean{ID: "beans-f1", Type: "feature", Title: "A moderately long feature title", Status: "todo", Priority: "high", Path: "f1.md"}
	leaf := &bean.Bean{ID: "beans-t1", Type: "task", Title: "A moderately long leaf title", Status: "in-progress", Priority: "critical", Tags: []string{"blocked"}, Path: "t1.md"}
	data := &roadmapData{
		Milestones: []milestoneGroup{
			{
				Milestone: m,
				Epics: []epicGroup{
					{Epic: e, Features: []featureGroup{{Feature: f, Items: []*bean.Bean{leaf}}}},
				},
			},
		},
	}

	cols := []int{1, 10, 40, 79, 80, 81, 95, 109, 110, 111, 200}
	for _, form := range []ui.Form{ui.FormTree, ui.FormTable} {
		for _, c := range cols {
			got := roadmapOutput(data, true, roadmapFormatTTY, c, true, "", true, form, config.Default())
			clamped := roadmapClampWidth(c)
			lines := strings.Split(got, "\n")
			for _, line := range lines {
				if w := ui.DisplayWidth(line); w > clamped {
					t.Errorf("form=%v cols=%d (clamped=%d): line exceeds width (%d): %q", form, c, clamped, w, line)
				}
			}
			// The divider (second line, both forms) is exactly `width` dashes
			// -- pinning it to `clamped` proves roadmapOutput actually clamps
			// the raw cols before rendering, not just that nothing overflows
			// (a render that ignored clamping and used a narrower raw width
			// throughout would still pass the no-overflow check above).
			if len(lines) < 2 {
				t.Fatalf("form=%v cols=%d: expected at least 2 lines, got %d", form, c, len(lines))
			}
			if w := ui.DisplayWidth(lines[1]); w != clamped {
				t.Errorf("form=%v cols=%d: divider width = %d, want %d (roadmapClampWidth(%d))", form, c, w, clamped, c)
			}
		}
	}
}
