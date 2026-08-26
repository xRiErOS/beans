package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/hmans/beans/internal/gitutil"
	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	completeSummary string
	completeCommit  string
	completeSet     []string
	completeJSON    bool
)

var completeCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark a bean as completed",
	Long:  `Marks an existing bean as completed.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		// Find the bean
		b, err := resolver.Bean(ctx, args[0])
		if err != nil {
			return cmdError(completeJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}

		if b == nil {
			return cmdError(completeJSON, output.ErrNotFound, "bean not found: %s", args[0])
		}

		// Validate the status
		if !cfg.IsValidStatus("completed") {
			return cmdError(completeJSON, output.ErrValidation, "invalid status: completed (must be %s)", cfg.StatusList())
		}

		if err := validateExtraKeys(completeSet, nil); err != nil {
			return cmdError(completeJSON, output.ErrValidation, "%s", err)
		}
		normalizedSet, err := normalizeCommitSets(completeSet)
		if err != nil {
			return cmdError(completeJSON, output.ErrValidation, "%s", err)
		}
		setMap, err := extraSetMap(normalizedSet)
		if err != nil {
			return cmdError(completeJSON, output.ErrValidation, "%s", err)
		}

		if completeCommit != "" {
			dir, err := os.Getwd()
			if err != nil {
				return cmdError(completeJSON, output.ErrValidation, "%s", err)
			}
			sha, err := gitutil.ResolveCommit(dir, completeCommit)
			if err != nil {
				return cmdError(completeJSON, output.ErrValidation, "%s", err)
			}
			// --commit and --set on the same field both yield the same
			// normalized SHA type; --commit wins as the more explicit flag.
			setMap[cfg.GetCommitField()] = sha
		}

		// Build the update input
		status := "completed"
		input := model.UpdateBeanInput{
			Status: &status,
		}

		// Handle optional summary
		if completeSummary != "" {
			appendText, err := resolveAppendContent(completeSummary)
			if err != nil {
				return cmdError(completeJSON, output.ErrValidation, "failed to resolve summary: %v", err)
			}
			summarySection := "## Summary of Changes\n\n" + appendText
			input.BodyMod = &model.BodyModification{
				Append: &summarySection,
			}
		}

		// Apply the update
		b, err = resolver.UpdateBean(ctx, b.ID, input, beancore.WithExtraOps(setMap, nil))
		if err != nil {
			return mutationError(completeJSON, err)
		}

		// Output result
		if completeJSON {
			return output.Success(b, "Bean completed")
		}

		fmt.Println(ui.Success.Render("Completed ") + ui.ID.Render(b.ID) + " " + b.Title)
		return nil
	},
}

func RegisterCompleteCmd(root *cobra.Command) {
	completeCmd.Flags().StringVar(&completeSummary, "summary", "", "Optional summary of changes")
	completeCmd.Flags().StringVar(&completeCommit, "commit", "", "Git ref (HEAD, branch, tag, SHA) recorded in the configured commit field")
	completeCmd.Flags().StringArrayVar(&completeSet, "set", nil, "Set an extra front matter key as key=value (can be repeated)")
	completeCmd.Flags().BoolVar(&completeJSON, "json", false, "Output as JSON")
	root.AddCommand(completeCmd)
}
