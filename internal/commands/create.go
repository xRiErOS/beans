package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

var (
	createStatus    string
	createType      string
	createPriority  string
	createBody      string
	createBodyFile  string
	createTag       []string
	createParent    string
	createBlocking  []string
	createBlockedBy []string
	createPrefix    string
	createSet       []string
	createUnset     []string
	createOrder     string
	createJSON      bool
)

var createCmd = &cobra.Command{
	Use:     "create [title]",
	Aliases: []string{"c", "new"},
	Short:   "Create a new bean",
	Long:    `Creates a new bean (issue) with a generated ID and optional title.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		if title == "" {
			title = "Untitled"
		}

		// Validate inputs
		if createStatus != "" && !cfg.IsValidStatus(createStatus) {
			return cmdError(createJSON, output.ErrInvalidStatus, "invalid status: %s (must be %s)", createStatus, cfg.StatusList())
		}
		if createType != "" && !cfg.IsValidType(createType) {
			return cmdError(createJSON, output.ErrValidation, "invalid type: %s (must be %s)", createType, cfg.TypeList())
		}
		if createPriority != "" && !cfg.IsValidPriority(createPriority) {
			return cmdError(createJSON, output.ErrValidation, "invalid priority: %s (must be %s)", createPriority, cfg.PriorityList())
		}
		if err := validateExtraKeys(createSet, createUnset); err != nil {
			return cmdError(createJSON, output.ErrValidation, "%s", err)
		}
		normalizedSet, err := normalizeCommitSets(createSet)
		if err != nil {
			return cmdError(createJSON, output.ErrValidation, "%s", err)
		}
		createSet = normalizedSet
		if cmd.Flags().Changed("order") && !bean.IsValidOrderKey(createOrder) {
			return cmdError(createJSON, output.ErrValidation, "invalid order value: %q (must be a non-empty base62 fractional index)", createOrder)
		}

		body, err := resolveContent(createBody, createBodyFile)
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "%s", err)
		}

		// Build GraphQL input
		input := model.CreateBeanInput{Title: title}
		if createStatus != "" {
			input.Status = &createStatus
		} else {
			defaultStatus := cfg.GetDefaultStatus()
			input.Status = &defaultStatus
		}
		if createType != "" {
			input.Type = &createType
		} else {
			defaultType := cfg.GetDefaultType()
			input.Type = &defaultType
		}
		if createPriority != "" {
			input.Priority = &createPriority
		}
		if body != "" {
			input.Body = &body
		}
		if len(createTag) > 0 {
			input.Tags = createTag
		}

		// Add parent
		if createParent != "" {
			input.Parent = &createParent
		}

		// Add blocking
		if len(createBlocking) > 0 {
			input.Blocking = createBlocking
		}

		// Add blocked_by
		if len(createBlockedBy) > 0 {
			input.BlockedBy = createBlockedBy
		}

		// Add custom prefix
		if createPrefix != "" {
			input.Prefix = &createPrefix
		}

		setMap, err := extraSetMap(createSet)
		if err != nil {
			return cmdError(createJSON, output.ErrValidation, "%s", err)
		}

		// Create via core resolver
		resolver := &beangraph.CoreResolver{Core: core}
		b, err := resolver.CreateBean(context.Background(), input, beancore.WithExtraOps(setMap, nil))
		if err != nil {
			return cmdError(createJSON, output.ErrFileError, "failed to create bean: %v", err)
		}

		// An explicit order and --unset aren't part of the generated
		// CreateBeanInput type or WithExtraOps (which only sets), so
		// they're applied and persisted as a second write.
		needsSecondWrite := len(createUnset) > 0 || cmd.Flags().Changed("order")
		if needsSecondWrite {
			// Capture the freshly-created bean's own etag before mutating it,
			// and pass it as ifMatch below instead of nil: nil bypasses
			// optimistic concurrency control -- silently ignored under a
			// normal config, and under require_if_match:true it makes the
			// write fail outright, leaving the bean on disk without the
			// order/unset that were just requested. The bean's own current
			// etag satisfies require_if_match and still catches a genuine
			// concurrent external write between the two writes.
			etag := b.ETag()
			for _, key := range createUnset {
				delete(b.Extra, key)
			}
			if cmd.Flags().Changed("order") {
				b.Order = createOrder
			}
			if err := core.Update(b, &etag); err != nil {
				return mutationError(createJSON, err)
			}
		}

		if createJSON {
			return output.Success(b, "Bean created")
		}

		fmt.Println(ui.Success.Render("Created ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		return nil
	},
}

func RegisterCreateCmd(root *cobra.Command) {
	// Build help text with allowed values from hardcoded config
	statusNames := make([]string, len(config.DefaultStatuses))
	for i, s := range config.DefaultStatuses {
		statusNames[i] = s.Name
	}
	typeNames := make([]string, len(config.DefaultTypes))
	for i, t := range config.DefaultTypes {
		typeNames[i] = t.Name
	}
	priorityNames := make([]string, len(config.DefaultPriorities))
	for i, p := range config.DefaultPriorities {
		priorityNames[i] = p.Name
	}

	createCmd.Flags().StringVarP(&createStatus, "status", "s", "", "Initial status ("+strings.Join(statusNames, ", ")+")")
	createCmd.Flags().StringVarP(&createType, "type", "t", "", "Bean type ("+strings.Join(typeNames, ", ")+")")
	createCmd.Flags().StringVarP(&createPriority, "priority", "p", "", "Priority level ("+strings.Join(priorityNames, ", ")+")")
	createCmd.Flags().StringVarP(&createBody, "body", "d", "", "Body content (use '-' to read from stdin)")
	createCmd.Flags().StringVar(&createBodyFile, "body-file", "", "Read body from file")
	createCmd.Flags().StringArrayVar(&createTag, "tag", nil, "Add tag (can be repeated)")
	createCmd.Flags().StringVar(&createParent, "parent", "", "Parent bean ID")
	createCmd.Flags().StringArrayVar(&createBlocking, "blocking", nil, "ID of bean this blocks (can be repeated)")
	createCmd.Flags().StringArrayVar(&createBlockedBy, "blocked-by", nil, "ID of bean that blocks this one (can be repeated)")
	createCmd.Flags().StringVar(&createPrefix, "prefix", "", "Custom ID prefix (overrides config prefix)")
	createCmd.Flags().StringArrayVar(&createSet, "set", nil, "Set an extra front matter key as key=value (can be repeated)")
	createCmd.Flags().StringArrayVar(&createUnset, "unset", nil, "Remove an extra front matter key (can be repeated)")
	createCmd.Flags().StringVar(&createOrder, "order", "", "Explicit fractional-index order value")
	createCmd.Flags().BoolVar(&createJSON, "json", false, "Output as JSON")
	createCmd.MarkFlagsMutuallyExclusive("body", "body-file")
	root.AddCommand(createCmd)
}
