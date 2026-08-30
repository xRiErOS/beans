package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/xRiErOS/beans/internal/output"
	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	nextType   []string
	nextTag    []string
	nextParent string
	nextSort   string
	nextDesc   bool
	nextJSON   bool
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Show the highest-priority ready bean",
	Long: `Finds the highest-priority bean available to start (not blocked, not in-progress/completed/scrapped/draft) and displays it.

The --type, --tag, --parent and --sort flags mean the same as in ` + "`beans list`" + `, so a narrowed
query can be moved between the two commands unchanged.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		// Same flag names and semantics as list, so that narrowing a query
		// does not require translating it between the two commands.
		filter := &model.BeanFilter{
			Type: nextType,
			Tags: nextTag,
		}
		if nextParent != "" {
			filter.ParentID = &nextParent
		}
		applyReadyFilter(filter)

		beans, err := resolver.Beans(ctx, filter)
		if err != nil {
			return cmdError(nextJSON, output.ErrValidation, "querying beans: %v", err)
		}

		sortBeans(beans, nextSort, nextDesc, cfg)

		if len(beans) == 0 {
			return reportNoReadyBeans(describeNextFilters())
		}

		// Delegate to show command for display
		showJSON = nextJSON
		return showCmd.RunE(showCmd, []string{beans[0].ID})
	},
}

// describeNextFilters renders the active narrowing flags as a human-readable
// list. It returns an empty string when next was called unfiltered, which is
// what keeps the unfiltered message byte-identical to earlier releases.
func describeNextFilters() string {
	var parts []string
	if len(nextType) > 0 {
		parts = append(parts, "--type "+strings.Join(nextType, ","))
	}
	if len(nextTag) > 0 {
		parts = append(parts, "--tag "+strings.Join(nextTag, ","))
	}
	if nextParent != "" {
		parts = append(parts, "--parent "+nextParent)
	}
	return strings.Join(parts, " ")
}

// reportNoReadyBeans prints the empty-result message. With filters active it
// names them, so that "nothing is ready" stays distinguishable from "the
// filter was too narrow".
func reportNoReadyBeans(filters string) error {
	if filters == "" {
		if nextJSON {
			return output.SuccessMessage("no ready beans found")
		}
		fmt.Println(ui.Muted.Render("No ready beans found. Check " + ui.ID.Render("beans list --is-blocked") + " or " + ui.ID.Render("beans list") + " for what's outstanding."))
		return nil
	}

	if nextJSON {
		return output.SuccessMessage("no ready beans found matching " + filters)
	}
	fmt.Println(ui.Muted.Render("No ready beans found matching "+ui.ID.Render(filters)+". Widen the filter, or check ") + ui.ID.Render("beans list "+filters) + ui.Muted.Render(" for what's outstanding."))
	return nil
}

func RegisterNextCmd(root *cobra.Command) {
	nextCmd.Flags().StringArrayVarP(&nextType, "type", "t", nil, "Filter by type (can be repeated, OR logic)")
	nextCmd.Flags().StringArrayVar(&nextTag, "tag", nil, "Filter by tag (can be repeated, OR logic)")
	nextCmd.Flags().StringVar(&nextParent, "parent", "", "Filter by parent ID")
	nextCmd.Flags().StringVar(&nextSort, "sort", "", "Sort by: created, updated, status, priority, id, order (order is scoped per parent, so pair it with --parent) (default: status, priority, type, title)")
	nextCmd.Flags().BoolVar(&nextDesc, "desc", false, "Reverse the sort order")
	nextCmd.Flags().BoolVar(&nextJSON, "json", false, "Output as JSON")
	root.AddCommand(nextCmd)
}
