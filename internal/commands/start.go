package commands

import (
	"context"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	startJSON bool
)

var startCmd = &cobra.Command{
	Use:   "start <id> [id...]",
	Short: "Mark one or more beans as in-progress",
	Long: `Marks one or more existing beans as in-progress.

A single bean is displayed in full, as beans show would. Several beans give one
confirmation line each, because n full bodies on one screen help nobody. Every
ID is resolved before the first bean is written.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		targets, code, err := resolveBatchTargets(ctx, resolver, args)
		if err != nil {
			return cmdError(startJSON, code, "%s", err)
		}

		// Validate the status
		if !cfg.IsValidStatus("in-progress") {
			return cmdError(startJSON, output.ErrValidation, "invalid status: in-progress (must be %s)", cfg.StatusList())
		}

		// Build the update input
		status := "in-progress"
		input := model.UpdateBeanInput{
			Status: &status,
		}

		if err := preflightStatusPolicy(targets, status, nil); err != nil {
			return mutationError(startJSON, err)
		}

		// Apply the update
		done := make([]*bean.Bean, 0, len(targets))
		for _, target := range targets {
			b, err := resolver.UpdateBean(ctx, target.ID, input)
			if err != nil {
				return emitBatchFailure(startJSON, done, err)
			}
			done = append(done, b)
		}

		if len(done) == 1 {
			// Delegate to show command for display
			showJSON = startJSON
			return showCmd.RunE(showCmd, []string{done[0].ID})
		}

		return emitBatchSuccess(startJSON, done,
			func(b *bean.Bean) error { return output.SuccessSingle(b) },
			func(b *bean.Bean) string {
				return ui.Success.Render("Started ") + ui.ID.Render(b.ID) + " " + b.Title
			})
	},
}

func RegisterStartCmd(root *cobra.Command) {
	startCmd.Flags().BoolVar(&startJSON, "json", false, "Output as JSON")
	root.AddCommand(startCmd)
}
