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
	milestonesJSON     bool
	milestonesAll      bool
	milestonesView     string
	milestonesMaxWidth int
	milestonesTags     bool
)

// milestoneEntry pairs a milestone bean with its descendant progress
// counts, per the descendantProgress percent-complete convention.
type milestoneEntry struct {
	Bean      *bean.Bean `json:"bean"`
	Completed int        `json:"completed"`
	Total     int        `json:"total"`
}

var milestonesCmd = &cobra.Command{
	Use:   "milestones",
	Short: "List milestones with descendant progress",
	Long:  `Lists all beans of type "milestone", each annotated with how many of its descendants (via any number of parent levels, e.g. epics and their tasks) are completed. Completed and scrapped milestones are hidden by default; use --all to include them.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		allBeans, err := resolver.Beans(ctx, nil)
		if err != nil {
			return cmdError(milestonesJSON, output.ErrValidation, "querying beans: %v", err)
		}

		var milestones []*bean.Bean
		for _, b := range allBeans {
			if b.Type != "milestone" {
				continue
			}
			if !milestonesAll && cfg.IsArchiveStatus(b.Status) {
				continue
			}
			milestones = append(milestones, b)
		}

		bean.SortByStatusPriorityAndType(milestones, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

		idx := buildChildrenIndex(allBeans)

		entries := make([]milestoneEntry, 0, len(milestones))
		for _, m := range milestones {
			completed, total := descendantProgress(m.ID, idx)
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

		form, ok := ui.ParseForm(milestonesView)
		if !ok {
			return cmdError(milestonesJSON, output.ErrValidation,
				"unknown --view %q: expected table or tree", milestonesView)
		}

		// Every row carries a Progress, even at 0/0 — that is what buys the
		// PROGRESS column from the shared engine without a separate opt-in.
		rows := make([]ui.Row, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, ui.Row{
				Bean:     e.Bean,
				Progress: &ui.Progress{Done: e.Completed, Total: e.Total},
			})
		}

		width := resolveWidth(milestonesMaxWidth, cmd.Flags().Changed("max-width"), cfg)
		fmt.Print(ui.Render(rows, form, "Milestones", width, milestonesTags, cfg))
		return nil
	},
}

func RegisterMilestonesCmd(root *cobra.Command) {
	milestonesCmd.Flags().BoolVar(&milestonesJSON, "json", false, "Output as JSON")
	milestonesCmd.Flags().BoolVar(&milestonesAll, "all", false, "Include completed and scrapped milestones")
	milestonesCmd.Flags().StringVar(&milestonesView, "view", "table",
		"Arrangement: table (flat, sortable) or tree (nested)")
	milestonesCmd.Flags().IntVar(&milestonesMaxWidth, "max-width", 0,
		"Cap the rendered width; 0 disables the cap (default: display.max_width, else 110)")
	milestonesCmd.Flags().BoolVar(&milestonesTags, "tags", false, "Render each milestone's tags")
	root.AddCommand(milestonesCmd)
}
