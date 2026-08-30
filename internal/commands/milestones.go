package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/spf13/cobra"
)

var (
	milestonesJSON bool
	milestonesAll  bool
)

// milestoneEntry pairs a milestone bean with its descendant progress
// counts, per the descendantProgress percent-complete convention.
type milestoneEntry struct {
	Bean      *bean.Bean `json:"bean"`
	Completed int        `json:"completed"`
	Total     int        `json:"total"`
}

// collectMilestones returns every bean on rank 1 whose type takes part in the
// aggregate views, filtered by archive status unless all is set.
func collectMilestones(allBeans []*bean.Bean, all bool) []*bean.Bean {
	var out []*bean.Bean
	for _, b := range allBeans {
		if cfg.RankOf(b.Type) != 1 {
			continue
		}
		if !cfg.IsRoadmapType(b.Type) {
			continue
		}
		if !all && cfg.IsArchiveStatus(b.Status) {
			continue
		}
		out = append(out, b)
	}
	return out
}

// visibleChildrenIndex returns children with every hidden subtree (D15)
// filtered out, so a caller walking it -- e.g. descendantProgress -- cannot
// see into a hidden container or anything beneath it: a hidden epic nested
// under a visible milestone must not contribute to that milestone's progress
// count.
func visibleChildrenIndex(children map[string][]*bean.Bean, hidden map[string]bool) map[string][]*bean.Bean {
	out := make(map[string][]*bean.Bean, len(children))
	for parent, kids := range children {
		var visible []*bean.Bean
		for _, k := range kids {
			if !hidden[k.ID] {
				visible = append(visible, k)
			}
		}
		if len(visible) > 0 {
			out[parent] = visible
		}
	}
	return out
}

var milestonesCmd = &cobra.Command{
	Use:   "milestones",
	Short: "List milestones with descendant progress",
	Long:  `Lists all beans on the top container rank, each annotated with how many of its descendants (via any number of parent levels, e.g. epics and their tasks) are completed. Completed and scrapped milestones are hidden by default; use --all to include them.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		allBeans, err := resolver.Beans(ctx, nil)
		if err != nil {
			return cmdError(milestonesJSON, output.ErrValidation, "querying beans: %v", err)
		}

		milestones := collectMilestones(allBeans, milestonesAll)

		bean.SortByStatusPriorityAndType(milestones, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

		idx := buildChildrenIndex(allBeans)
		hidden := hiddenSubtrees(allBeans, idx)
		visibleIdx := visibleChildrenIndex(idx, hidden)

		entries := make([]milestoneEntry, 0, len(milestones))
		for _, m := range milestones {
			completed, total := descendantProgress(m.ID, visibleIdx)
			entries = append(entries, milestoneEntry{Bean: m, Completed: completed, Total: total})
		}

		if milestonesJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}

		if len(entries) == 0 {
			fmt.Println(ui.Muted.Render("No milestones found."))
			return nil
		}

		for _, e := range entries {
			fmt.Printf("%s %s %s\n",
				ui.ID.Render(e.Bean.ID),
				e.Bean.Title,
				ui.Muted.Render(fmt.Sprintf("(%d/%d completed)", e.Completed, e.Total)),
			)
		}

		return nil
	},
}

func RegisterMilestonesCmd(root *cobra.Command) {
	milestonesCmd.Flags().BoolVar(&milestonesJSON, "json", false, "Output as JSON")
	milestonesCmd.Flags().BoolVar(&milestonesAll, "all", false, "Include completed and scrapped milestones")
	root.AddCommand(milestonesCmd)
}
