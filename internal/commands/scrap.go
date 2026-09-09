package commands

import (
	"context"
	"strings"

	"github.com/xRiErOS/beans/internal/output"
	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	scrapReason string
	scrapJSON   bool
)

// scrapTargetStatus is the status scrap writes on success (RunE below) and
// the status scrapCmd's completion narrowing excludes as already-there, so
// the write path and the completion predicate can never drift onto two
// different literals (beans-j5so).
const scrapTargetStatus = "scrapped"

var scrapCmd = &cobra.Command{
	Use:   "scrap <id> [id...]",
	Short: "Mark one or more beans as scrapped",
	Long: `Marks one or more existing beans as scrapped.

--reason applies to every bean named in the call. Every ID is resolved before
the first bean is written, so an unknown ID leaves the whole batch alone.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		// Validate reason is not empty
		if scrapReason == "" {
			return cmdError(scrapJSON, output.ErrValidation, "reason is required")
		}

		targets, code, err := resolveBatchTargets(ctx, resolver, args)
		if err != nil {
			return cmdError(scrapJSON, code, "%s", err)
		}

		// Validate the status
		if !cfg.IsValidStatus(scrapTargetStatus) {
			return cmdError(scrapJSON, output.ErrValidation, "invalid status: %s (must be %s)", scrapTargetStatus, strings.Join(cfg.StatusNames(), ", "))
		}

		// Build the update input
		status := scrapTargetStatus
		input := model.UpdateBeanInput{
			Status: &status,
		}

		// Append the reason section. Resolved once, before the loop: with
		// "-" this reads stdin, which can only be drained once.
		appendText, err := resolveAppendContent(scrapReason)
		if err != nil {
			return cmdError(scrapJSON, output.ErrValidation, "failed to resolve reason: %v", err)
		}
		reasonSection := "## Reason for Scrapping\n\n" + appendText
		input.BodyMod = &model.BodyModification{
			Append: &reasonSection,
		}

		if err := preflightStatusPolicy(targets, status, nil); err != nil {
			return mutationError(scrapJSON, err)
		}

		// Apply the update
		done := make([]*bean.Bean, 0, len(targets))
		for _, target := range targets {
			b, err := resolver.UpdateBean(ctx, target.ID, input)
			if err != nil {
				return emitBatchFailure(scrapJSON, done, err)
			}
			done = append(done, b)
		}

		return emitBatchSuccess(scrapJSON, done,
			func(b *bean.Bean) error { return output.SuccessSingle(b) },
			func(b *bean.Bean) string {
				return ui.Success.Render("Scrapped ") + ui.ID.Render(b.ID) + " " + b.Title
			})
	},
}

func RegisterScrapCmd(root *cobra.Command) {
	scrapCmd.Flags().StringVar(&scrapReason, "reason", "", "Reason for scrapping, applied to every bean in the call (required)")
	scrapCmd.MarkFlagRequired("reason")
	scrapCmd.Flags().BoolVar(&scrapJSON, "json", false, "Output as JSON")
	// scrap excludes beans already scrapped -- scrapping again is not a
	// valid action. A completed bean stays offered: scrap remains a valid
	// way to close out a completed bean differently (beans-j5so).
	scrapCmd.ValidArgsFunction = completionUnboundedFiltered(func(b *bean.Bean) bool {
		return b.Status != scrapTargetStatus
	})
	root.AddCommand(scrapCmd)
}
