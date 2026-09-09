package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/beangraph/model"
	"github.com/xRiErOS/beans/pkg/candidates"
	"github.com/xRiErOS/beans/internal/output"
	"github.com/xRiErOS/beans/internal/ui"
	"github.com/spf13/cobra"
)

var (
	updateStatus          string
	updateType            string
	updatePriority        string
	updateTitle           string
	updateBody            string
	updateBodyFile        string
	updateBodyReplaceOld  string
	updateBodyReplaceNew  string
	updateBodyAppend      string
	updateParent          string
	updateRemoveParent    bool
	updateBlocking        []string
	updateRemoveBlocking  []string
	updateBlockedBy       []string
	updateRemoveBlockedBy []string
	updateTag             []string
	updateRemoveTag       []string
	updateSet             []string
	updateUnset           []string
	updateIfMatch         string
	updateJSON            bool
)

var updateCmd = &cobra.Command{
	Use:     "update <id>",
	Aliases: []string{"u"},
	Short:   "Update a bean's properties",
	Long:    `Updates one or more properties of an existing bean.`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		if err := validateExtraKeys(updateSet, updateUnset); err != nil {
			return cmdError(updateJSON, output.ErrValidation, "%s", err)
		}
		normalizedSet, err := normalizeCommitSets(updateSet)
		if err != nil {
			return cmdError(updateJSON, output.ErrValidation, "%s", err)
		}
		updateSet = normalizedSet

		// Find the bean
		b, err := resolver.Bean(ctx, args[0])
		if err != nil {
			return cmdError(updateJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}

		// If not found, check the archive and unarchive if present
		wasArchived := false
		if b == nil {
			unarchived, unarchiveErr := core.LoadAndUnarchive(args[0])
			if unarchiveErr != nil {
				return cmdError(updateJSON, output.ErrNotFound, "bean not found: %s", args[0])
			}
			// Re-query to get the model.Bean
			b, err = resolver.Bean(ctx, unarchived.ID)
			if err != nil || b == nil {
				return cmdError(updateJSON, output.ErrNotFound, "bean not found: %s", args[0])
			}
			wasArchived = true
		}

		// Track changes for output
		var changes []string

		// Prepare ifMatch for GraphQL mutations
		var ifMatch *string
		if updateIfMatch != "" {
			ifMatch = &updateIfMatch
		}

		// Build and validate field updates
		input, fieldChanges, err := buildUpdateInput(cmd, b.Tags, b.Body)
		if err != nil {
			return cmdError(updateJSON, output.ErrValidation, "%s", err)
		}
		changes = append(changes, fieldChanges...)

		// Extra front matter keys aren't part of buildUpdateInput's generated
		// UpdateBeanInput, but they still count as a change.
		hasExtraOps := len(updateSet) > 0 || len(updateUnset) > 0
		if hasExtraOps {
			changes = append(changes, "extra")
		}

		// Add ifMatch to input if provided
		if ifMatch != nil {
			input.IfMatch = ifMatch
		}

		// Apply all updates atomically via a single UpdateBean mutation --
		// field updates, body modifications, relationship changes, and
		// extra front matter keys all land in one write under one etag, so
		// a status change and the extra fields a status policy demands
		// (e.g. commit) can't be split across writes.
		if hasFieldUpdates(input) || hasExtraOps {
			setMap, err := extraSetMap(updateSet)
			if err != nil {
				return cmdError(updateJSON, output.ErrValidation, "%s", err)
			}
			b, err = resolver.UpdateBean(ctx, b.ID, input, beancore.WithExtraOps(setMap, updateUnset))
			if err != nil {
				return mutationError(updateJSON, err)
			}
		}

		// Require at least one change
		if len(changes) == 0 {
			return cmdError(updateJSON, output.ErrValidation,
				"no changes specified (use --status, --type, --priority, --title, --body, --parent, --blocking, --blocked-by, --tag, or their --remove-* variants)")
		}

		// Output result. --json returns the resulting bean directly (same
		// schema as `beans show --json`), not a {success,bean,message}
		// envelope: a caller applying many mutations and suppressing
		// stdout still gets a read-after-write state to verify against,
		// with one command instead of two (beans-13ae).
		if updateJSON {
			return output.SuccessSingle(b)
		}

		if wasArchived {
			fmt.Println(ui.Success.Render("Unarchived and updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		} else {
			fmt.Println(ui.Success.Render("Updated ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		}
		return nil
	},
}

// buildUpdateInput constructs the GraphQL input from flags and returns which fields changed.
func buildUpdateInput(cmd *cobra.Command, existingTags []string, currentBody string) (model.UpdateBeanInput, []string, error) {
	var input model.UpdateBeanInput
	var changes []string

	if cmd.Flags().Changed("status") {
		if !cfg.IsValidStatus(updateStatus) {
			return input, nil, fmt.Errorf("invalid status: %s (must be %s)", updateStatus, strings.Join(cfg.StatusNames(), ", "))
		}
		input.Status = &updateStatus
		changes = append(changes, "status")
	}

	if cmd.Flags().Changed("type") {
		if !cfg.IsValidType(updateType) {
			return input, nil, fmt.Errorf("invalid type: %s (must be %s)", updateType, strings.Join(cfg.TypeNames(), ", "))
		}
		input.Type = &updateType
		changes = append(changes, "type")
	}

	if cmd.Flags().Changed("priority") {
		if !cfg.IsValidPriority(updatePriority) {
			return input, nil, fmt.Errorf("invalid priority: %s (must be %s)", updatePriority, strings.Join(cfg.PriorityNames(), ", "))
		}
		input.Priority = &updatePriority
		changes = append(changes, "priority")
	}

	if cmd.Flags().Changed("title") {
		input.Title = &updateTitle
		changes = append(changes, "title")
	}

	// Handle body modifications
	if cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file") {
		// Full body replacement
		body, err := resolveContent(updateBody, updateBodyFile)
		if err != nil {
			return input, nil, err
		}
		input.Body = &body
		changes = append(changes, "body")
	} else if cmd.Flags().Changed("body-replace-old") || cmd.Flags().Changed("body-append") {
		// Body modifications via bodyMod
		bodyMod := &model.BodyModification{}

		if cmd.Flags().Changed("body-replace-old") {
			// --body-replace-old requires --body-replace-new (enforced by MarkFlagsRequiredTogether)
			bodyMod.Replace = []*model.ReplaceOperation{
				{
					Old: bean.UnescapeBody(updateBodyReplaceOld),
					New: bean.UnescapeBody(updateBodyReplaceNew),
				},
			}
		}

		if cmd.Flags().Changed("body-append") {
			appendText, err := resolveAppendContent(updateBodyAppend)
			if err != nil {
				return input, nil, err
			}
			bodyMod.Append = &appendText
		}

		input.BodyMod = bodyMod
		changes = append(changes, "body")
	}

	// Handle tags using granular add/remove (consistent with relationships)
	if len(updateTag) > 0 {
		input.AddTags = updateTag
		changes = append(changes, "tags")
	}
	if len(updateRemoveTag) > 0 {
		input.RemoveTags = updateRemoveTag
		changes = append(changes, "tags")
	}

	// Handle parent relationship
	if cmd.Flags().Changed("parent") {
		input.Parent = &updateParent
		changes = append(changes, "parent")
	} else if updateRemoveParent {
		emptyParent := ""
		input.Parent = &emptyParent
		changes = append(changes, "parent")
	}

	// Handle blocking relationships
	if len(updateBlocking) > 0 {
		input.AddBlocking = updateBlocking
		changes = append(changes, "blocking")
	}
	if len(updateRemoveBlocking) > 0 {
		input.RemoveBlocking = updateRemoveBlocking
		changes = append(changes, "blocking")
	}

	// Handle blocked-by relationships
	if len(updateBlockedBy) > 0 {
		input.AddBlockedBy = updateBlockedBy
		changes = append(changes, "blocked-by")
	}
	if len(updateRemoveBlockedBy) > 0 {
		input.RemoveBlockedBy = updateRemoveBlockedBy
		changes = append(changes, "blocked-by")
	}

	return input, changes, nil
}

// hasFieldUpdates returns true if any field in the input is set.
func hasFieldUpdates(input model.UpdateBeanInput) bool {
	return input.Status != nil || input.Type != nil || input.Priority != nil ||
		input.Title != nil || input.Body != nil || input.BodyMod != nil || input.Tags != nil ||
		input.AddTags != nil || input.RemoveTags != nil ||
		input.Parent != nil || input.AddBlocking != nil || input.RemoveBlocking != nil ||
		input.AddBlockedBy != nil || input.RemoveBlockedBy != nil
}

// isConflictError returns true if the error is an ETag-related conflict error.
func isConflictError(err error) bool {
	var mismatchErr *beancore.ETagMismatchError
	var requiredErr *beancore.ETagRequiredError
	return errors.As(err, &mismatchErr) || errors.As(err, &requiredErr)
}

// mutationErrorCode maps a write-path error to the output error code that
// describes it. Split out of mutationError so that the batch verbs, which
// have to report a failure alongside the beans already written, classify it
// exactly the same way.
func mutationErrorCode(err error) string {
	if isConflictError(err) {
		return output.ErrConflict
	}
	var policyErr *beancore.PolicyViolationError
	if errors.As(err, &policyErr) {
		return output.ErrPolicy
	}
	return output.ErrValidation
}

// mutationError returns a cmdError with the appropriate error code based on the error type.
func mutationError(jsonOutput bool, err error) error {
	return cmdError(jsonOutput, mutationErrorCode(err), "%s", err)
}

func RegisterUpdateCmd(root *cobra.Command) {
	// Help text sources its allowed values from cfg's canonical accessors
	// (pkg/config/config.go StatusNames/TypeNames/PriorityNames), the same
	// read path the completion funcs below use -- no local copy of the name
	// lists (beans-pkq3 AC-02/AC-08, SC-01). cfg is nil here (Register* runs
	// before PersistentPreRunE loads it); StatusList/TypeList/PriorityList
	// are nil-receiver safe and fall back to DefaultStatuses/Types/Priorities.
	statusNames := cfg.StatusNames()
	typeNames := cfg.TypeNames()
	priorityNames := cfg.PriorityNames()

	updateCmd.Flags().StringVarP(&updateStatus, "status", "s", "", "New status ("+strings.Join(statusNames, ", ")+")")
	updateCmd.Flags().StringVarP(&updateType, "type", "t", "", "New type ("+strings.Join(typeNames, ", ")+")")
	updateCmd.Flags().StringVarP(&updatePriority, "priority", "p", "", "New priority ("+strings.Join(priorityNames, ", ")+", or empty to clear)")
	updateCmd.Flags().StringVar(&updateTitle, "title", "", "New title")
	updateCmd.Flags().StringVarP(&updateBody, "body", "d", "", "New body (use '-' to read from stdin)")
	updateCmd.Flags().StringVar(&updateBodyFile, "body-file", "", "Read body from file")
	updateCmd.Flags().StringVar(&updateBodyReplaceOld, "body-replace-old", "", "Text to find and replace (requires --body-replace-new)")
	updateCmd.Flags().StringVar(&updateBodyReplaceNew, "body-replace-new", "", "Replacement text (requires --body-replace-old)")
	updateCmd.Flags().StringVar(&updateBodyAppend, "body-append", "", "Text to append to body (use '-' for stdin)")
	updateCmd.Flags().StringVar(&updateParent, "parent", "", "Set parent bean ID")
	updateCmd.Flags().BoolVar(&updateRemoveParent, "remove-parent", false, "Remove parent")
	updateCmd.Flags().StringArrayVar(&updateBlocking, "blocking", nil, "ID of bean this blocks (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemoveBlocking, "remove-blocking", nil, "ID of bean to unblock (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateBlockedBy, "blocked-by", nil, "ID of bean that blocks this one (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemoveBlockedBy, "remove-blocked-by", nil, "ID of blocker bean to remove (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateTag, "tag", nil, "Add tag (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateRemoveTag, "remove-tag", nil, "Remove tag (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateSet, "set", nil, "Set an extra front matter key as key=value (can be repeated)")
	updateCmd.Flags().StringArrayVar(&updateUnset, "unset", nil, "Remove an extra front matter key (can be repeated)")
	updateCmd.Flags().StringVar(&updateIfMatch, "if-match", "", "Only update if etag matches (optimistic locking)")
	updateCmd.MarkFlagsMutuallyExclusive("parent", "remove-parent")
	updateCmd.Flags().BoolVar(&updateJSON, "json", false, "Output as JSON")
	// body and body-file are mutually exclusive with body modifications
	updateCmd.MarkFlagsMutuallyExclusive("body", "body-file", "body-replace-old")
	updateCmd.MarkFlagsMutuallyExclusive("body", "body-file", "body-append")
	// body-replace-old and body-append can now be used together!
	updateCmd.MarkFlagsRequiredTogether("body-replace-old", "body-replace-new")
	updateCmd.ValidArgsFunction = completionUpTo(1)
	_ = updateCmd.RegisterFlagCompletionFunc("status", statusFlagCompletion)
	_ = updateCmd.RegisterFlagCompletionFunc("type", typeFlagCompletion)
	_ = updateCmd.RegisterFlagCompletionFunc("priority", priorityFlagCompletion)
	_ = updateCmd.RegisterFlagCompletionFunc("tag", tagFlagCompletion)
	_ = updateCmd.RegisterFlagCompletionFunc("parent", updateParentFlagCompletion)
	_ = updateCmd.RegisterFlagCompletionFunc("blocked-by", blockedByFlagCompletion)
	_ = updateCmd.RegisterFlagCompletionFunc("blocking", blockingFlagCompletion)
	root.AddCommand(updateCmd)
}

// statusFlagCompletion offers --status candidates for create/update/list:
// the merged status names from *config.Config's canonical accessor, never
// a second, independently maintained list (beans-pkq3 AC-02, SC-01).
func statusFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return cfg.StatusNames(), completionDirective
}

// typeFlagCompletion is statusFlagCompletion's --type counterpart.
func typeFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return cfg.TypeNames(), completionDirective
}

// priorityFlagCompletion is statusFlagCompletion's --priority counterpart.
func priorityFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return cfg.PriorityNames(), completionDirective
}

// tagFlagCompletion offers every tag already in use across the store,
// annotated with its usage count, sourced from pkg/candidates.TagCandidates
// (beans-v725) -- beans-pkq3 AC-03. Shared by create/update/list/tag's
// --tag.
func tagFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	resolver := &beangraph.CoreResolver{Core: core}
	tags, err := candidates.TagCandidates(context.Background(), resolver)
	if err != nil {
		return nil, completionDirective
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		out = append(out, fmt.Sprintf("%s\t%d bean(s)", t.Tag, t.Count))
	}
	return out, completionDirective
}

// beanFlagCandidates formats an already-filtered bean list as
// shell-completion candidates ("id\ttype title"), mirroring completion.go's
// beanIDCandidates shape for a caller that holds its own pkg/candidates
// result rather than every bean in the store.
func beanFlagCandidates(beans []*bean.Bean) []string {
	out := make([]string, 0, len(beans))
	for _, b := range beans {
		out = append(out, b.ID+"\t"+b.Type+" "+b.Title)
	}
	return out
}

// updateParentFlagCompletion offers update --parent candidates: beans whose
// type is a valid parent for the bean update's own positional argument
// names, excluding that bean and its descendants (candidates.
// ParentCandidates, beans-v725) -- beans-pkq3 AC-04.
func updateParentFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 || core == nil {
		return nil, completionDirective
	}
	b, err := core.Get(args[0])
	if err != nil {
		return nil, completionDirective
	}
	resolver := &beangraph.CoreResolver{Core: core}
	eligible, err := candidates.ParentCandidates(context.Background(), resolver, cfg, []string{b.ID}, []string{b.Type})
	if err != nil {
		return nil, completionDirective
	}
	return beanFlagCandidates(eligible), completionDirective
}

// blockedByFlagCompletion and blockingFlagCompletion both offer every other
// bean in the store (candidates.BlockingCandidates, beans-v725) -- neither
// edge direction excludes descendants (beans-pkq3 AC-04; see pkg/candidates'
// BlockingCandidates doc comment).
func blockedByFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return blockingLikeFlagCompletion(args)
}

func blockingFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return blockingLikeFlagCompletion(args)
}

// blockingLikeFlagCompletion is blockedByFlagCompletion's and
// blockingFlagCompletion's shared body: args[0], when present, is the bean
// being updated, excluded from its own candidate list.
func blockingLikeFlagCompletion(args []string) ([]string, cobra.ShellCompDirective) {
	var beanID string
	if len(args) > 0 {
		beanID = args[0]
	}
	resolver := &beangraph.CoreResolver{Core: core}
	eligible, err := candidates.BlockingCandidates(context.Background(), resolver, cfg, beanID)
	if err != nil {
		return nil, completionDirective
	}
	return beanFlagCandidates(eligible), completionDirective
}
