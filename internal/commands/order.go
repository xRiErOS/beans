package commands

import (
	"context"
	"fmt"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/spf13/cobra"
)

var (
	orderAfter  string
	orderBefore string
	orderFirst  bool
	orderLast   bool
	orderJSON   bool
)

var orderCmd = &cobra.Command{
	Use:   "order <id>",
	Short: "Place a bean relative to its siblings",
	Long: `Sets a bean's manual order key relative to its siblings under the same
parent, using a fractional index. This means a move only ever writes the one
bean file being placed, never the neighbours around it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		resolver := &beangraph.CoreResolver{Core: core}

		if !orderFirst && !orderLast && orderAfter == "" && orderBefore == "" {
			return cmdError(orderJSON, output.ErrValidation,
				"one of --after, --before, --first, or --last is required")
		}

		b, err := resolver.Bean(ctx, args[0])
		if err != nil {
			return cmdError(orderJSON, output.ErrNotFound, "failed to find bean: %v", err)
		}
		if b == nil {
			return cmdError(orderJSON, output.ErrNotFound, "bean not found: %s", args[0])
		}

		siblings, err := orderSiblingsOf(resolver, ctx, b)
		if err != nil {
			return cmdError(orderJSON, output.ErrValidation, "%s", err)
		}

		newOrder, err := computeOrderPlacement(resolver, ctx, b, siblings)
		if err != nil {
			return cmdError(orderJSON, output.ErrValidation, "%s", err)
		}

		// Pass the bean's own etag as ifMatch instead of nil: nil bypasses
		// optimistic concurrency control entirely -- silently ignored under a
		// normal config, and under require_if_match:true it makes the write
		// fail outright. b was loaded via resolver.Bean/Core.Load, so its
		// ETag() is the etag of the bytes actually on disk (cached at parse
		// time), not a re-render of the in-memory struct -- see bean.ETag.
		etag := b.ETag()
		b.Order = newOrder
		if err := core.Update(b, &etag); err != nil {
			return mutationError(orderJSON, err)
		}

		if orderJSON {
			return output.Success(b, "Bean order updated")
		}
		fmt.Println(ui.Success.Render("Ordered ") + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path))
		return nil
	},
}

// orderSiblingsOf returns b's siblings (same Parent, b itself excluded),
// sorted by their current Order. Filtering happens manually rather than via
// model.BeanFilter.ParentID because that filter treats an empty ParentID as
// "no filter" — which would return every bean instead of just the top-level
// ones for a parent-less bean.
func orderSiblingsOf(resolver *beangraph.CoreResolver, ctx context.Context, b *bean.Bean) ([]*bean.Bean, error) {
	all, err := resolver.Beans(ctx, nil)
	if err != nil {
		return nil, err
	}

	var siblings []*bean.Bean
	for _, s := range all {
		if s.ID == b.ID {
			continue
		}
		if s.Parent == b.Parent {
			siblings = append(siblings, s)
		}
	}
	bean.SortByOrder(siblings)
	return siblings, nil
}

// computeOrderPlacement resolves the requested placement flag into a
// concrete fractional-index Order value for b, given its sorted siblings.
func computeOrderPlacement(resolver *beangraph.CoreResolver, ctx context.Context, b *bean.Bean, siblings []*bean.Bean) (string, error) {
	switch {
	case orderFirst:
		if len(siblings) == 0 {
			return bean.OrderBetween("", ""), nil
		}
		return bean.OrderBetween("", siblings[0].Order), nil

	case orderLast:
		return bean.OrderBetween(lastOrderedSibling(siblings), ""), nil

	case orderAfter != "":
		return orderRelativeTo(resolver, ctx, b, siblings, orderAfter, true)

	case orderBefore != "":
		return orderRelativeTo(resolver, ctx, b, siblings, orderBefore, false)
	}

	return "", fmt.Errorf("one of --after, --before, --first, or --last is required")
}

// lastOrderedSibling returns the Order of the last sibling (in siblings'
// SortByOrder order) that actually carries one, or "" if none do.
// SortByOrder sorts Order-having siblings before Order-less ones, so
// siblings[len(siblings)-1] is only a safe stand-in for "the sibling with
// the greatest Order" when every sibling happens to have one -- on a store
// where some don't (the normal starting state, since nothing wrote Order
// before this feature existed), it silently picks an Order-less sibling and
// yields "" as if there were no ordered siblings at all yet, which is wrong
// once at least one actually has an Order.
func lastOrderedSibling(siblings []*bean.Bean) string {
	for i := len(siblings) - 1; i >= 0; i-- {
		if siblings[i].Order != "" {
			return siblings[i].Order
		}
	}
	return ""
}

// orderRelativeTo resolves --after/--before: it looks up the named sibling,
// validates it shares b's parent (R-12: order is scoped per parent), and
// returns a key between that sibling and its neighbour on the requested
// side.
func orderRelativeTo(resolver *beangraph.CoreResolver, ctx context.Context, b *bean.Bean, siblings []*bean.Bean, refID string, after bool) (string, error) {
	ref, err := resolver.Bean(ctx, refID)
	if err != nil {
		return "", fmt.Errorf("failed to find bean %s: %w", refID, err)
	}
	if ref == nil {
		return "", fmt.Errorf("bean not found: %s", refID)
	}
	if ref.ID == b.ID {
		return "", fmt.Errorf("cannot place a bean relative to itself")
	}
	if ref.Parent != b.Parent {
		return "", fmt.Errorf("order is scoped per parent: %s has a different parent than %s", ref.ID, b.ID)
	}
	if ref.Order == "" {
		return "", fmt.Errorf("bean %s has no explicit order yet; place it first (e.g. `beans order %s --last`) before placing another bean relative to it", ref.ID, ref.ID)
	}

	idx := -1
	for i, s := range siblings {
		if s.ID == ref.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "", fmt.Errorf("bean not found: %s", refID)
	}

	if after {
		next := ""
		if idx+1 < len(siblings) {
			next = siblings[idx+1].Order
		}
		return bean.OrderBetween(ref.Order, next), nil
	}

	prev := ""
	if idx > 0 {
		prev = siblings[idx-1].Order
	}
	return bean.OrderBetween(prev, ref.Order), nil
}

// RegisterOrderCmd adds the order command to root. Flag registration is
// idempotent (guarded by a Lookup check) because orderCmd is a package-level
// singleton and tests that want to exercise cobra's flag-group validation
// (MarkFlagsMutuallyExclusive fires during Execute(), not when calling RunE
// directly) need to register it into a throwaway root without panicking on
// a second flag definition.
func RegisterOrderCmd(root *cobra.Command) {
	if orderCmd.Flags().Lookup("after") == nil {
		orderCmd.Flags().StringVar(&orderAfter, "after", "", "Place after this sibling bean")
		orderCmd.Flags().StringVar(&orderBefore, "before", "", "Place before this sibling bean")
		orderCmd.Flags().BoolVar(&orderFirst, "first", false, "Place before all siblings")
		orderCmd.Flags().BoolVar(&orderLast, "last", false, "Place after all siblings")
		orderCmd.Flags().BoolVar(&orderJSON, "json", false, "Output as JSON")
		orderCmd.MarkFlagsMutuallyExclusive("after", "before", "first", "last")
	}
	root.AddCommand(orderCmd)
}
