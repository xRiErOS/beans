package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/spf13/cobra"
)

var (
	progressJSON   bool
	progressParent string
)

// progressBarWidth is the fixed width, in characters, of the plain-text
// progress bar rendered below the per-status counts.
const progressBarWidth = 20

// progressResult is the --json shape for `beans progress`: per-status
// counts plus the derived completed/total/percent figures. It intentionally
// carries no bar string — the bar is a presentation-only concern of the
// plain-text renderer below.
type progressResult struct {
	Counts    map[string]int `json:"counts"`
	Completed int            `json:"completed"`
	Total     int            `json:"total"`
	Percent   int            `json:"percent"`
}

var progressCmd = &cobra.Command{
	Use:   "progress",
	Short: "Show a summary of work status across all beans",
	Long:  `Shows counts by status across every configured status, plus a percent-complete figure (completed / (total - scrapped)). Use --parent to scope the counts to a single bean's descendants (e.g. a milestone or epic) instead of the whole workspace.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		allBeans, err := resolver.Beans(ctx, nil)
		if err != nil {
			return cmdError(progressJSON, output.ErrValidation, "querying beans: %v", err)
		}

		scope := allBeans
		if progressParent != "" {
			parent, err := resolver.Bean(ctx, progressParent)
			if err != nil {
				return cmdError(progressJSON, output.ErrNotFound, "failed to find bean: %v", err)
			}
			if parent == nil {
				return cmdError(progressJSON, output.ErrNotFound, "bean not found: %s", progressParent)
			}
			idx := buildChildrenIndex(allBeans)
			scope = descendants(parent.ID, idx)
		}

		statusNames := cfg.StatusNames()
		counts := make(map[string]int, len(statusNames))
		for _, s := range statusNames {
			counts[s] = 0
		}
		for _, b := range scope {
			counts[b.Status]++
		}

		completed := counts["completed"]
		total := len(scope) - counts["scrapped"]
		// Integer division truncates toward zero (23*100/40 = 57.5 -> 57),
		// matching the epic's own worked example (beans-m364) which states
		// "57%" for that exact input rather than a rounded-up 58%.
		percent := 0
		if total > 0 {
			percent = completed * 100 / total
		}

		if progressJSON {
			result := progressResult{Counts: counts, Completed: completed, Total: total, Percent: percent}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		for _, sc := range cfg.StatusList() {
			// Measure and pad the plain label first, then colour it — never
			// the other way around, or the padding would count escape bytes
			// as visible width.
			label := progressStatusLabel(sc.Name)
			style := lipgloss.NewStyle().Foreground(ui.ResolveColor(sc.Color)).Bold(!sc.Archive)
			fmt.Printf("%s: %d\n", style.Render(label), counts[sc.Name])
		}

		filled, empty := progressBarSegments(percent)
		bar := lipgloss.NewStyle().Foreground(ui.ResolveColor("green")).Render(filled) +
			lipgloss.NewStyle().Foreground(ui.ResolveColor("surface2")).Render(empty)
		fmt.Printf("%s %d%% complete\n", bar, percent)

		return nil
	},
}

// progressStatusLabel renders a status name like "in-progress" as "In
// Progress" for plain-text display, matching the epic's example rendering.
func progressStatusLabel(status string) string {
	words := strings.Split(status, "-")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// progressBarSegments returns the filled and empty plain-text runs of the
// bar separately, so a caller can colour each run independently without
// measuring against text that already carries colour escapes.
func progressBarSegments(percent int) (filled, empty string) {
	n := percent * progressBarWidth / 100
	if n > progressBarWidth {
		n = progressBarWidth
	}
	if n < 0 {
		n = 0
	}
	return strings.Repeat("━", n), strings.Repeat("░", progressBarWidth-n)
}

func RegisterProgressCmd(root *cobra.Command) {
	progressCmd.Flags().BoolVar(&progressJSON, "json", false, "Output as JSON")
	progressCmd.Flags().StringVar(&progressParent, "parent", "", "Scope counts to this bean's descendants")
	root.AddCommand(progressCmd)
}
