package commands

import (
	"context"

	"github.com/xRiErOS/beans/internal/output"
	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/beangraph/model"
	"github.com/spf13/cobra"
)

var (
	tagAdd    []string
	tagRemove []string
	tagJSON   bool
)

var tagCmd = &cobra.Command{
	Use:   "tag <id> [id...]",
	Short: "Add or remove tags on one or more beans",
	Long: `Adds and removes tags on one or more beans in a single call.

Tags are merged, not replaced: --tag adds, --remove-tag takes away, and every
tag not named is left alone. At least one of the two is required. Every ID is
resolved before the first bean is written.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		if len(tagAdd) == 0 && len(tagRemove) == 0 {
			return cmdError(tagJSON, output.ErrValidation, "no tags given (use --tag or --remove-tag)")
		}

		targets, code, err := resolveBatchTargets(ctx, resolver, args)
		if err != nil {
			return cmdError(tagJSON, code, "%s", err)
		}

		// Built here rather than through buildUpdateInput: that helper reads
		// the update* globals, which would let a previous update in the same
		// process leak into a tag call.
		input := model.UpdateBeanInput{
			AddTags:    tagAdd,
			RemoveTags: tagRemove,
		}

		done := make([]*bean.Bean, 0, len(targets))
		for _, target := range targets {
			b, err := resolver.UpdateBean(ctx, target.ID, input)
			if err != nil {
				return emitBatchFailure(tagJSON, done, err)
			}
			done = append(done, b)
		}

		return emitBatchSuccess(tagJSON, done,
			output.SuccessSingle,
			func(b *bean.Bean) string {
				return ui.Success.Render("Tagged ") + ui.ID.Render(b.ID) + " " + b.Title
			})
	},
}

func RegisterTagCmd(root *cobra.Command) {
	tagCmd.Flags().StringArrayVar(&tagAdd, "tag", nil, "Add tag (can be repeated)")
	tagCmd.Flags().StringArrayVar(&tagRemove, "remove-tag", nil, "Remove tag (can be repeated)")
	tagCmd.Flags().BoolVar(&tagJSON, "json", false, "Output as JSON")
	root.AddCommand(tagCmd)
}
