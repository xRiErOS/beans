package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

// setupGraphTest installs a throwaway core and default config into the
// package globals graphCmd.RunE reads, mirroring setupTagTest.
func setupGraphTest(t *testing.T) {
	t.Helper()
	beansDir := filepath.Join(t.TempDir(), ".beans")
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
}

// mkGraphBean creates and persists a bean with the given relationships.
func mkGraphBean(t *testing.T, id, title, beanType, parent string, blocking, blockedBy []string) *bean.Bean {
	t.Helper()
	b := &bean.Bean{
		ID:        id,
		Slug:      bean.Slugify(title),
		Title:     title,
		Status:    "todo",
		Type:      beanType,
		Parent:    parent,
		Blocking:  blocking,
		BlockedBy: blockedBy,
	}
	if err := core.Create(b); err != nil {
		t.Fatalf("core.Create(%s) error = %v", id, err)
	}
	return b
}

// seedGraphFixtures creates the standard fixture set used by most tests:
//
//	beans-aaaa  Parent epic   (epic)
//	beans-bbbb  Child task    (task)  Parent: aaaa, Blocking: [cccc]
//	beans-cccc  Blocked task  (task)  BlockedBy: [bbbb]
//	beans-dddd  Lonely task   (task)  Parent: beans-zzzz (dangling)
func seedGraphFixtures(t *testing.T) {
	t.Helper()
	mkGraphBean(t, "beans-aaaa", "Parent epic", "epic", "", nil, nil)
	mkGraphBean(t, "beans-bbbb", "Child task", "task", "beans-aaaa", []string{"beans-cccc"}, nil)
	mkGraphBean(t, "beans-cccc", "Blocked task", "task", "", nil, []string{"beans-bbbb"})
	mkGraphBean(t, "beans-dddd", "Lonely task", "task", "beans-zzzz", nil, nil)
}

// resetGraphFlags clears the graph* package globals to their registered
// defaults and restores the previous values afterwards.
func resetGraphFlags(t *testing.T) {
	t.Helper()
	oldFormat, oldRelation, oldDepth := graphFormat, graphRelation, graphDepth
	graphFormat, graphRelation, graphDepth = "dot", nil, 1
	t.Cleanup(func() {
		graphFormat, graphRelation, graphDepth = oldFormat, oldRelation, oldDepth
	})
}

// graphCmdWithFlags builds a throwaway command carrying the same flags as
// graphCmd, mirroring createCmdWithOrderFlag/milestonesCmdWithFlags: the
// package-level graphCmd singleton must not be registered twice, but RunE
// needs a *cobra.Command whose Flags().Changed("depth") behaves correctly
// under real flag parsing.
func graphCmdWithFlags() *cobra.Command {
	c := &cobra.Command{Use: "graph"}
	c.Flags().StringVar(&graphFormat, "format", "dot", "")
	c.Flags().StringArrayVar(&graphRelation, "relation", nil, "")
	c.Flags().IntVar(&graphDepth, "depth", 1, "")
	return c
}

// runGraph parses flags then args as a real cobra invocation would, and
// returns graphCmd.RunE's captured stdout.
func runGraph(t *testing.T, args ...string) (string, error) {
	t.Helper()
	c := graphCmdWithFlags()
	var buf bytes.Buffer
	c.SetOut(&buf)
	if err := c.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags(%v) error = %v", args, err)
	}
	err := graphCmd.RunE(c, c.Flags().Args())
	return buf.String(), err
}

func TestGraphDotDeduplicatesMirroredBlockEdge(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t)
	if err != nil {
		t.Fatalf("runGraph() error = %v", err)
	}
	if got := strings.Count(out, `"beans-bbbb" -> "beans-cccc" [label="blocks"]`); got != 1 {
		t.Errorf("blocks edge count = %d, want 1; output:\n%s", got, out)
	}
	if !strings.Contains(out, `"beans-aaaa" -> "beans-bbbb" [label="parent"]`) {
		t.Errorf("missing parent edge; output:\n%s", out)
	}
}

func TestGraphSkipsDanglingAndSelfLinks(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t)
	if err != nil {
		t.Fatalf("runGraph() error = %v", err)
	}
	if strings.Contains(out, "beans-zzzz") {
		t.Errorf("output mentions dangling target beans-zzzz; output:\n%s", out)
	}
	if !strings.Contains(out, `"beans-dddd"`) {
		t.Errorf("beans-dddd node missing despite its own dangling parent link; output:\n%s", out)
	}
}

func TestGraphASCIIEdgeList(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t, "--format", "ascii")
	if err != nil {
		t.Fatalf("runGraph() error = %v", err)
	}
	want := "beans-aaaa ──parent──> beans-bbbb\nbeans-bbbb ──blocks──> beans-cccc\n"
	if out != want {
		t.Errorf("ascii output = %q, want %q", out, want)
	}
}

