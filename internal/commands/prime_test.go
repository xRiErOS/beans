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
		GraphQLSchema: "",
		Types:         config.DefaultTypes,
		Statuses:      config.DefaultStatuses,
		Priorities:    config.DefaultPriorities,
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
