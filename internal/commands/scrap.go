package commands

import (
	"context"
	"fmt"

	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	scrapReason string
	scrapJSON   bool
)

var scrapCmd = &cobra.Command{
	Use:   "scrap <id>",
	Short: "Mark a bean as scrapped",
	Long:  `Marks an existing bean as scrapped.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		// Validate reason is not empty
		if scrapReason == "" {
			return cmdError(scrapJSON, output.ErrValidation, "reason is required")
		}

		// Find the bean
		b, err := resolver.Bean(ctx, args[0])
		if err != nil {
			return cmdError(scrapJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}

		if b == nil {
			return cmdError(scrapJSON, output.ErrNotFound, "bean not found: %s", args[0])
		}

		// Validate the status
		if !cfg.IsValidStatus("scrapped") {
			return cmdError(scrapJSON, output.ErrValidation, "invalid status: scrapped (must be %s)", cfg.StatusList())
		}

		// Build the update input
		status := "scrapped"
		input := model.UpdateBeanInput{
			Status: &status,
		}

		// Append the reason section
		appendText, err := resolveAppendContent(scrapReason)
		if err != nil {
			return cmdError(scrapJSON, output.ErrValidation, "failed to resolve reason: %v", err)
		}
		reasonSection := "## Reason for Scrapping\n\n" + appendText
		input.BodyMod = &model.BodyModification{
			Append: &reasonSection,
		}

		// Apply the update
		b, err = resolver.UpdateBean(ctx, b.ID, input)
		if err != nil {
			return cmdError(scrapJSON, output.ErrValidation, "failed to update bean: %v", err)
		}

		// Output result
		if scrapJSON {
			return output.Success(b, "Bean scrapped")
		}

		fmt.Println(ui.Success.Render("Scrapped ") + ui.ID.Render(b.ID) + " " + b.Title)
		return nil
	},
}

func RegisterScrapCmd(root *cobra.Command) {
	scrapCmd.Flags().StringVar(&scrapReason, "reason", "", "Reason for scrapping (required)")
	scrapCmd.MarkFlagRequired("reason")
	scrapCmd.Flags().BoolVar(&scrapJSON, "json", false, "Output as JSON")
	root.AddCommand(scrapCmd)
}
