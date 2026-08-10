package commands

import (
	"context"
	"fmt"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	nextJSON bool
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show the highest-priority ready bean",
	Long:  `Finds the highest-priority bean available to start (not blocked, not in-progress/completed/scrapped/draft) and displays it.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		filter := &model.BeanFilter{}
		applyReadyFilter(filter)

		beans, err := resolver.Beans(ctx, filter)
		if err != nil {
			return cmdError(nextJSON, output.ErrValidation, "querying beans: %v", err)
		}

		bean.SortByStatusPriorityAndType(beans, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

		if len(beans) == 0 {
			if nextJSON {
				return output.SuccessMessage("no ready beans found")
			}
			fmt.Println(ui.Muted.Render("No ready beans found. Check " + ui.ID.Render("beans list --is-blocked") + " or " + ui.ID.Render("beans list") + " for what's outstanding."))
			return nil
		}

		// Delegate to show command for display
		showJSON = nextJSON
		return showCmd.RunE(showCmd, []string{beans[0].ID})
	},
}

func RegisterNextCmd(root *cobra.Command) {
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output as JSON")
	root.AddCommand(nextCmd)
}
