package commands

import (
	"context"

	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/spf13/cobra"
)

var (
	startJSON bool
)

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Mark a bean as in-progress",
	Long:  `Marks an existing bean as in-progress and displays it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		// Find the bean
		b, err := resolver.Bean(ctx, args[0])
		if err != nil {
			return cmdError(startJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}

		if b == nil {
			return cmdError(startJSON, output.ErrNotFound, "bean not found: %s", args[0])
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

		// Apply the update
		b, err = resolver.UpdateBean(ctx, b.ID, input)
		if err != nil {
			return cmdError(startJSON, output.ErrValidation, "failed to update bean: %v", err)
		}

		// Delegate to show command for display
		showJSON = startJSON
		return showCmd.RunE(showCmd, []string{b.ID})
	},
}

func RegisterStartCmd(root *cobra.Command) {
	startCmd.Flags().BoolVar(&startJSON, "json", false, "Output as JSON")
	root.AddCommand(startCmd)
}
