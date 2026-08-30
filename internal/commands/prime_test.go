package commands

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/hmans/beans/pkg/config"
)

// primeCmdOutput runs the real primeCmd.RunE against a fresh temp directory
// holding cfg's saved .beans.yml, and returns everything it writes to
// stdout. Like runInitCmd (see init_test.go), it drives the package-level
// flag variables RunE reads directly rather than going through cobra's flag
// parsing, and captures stdout the same way captureListStdout does for
// listCmd's --json path.
func primeCmdOutput(t *testing.T, cfg *config.Config) string {
	t.Helper()

	dir := t.TempDir()
	cfg.SetConfigDir(dir)
	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}
	t.Chdir(dir)

	origBeansPath, origConfigPath := beansPath, configPath
	beansPath, configPath = "", ""
	t.Cleanup(func() { beansPath, configPath = origBeansPath, origConfigPath })

	out := captureListStdout(t, func() {
		if err := primeCmd.RunE(primeCmd, nil); err != nil {
			t.Fatalf("primeCmd.RunE: %v", err)
		}
	})
	return string(out)
}

// renderPrimeTemplate executes the embedded agent prompt template with a
// minimal but representative promptData, the same shape primeCmd.RunE
// builds, and returns the rendered text.
func renderPrimeTemplate(t *testing.T) string {
	t.Helper()
	tmpl, err := template.New("prompt").Parse(agentPromptTemplate)
	if err != nil {
		t.Fatalf("parsing agentPromptTemplate: %v", err)
	}
	data := promptData{
		Types:      config.DefaultTypes,
		Statuses:   config.DefaultStatuses,
		Priorities: config.DefaultPriorities,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("executing agentPromptTemplate: %v", err)
	}
	return buf.String()
}

// The prime prompt is the only interface most agents ever read before using
// beans -- a feature that exists only in --help output is, for that
// audience, undiscoverable. beans-xej5 shipped --set/--unset/--where and
// beans order/--sort order/--order without ever touching this template.
func TestPrimeCmdDocumentsCustomFrontMatter(t *testing.T) {
	out := renderPrimeTemplate(t)
	for _, want := range []string{"--set", "--unset", "--where"} {
		if !strings.Contains(out, want) {
			t.Errorf("prime output missing %q (custom front matter undocumented)", want)
		}
	}
}

func TestPrimeCmdDocumentsManualOrdering(t *testing.T) {
	out := renderPrimeTemplate(t)
	for _, want := range []string{"beans order", "--sort order", "--order"} {
		if !strings.Contains(out, want) {
			t.Errorf("prime output missing %q (manual ordering undocumented)", want)
		}
	}
}

// Task 7 (beans-9m5y) replaced the old beans update -s <status> lifecycle
// prose with dedicated wrapper commands. The prime template is the only
// interface most agents read, so each recipe must be documented by name
// with its defining flag -- undocumented commands are undiscoverable.
func TestPrimeCmdDocumentsWorkflowRecipes(t *testing.T) {
	out := renderPrimeTemplate(t)
	for _, want := range []string{
		"beans start <id>",
		"beans complete <id>",
		// The exact signature is pinned by TestPrimeCmdDocumentsBatchAndFilters;
		// here it is only the recipe and its defining flag that must be present.
		"beans scrap",
		"--reason",
		"beans next",
		"beans milestones",
		"beans progress",
		"list --ready",
		"list --is-blocked",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prime output missing %q (workflow recipe undocumented)", want)
		}
	}
}

// The batch signatures and the tag verb exist only for agents that know to
// reach for them. next's filters are the same case: an agent that has only
// ever seen bare `beans next` will keep pulling the wrong bean out of a
// large store.
func TestPrimeCmdDocumentsBatchAndFilters(t *testing.T) {
	out := renderPrimeTemplate(t)
	for _, want := range []string{
		"beans complete <id> [id...]",
		"beans scrap <id> [id...]",
		"beans start <id> [id...]",
		"beans tag <id> [id...]",
		"--remove-tag",
		"beans next --type",
		"--desc",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prime output missing %q (batch/filter capability undocumented)", want)
		}
	}
}

