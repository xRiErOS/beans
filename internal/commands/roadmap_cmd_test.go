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

	got := roadmapOutput(data, true, roadmapFormatMarkdown, 80, true, "", false)
	if !strings.HasPrefix(got, "# Roadmap") {
		t.Errorf("got prefix %q, want %q (--format markdown must win over isTTY=true)", got[:min(len(got), 20)], "# Roadmap")
	}
	if !strings.Contains(got, "img.shields.io") {
		t.Error("expected markdown badges even though isTTY was true")
	}
}

func TestRoadmapOutputFormatTTYOverridesNonTTY(t *testing.T) {
	data := roadmapOutputFixture()

	got := roadmapOutput(data, false, roadmapFormatTTY, 80, true, "", false)
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
// output to the pre-existing isTTY-only switch, in both directions.
func TestRoadmapOutputFormatAutoPreservesDetection(t *testing.T) {
	data := roadmapOutputFixture()

	if got, want := roadmapOutput(data, true, roadmapFormatAuto, 80, true, "", false),
		renderRoadmapPretty(data, roadmapClampWidth(80), false); got != want {
		t.Errorf("roadmapFormatAuto with isTTY=true diverged from renderRoadmapPretty:\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := roadmapOutput(data, false, roadmapFormatAuto, 0, true, "", false),
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
