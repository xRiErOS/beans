package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/hmans/beans/internal/output"
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
			scope = descendants(progressParent, idx)
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

		for _, s := range statusNames {
			fmt.Printf("%s: %d\n", progressStatusLabel(s), counts[s])
		}
		fmt.Println(progressBar(percent) + fmt.Sprintf(" %d%% complete", percent))

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

// progressBar renders a fixed-width unicode block bar scaled by percent.
func progressBar(percent int) string {
	filled := percent * progressBarWidth / 100
	if filled > progressBarWidth {
		filled = progressBarWidth
	}
	if filled < 0 {
		filled = 0
	}
	return strings.Repeat("━", filled) + strings.Repeat("░", progressBarWidth-filled)
}

func RegisterProgressCmd(root *cobra.Command) {
	progressCmd.Flags().BoolVar(&progressJSON, "json", false, "Output as JSON")
	progressCmd.Flags().StringVar(&progressParent, "parent", "", "Scope counts to this bean's descendants")
	root.AddCommand(progressCmd)
}