// A failure is the one output shape an agent cannot afford to guess at: it
// decides whether the agent reads stdout or stderr, and whether it treats a
// missing document as a crash. beans-ra75 shipped the shape without touching
// this template.
func TestPrimeCmdDocumentsFailureShape(t *testing.T) {
	out := renderPrimeTemplate(t)
	for _, want := range []string{
		"Run '<command> --help' for usage.",
		`{"success": false`,
		"exits 1",
		"stderr stays empty",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prime output missing %q (failure shape undocumented)", want)
		}
	}
}

func TestRankLinesListEveryOccupiedRank(t *testing.T) {
	got := rankLines(&config.Config{})
	want := []string{
		"rank 1: milestone",
		"rank 2: epic",
		"rank 3: feature",
		"rank 4: bug, task",
	}
	if len(got) != len(want) {
		t.Fatalf("rankLines() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rankLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRankLinesSkipAnEmptyRank(t *testing.T) {
	rank := 1
	c := &config.Config{Types: []config.TypeOverride{{Name: "release", Rank: &rank}}}
	for _, line := range rankLines(c) {
		if strings.HasPrefix(line, "rank 3:") && !strings.Contains(line, "feature") {
			t.Errorf("empty rank rendered: %q", line)
		}
	}
}

func TestPromptTemplateNoLongerCarriesTheLinearChain(t *testing.T) {
	if strings.Contains(agentPromptTemplate, "milestone → epic → feature → task/bug") {
		t.Error("the template still hands agents the misleading linear chain")
	}
	if !strings.Contains(agentPromptTemplate, "higher rank") {
		t.Error("the template does not explain the rank rule")
	}
}

// issueTypesSection extracts the "## Issue Types" section of a rendered
// prime prompt (up to the next "## " heading), so assertions about it don't
// also match an unrelated illustrative example elsewhere in the document
// (e.g. the Recipes section's "...descendants (e.g. a milestone or epic)").
func issueTypesSection(t *testing.T, out string) string {
	t.Helper()
	start := strings.Index(out, "## Issue Types")
	if start == -1 {
		t.Fatalf("prompt has no ## Issue Types section:\n%s", out)
	}
	rest := out[start+len("## Issue Types"):]
	if end := strings.Index(rest, "\n## "); end != -1 {
		rest = rest[:end]
	}
	return rest
}

// An exclusive config (beans init --profile todo, say) is its own complete
// type table: DefaultTypes' milestone/epic/feature/bug must not leak into
// the "## Issue Types" section, and that section must name exactly the same
// types as the "## Relationships" rank lines -- both are derived from the
// one live config an agent is actually working in.
func TestPrimeExclusiveConfigTypesAndRanksAgree(t *testing.T) {
	rank := config.LeafRank
	cfg := config.DefaultWithPrefix("demo-")
	cfg.TypesExclusive = true
	cfg.Types = []config.TypeOverride{{Name: "task", Rank: &rank, Short: "T"}}

	out := primeCmdOutput(t, cfg)
	issueTypes := issueTypesSection(t, out)

	for _, name := range []string{"milestone", "epic", "feature", "bug"} {
		if strings.Contains(issueTypes, name) {
			t.Errorf("exclusive config's issue types section still mentions %q:\n%s", name, issueTypes)
		}
	}
	if !strings.Contains(out, "rank 4: task") {
		t.Errorf("rank section does not name the only configured type:\n%s", out)
	}
	if !strings.Contains(issueTypes, "**task**") {
		t.Errorf("issue types section does not name the only configured type:\n%s", issueTypes)
	}
}

// The ordinary (non-exclusive) path stays pinned: a project that never
// touches types_exclusive still gets the full built-in table in both
// sections.
func TestPrimeDefaultConfigTypesAndRanksAgree(t *testing.T) {
	cfg := config.DefaultWithPrefix("demo-")

	out := primeCmdOutput(t, cfg)

	for _, name := range []string{"milestone", "epic", "feature", "bug", "task"} {
		if !strings.Contains(out, name) {
			t.Errorf("default config's prompt is missing built-in type %q:\n%s", name, out)
		}
	}
	if !strings.Contains(out, "rank 1: milestone") {
		t.Errorf("rank section does not list the default rank-1 type:\n%s", out)
	}
}