func TestGraphRelationFilter(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t, "--relation", "parent")
	if err != nil {
		t.Fatalf("runGraph(--relation parent) error = %v", err)
	}
	if strings.Contains(out, `label="blocks"`) {
		t.Errorf("--relation parent leaked a blocks edge; output:\n%s", out)
	}

	out, err = runGraph(t, "--relation", "blocks")
	if err != nil {
		t.Fatalf("runGraph(--relation blocks) error = %v", err)
	}
	if strings.Contains(out, `label="parent"`) {
		t.Errorf("--relation blocks leaked a parent edge; output:\n%s", out)
	}

	_, err = runGraph(t, "--relation", "bogus")
	if err == nil || !strings.Contains(err.Error(), "invalid --relation") {
		t.Errorf("runGraph(--relation bogus) error = %v, want message containing %q", err, "invalid --relation")
	}
}

func TestGraphScopeDepth(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t, "beans-aaaa")
	if err != nil {
		t.Fatalf("runGraph(beans-aaaa) error = %v", err)
	}
	if !strings.Contains(out, `label="parent"`) {
		t.Errorf("depth-1 scope missing the parent edge; output:\n%s", out)
	}
	if strings.Contains(out, `"beans-bbbb" -> "beans-cccc"`) {
		t.Errorf("depth-1 scope leaked the 2-hop blocks edge; output:\n%s", out)
	}

	out, err = runGraph(t, "beans-aaaa", "--depth", "2")
	if err != nil {
		t.Fatalf("runGraph(beans-aaaa --depth 2) error = %v", err)
	}
	if !strings.Contains(out, `"beans-bbbb" -> "beans-cccc"`) {
		t.Errorf("depth-2 scope missing the 2-hop blocks edge; output:\n%s", out)
	}

	out, err = runGraph(t, "beans-aaaa", "--depth", "0")
	if err != nil {
		t.Fatalf("runGraph(beans-aaaa --depth 0) error = %v", err)
	}
	if !strings.Contains(out, `"beans-bbbb" -> "beans-cccc"`) {
		t.Errorf("depth-0 scope missing the 2-hop blocks edge; output:\n%s", out)
	}
}

func TestGraphJSONIsolatedNode(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t, "beans-dddd", "--format", "json")
	if err != nil {
		t.Fatalf("runGraph(beans-dddd --format json) error = %v", err)
	}
	var model graphModel
	if err := json.Unmarshal([]byte(out), &model); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", out, err)
	}
	if len(model.Nodes) != 1 || model.Nodes[0].ID != "beans-dddd" {
		t.Errorf("nodes = %+v, want exactly one node beans-dddd", model.Nodes)
	}
	if len(model.Edges) != 0 {
		t.Errorf("edges = %+v, want none", model.Edges)
	}
}

func TestGraphNodeFillcolorComesFromConfig(t *testing.T) {
	withTrueColor(t)
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	out, err := runGraph(t)
	if err != nil {
		t.Fatalf("runGraph() error = %v", err)
	}
	sc := cfg.GetStatus("todo")
	if sc == nil || sc.Color == "" {
		t.Fatalf("default config has no color for status todo")
	}
	want := string(ui.ResolveColor(sc.Color))
	if !strings.Contains(out, `fillcolor="`+want+`"`) {
		t.Errorf("output missing fillcolor=%q derived from config; output:\n%s", want, out)
	}
}

func TestGraphDotEscapesTitleQuotes(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	mkGraphBean(t, "beans-quot", `He said "hi"`, "task", "", nil, nil)

	out, err := runGraph(t)
	if err != nil {
		t.Fatalf("runGraph() error = %v", err)
	}
	if !strings.Contains(out, `He said \"hi\"`) {
		t.Errorf("output missing escaped title; output:\n%s", out)
	}
	if strings.Contains(out, `He said "hi"`) {
		t.Errorf("output contains unescaped title quotes; output:\n%s", out)
	}
}

func TestGraphRejectsDepthWithoutID(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	_, err := runGraph(t, "--depth", "2")
	if err == nil || !strings.Contains(err.Error(), "--depth requires a bean id") {
		t.Errorf("error = %v, want message containing %q", err, "--depth requires a bean id")
	}
}

func TestGraphRejectsUnknownFormat(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	_, err := runGraph(t, "--format", "yaml")
	if err == nil || !strings.Contains(err.Error(), "invalid --format") {
		t.Errorf("error = %v, want message containing %q", err, "invalid --format")
	}
}

func TestGraphRejectsUnknownBeanID(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)
	seedGraphFixtures(t)

	_, err := runGraph(t, "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "bean not found: nonexistent") {
		t.Errorf("error = %v, want message containing %q", err, "bean not found: nonexistent")
	}
}

func TestGraphEmptyStoreIsNotAnError(t *testing.T) {
	setupGraphTest(t)
	resetGraphFlags(t)

	out, err := runGraph(t)
	if err != nil {
		t.Fatalf("runGraph() error = %v", err)
	}
	if !strings.Contains(out, "digraph beans {") || !strings.Contains(out, "}") {
		t.Errorf("dot output on empty store = %q, want a valid empty digraph", out)
	}

	out, err = runGraph(t, "--format", "ascii")
	if err != nil {
		t.Fatalf("runGraph(--format ascii) error = %v", err)
	}
	if out != "no relationships\n" {
		t.Errorf("ascii output on empty store = %q, want %q", out, "no relationships\n")
	}
}
