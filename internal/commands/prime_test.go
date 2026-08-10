package commands

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/hmans/beans/pkg/config"
)

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
		"beans scrap <id> --reason",
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
