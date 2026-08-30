package commands

import (
	"context"
	"fmt"
	"sort"

	"github.com/hmans/beans/internal/output"
	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/hmans/beans/pkg/beangraph/model"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

var (
	listJSON        bool
	listSearch      string
	listStatus      []string
	listNoStatus    []string
	listType        []string
	listNoType      []string
	listPriority    []string
	listNoPriority  []string
	listTag         []string
	listNoTag       []string
	listHasParent   bool
	listNoParent    bool
	listParentID    string
	listHasBlocking bool
	listNoBlocking  bool
	listIsBlocked   bool
	listUnblocked   bool
	listReady       bool
	listQuiet       bool
	listSort        string
	listDesc        bool
	listFull        bool
	listWhere       []string
	listView        string
	listMaxWidth    int
	listTags        bool
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all beans",
	Long: `Lists all beans in the .beans directory.

Search Syntax (--search/-S):
  The search flag supports Bleve query string syntax:

  login          Exact term match
  login~         Fuzzy match (1 edit distance, finds "loggin", "logins")
  login~2        Fuzzy match (2 edit distance)
  log*           Wildcard prefix match
  "user login"   Exact phrase match
  user AND login Both terms required
  user OR login  Either term matches
  slug:auth      Search only in slug field
  title:login    Search only in title field
  body:auth      Search only in body field`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Flag validation comes before any data access and before every
		// early return: an empty result set or --json must not swallow a
		// typo in --view (beans-cfky).
		form, ok := ui.ParseForm(listView)
		if !ok {
			return cmdError(listJSON, output.ErrValidation,
				"unknown --view %q: expected table or tree", listView)
		}
		// Validate --where up front (usage errors, reserved keys) before
		// running the query, analogous to validateExtraKeys in create.go/update.go.
		if err := validateWhereKeys(listWhere); err != nil {
			return err
		}

		// Build GraphQL filter from CLI flags
		filter := &model.BeanFilter{
			Status:          listStatus,
			ExcludeStatus:   listNoStatus,
			Type:            listType,
			ExcludeType:     listNoType,
			Priority:        listPriority,
			ExcludePriority: listNoPriority,
			Tags:            listTag,
			ExcludeTags:     listNoTag,
		}

		// Add search filter if provided
		if listSearch != "" {
			filter.Search = &listSearch
		}

		// Add parent/blocks filters
		if listHasParent {
			filter.HasParent = &listHasParent
		}
		if listNoParent {
			filter.NoParent = &listNoParent
		}
		if listParentID != "" {
			filter.ParentID = &listParentID
		}
		if listHasBlocking {
			filter.HasBlocking = &listHasBlocking
		}
		if listNoBlocking {
			filter.NoBlocking = &listNoBlocking
		}
		// --ready and --is-blocked are mutually exclusive
		if listReady && listIsBlocked {
			return fmt.Errorf("--ready and --is-blocked are mutually exclusive")
		}
		// --unblocked is the exact inverse of --is-blocked, so the two
		// can never be satisfied together.
		if listUnblocked && listIsBlocked {
			return fmt.Errorf("--unblocked and --is-blocked are mutually exclusive")
		}

		if listIsBlocked {
			filter.IsBlocked = &listIsBlocked
		}

		// --unblocked keeps only beans with no unresolved blocker, direct or
		// inherited from an ancestor. Unlike --ready it says nothing about
		// status, so drafts and in-progress beans survive it. It is applied
		// after --is-blocked, which the check above has already ruled out,
		// and before --ready, which asks for the same value.
		if listUnblocked {
			unblocked := false
			filter.IsBlocked = &unblocked
		}

		// --ready: beans available to start (not blocked, excludes in-progress/completed/scrapped/draft,
		// and excludes beans with implicit terminal status from a scrapped/completed ancestor)
		if listReady {
			applyReadyFilter(filter)
		}

		// Execute query via core resolver
		resolver := &beangraph.CoreResolver{Core: core}
		beans, err := resolver.Beans(context.Background(), filter)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		// --where filters on extra front matter keys after the graph query,
		// since BeanFilter (a generated GraphQL type) does not carry them (AC1/AC2).
		beans = filterByWhere(beans, listWhere)

		// Sort beans
		sortBeans(beans, listSort, listDesc, cfg)

		// JSON output (flat list)
		if listJSON {
			if !listFull {
				for _, b := range beans {
					b.Body = ""
				}
			}
			return output.SuccessMultiple(beans)
		}

		// Quiet mode: just IDs (flat)
		if listQuiet {
			for _, b := range beans {
				fmt.Println(b.ID)
			}
			return nil
		}

		// Both forms need every bean: the tree form to resolve ancestors for
		// context, the table form to compute implicit statuses.
		allBeans, err := resolver.Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying all beans for tree: %w", err)
		}

		// Pre-compute implicit statuses for all beans
		implicitStatuses := make(map[string]string, len(allBeans))
		for _, b := range allBeans {
			if status, _ := core.ImplicitStatus(b.ID); status != "" {
				implicitStatuses[b.ID] = status
			}
		}

		// Create sort function for tree building
		sortFn := func(b []*bean.Bean) {
			sortBeans(b, listSort, listDesc, cfg)
		}

		// Build tree
		tree := ui.BuildTree(beans, allBeans, sortFn, implicitStatuses)

		if len(tree) == 0 {
			fmt.Println(ui.Muted.Render("No beans found. Create one with: beans new <title>"))
			return nil
		}

		width := resolveWidth(listMaxWidth, cmd.Flags().Changed("max-width"), cfg)

		// The row source differs by form, not just by what Render does with
		// it afterwards: BuildTree deliberately widens `tree` beyond `beans`
		// with unmatched ancestors for context (ui.TreeNode.Matched), which
		// is correct for --view tree but wrong for --view table — a filtered
		// table promising comparable peers must not grow rows the filter
		// rejected. RowsFromFlatItems(FlattenTree(tree)) would carry that
		// leak through even though Render's table path re-flattens depth to
		// 0, because flattening depth does not drop the extra beans. FlatRows
		// on the already-filtered `beans` avoids both the leak and computing
		// tree depth Render would then discard.
		var rows []ui.Row
		if form == ui.FormTree {
			rows = ui.RowsFromFlatItems(ui.FlattenTree(tree))
		} else {
			rows = ui.FlatRows(beans)
		}
		fmt.Print(ui.Render(rows, form, "Beans", width, listTags, cfg))
		return nil
	},
}

// applyReadyFilter mutates filter in place to express "--ready": not
// blocked, excludes in-progress/completed/scrapped/draft, and excludes
// beans with implicit terminal status from a scrapped/completed ancestor.
// It mutates an already partially-built *model.BeanFilter (e.g. one that
// already carries --type/--status flags) rather than constructing a fresh
// one, so callers keep whatever filters they set beforehand.
func applyReadyFilter(filter *model.BeanFilter) {
	isBlocked := false
	excludeImplicitTerminal := true
	filter.IsBlocked = &isBlocked
	filter.ExcludeStatus = append(filter.ExcludeStatus, "in-progress", "completed", "scrapped", "draft")
	filter.ExcludeImplicitTerminal = &excludeImplicitTerminal
}

// sortBeans orders beans in place. desc reverses the finished ordering, so
// every sort field gains a direction without each branch having to know
// about one. With sortBy == "order" the reversal also flips the parent
// groups the order branch builds, not just the beans inside them.
func sortBeans(beans []*bean.Bean, sortBy string, desc bool, cfg *config.Config) {
	statusNames := cfg.StatusNames()
	priorityNames := cfg.PriorityNames()
	typeNames := cfg.TypeNames()

	switch sortBy {
	case "created":
		sort.Slice(beans, func(i, j int) bool {
			if beans[i].CreatedAt == nil && beans[j].CreatedAt == nil {
				return beans[i].ID < beans[j].ID
			}
			if beans[i].CreatedAt == nil {
				return false
			}
			if beans[j].CreatedAt == nil {
				return true
			}
			return beans[i].CreatedAt.After(*beans[j].CreatedAt)
		})
	case "updated":
		sort.Slice(beans, func(i, j int) bool {
			if beans[i].UpdatedAt == nil && beans[j].UpdatedAt == nil {
				return beans[i].ID < beans[j].ID
			}
			if beans[i].UpdatedAt == nil {
				return false
			}
			if beans[j].UpdatedAt == nil {
				return true
			}
			return beans[i].UpdatedAt.After(*beans[j].UpdatedAt)
		})
	case "status":
		// Build status order from configured statuses
		statusOrder := make(map[string]int)
		for i, s := range statusNames {
			statusOrder[s] = i
		}
		sort.Slice(beans, func(i, j int) bool {
			oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
			if oi != oj {
				return oi < oj
			}
			return beans[i].ID < beans[j].ID
		})
	case "priority":
		// Build priority order from configured priorities
		priorityOrder := make(map[string]int)
		for i, p := range priorityNames {
			priorityOrder[p] = i
		}
		// Find normal priority index for beans without priority
		normalIdx := len(priorityNames)
		for i, p := range priorityNames {
			if p == "normal" {
				normalIdx = i
				break
			}
		}
		sort.Slice(beans, func(i, j int) bool {
			pi := normalIdx
			if beans[i].Priority != "" {
				if order, ok := priorityOrder[beans[i].Priority]; ok {
					pi = order
				}
			}
			pj := normalIdx
			if beans[j].Priority != "" {
				if order, ok := priorityOrder[beans[j].Priority]; ok {
					pj = order
				}
			}
			if pi != pj {
				return pi < pj
			}
			return beans[i].ID < beans[j].ID
		})
	case "id":
		sort.Slice(beans, func(i, j int) bool {
			return beans[i].ID < beans[j].ID
		})
	case "order":
		// Order is a fractional index scoped per parent (R-12): two
		// siblings under different parents can carry the same Order value,
		// so sorting the flat list by Order alone would interleave
		// unrelated groups. Group beans by Parent first — groups keep the
		// order in which their first member appears in the input, for
		// determinism — then sort each group with bean.SortByOrder, which
		// places beans without an Order value after every sibling that has
		// one (AC1/AC2).
		groupOrder := make([]string, 0)
		groups := make(map[string][]*bean.Bean)
		for _, b := range beans {
			if _, ok := groups[b.Parent]; !ok {
				groupOrder = append(groupOrder, b.Parent)
			}
			groups[b.Parent] = append(groups[b.Parent], b)
		}
		result := make([]*bean.Bean, 0, len(beans))
		for _, p := range groupOrder {
			group := groups[p]
			bean.SortByOrder(group)
			result = append(result, group...)
		}
		copy(beans, result)
	default:
		// Default: sort by status order, then priority, then type order, then title (same as TUI)
		bean.SortByStatusPriorityAndType(beans, statusNames, priorityNames, typeNames)
	}

	if desc {
		for i, j := 0, len(beans)-1; i < j; i, j = i+1, j-1 {
			beans[i], beans[j] = beans[j], beans[i]
		}
	}
}

// validateWhereKeys checks every --where argument up front: each entry must
// carry "=" (a usage error otherwise, same shape as --set) and must not name
// a reserved schema field (AC3).
func validateWhereKeys(wheres []string) error {
	for _, w := range wheres {
		key, _, err := parseSetPair(w, "--where")
		if err != nil {
			return err
		}
		if err := checkReservedKey(key); err != nil {
			return err
		}
	}
	return nil
}

// filterByWhere returns the subset of beans whose Extra map satisfies every
// key=value pair in wheres (AND semantics, AC1/AC2). A key carried by no bean
// simply yields an empty result (AC4). Callers must run validateWhereKeys
// first; a malformed pair here is skipped rather than erroring, since
// filtering runs after the query and has no error path back to the caller.
func filterByWhere(beans []*bean.Bean, wheres []string) []*bean.Bean {
	if len(wheres) == 0 {
		return beans
	}

	type pair struct{ key, value string }
	pairs := make([]pair, 0, len(wheres))
	for _, w := range wheres {
		key, value, err := parseSetPair(w, "--where")
		if err != nil {
			continue
		}
		pairs = append(pairs, pair{key, value})
	}

	result := make([]*bean.Bean, 0, len(beans))
	for _, b := range beans {
		matches := true
		for _, p := range pairs {
			v, ok := b.Extra[p.key]
			if !ok || fmt.Sprintf("%v", v) != p.value {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, b)
		}
	}
	return result
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// RegisterListCmd adds the list command to root. Flag registration is
// idempotent (guarded by a Lookup check) because listCmd is a package-level
// singleton and tests that want a real cobra.Execute() (so --view's parsing
// and cmd.Flags().Changed("max-width") behave as they do for the CLI) need
// to register it into a throwaway root without panicking on a second flag
// definition — the same pattern RegisterOrderCmd uses.
func RegisterListCmd(root *cobra.Command) {
	if listCmd.Flags().Lookup("json") == nil {
		listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
		listCmd.Flags().StringVarP(&listSearch, "search", "S", "", "Full-text search in title and body")
		listCmd.Flags().StringArrayVarP(&listStatus, "status", "s", nil, "Filter by status (can be repeated)")
		listCmd.Flags().StringArrayVar(&listNoStatus, "no-status", nil, "Exclude by status (can be repeated)")
		listCmd.Flags().StringArrayVarP(&listType, "type", "t", nil, "Filter by type (can be repeated)")
		listCmd.Flags().StringArrayVar(&listNoType, "no-type", nil, "Exclude by type (can be repeated)")
		listCmd.Flags().StringArrayVarP(&listPriority, "priority", "p", nil, "Filter by priority (can be repeated)")
		listCmd.Flags().StringArrayVar(&listNoPriority, "no-priority", nil, "Exclude by priority (can be repeated)")
		listCmd.Flags().StringArrayVar(&listTag, "tag", nil, "Filter by tag (can be repeated, OR logic)")
		listCmd.Flags().StringArrayVar(&listNoTag, "no-tag", nil, "Exclude beans with tag (can be repeated)")
		listCmd.Flags().BoolVar(&listHasParent, "has-parent", false, "Filter beans with a parent")
		listCmd.Flags().BoolVar(&listNoParent, "no-parent", false, "Filter beans without a parent")
		listCmd.Flags().StringVar(&listParentID, "parent", "", "Filter by parent ID")
		listCmd.Flags().BoolVar(&listHasBlocking, "has-blocking", false, "Filter beans that are blocking others")
		listCmd.Flags().BoolVar(&listNoBlocking, "no-blocking", false, "Filter beans that aren't blocking others")
		listCmd.Flags().BoolVar(&listIsBlocked, "is-blocked", false, "Filter beans that are blocked by others")
		listCmd.Flags().BoolVar(&listUnblocked, "unblocked", false, "Filter beans with no unresolved blocker")
		listCmd.Flags().BoolVar(&listReady, "ready", false, "Filter beans available to start (not blocked, excludes in-progress/completed/scrapped/draft)")
		listCmd.Flags().BoolVarP(&listQuiet, "quiet", "q", false, "Only output IDs (one per line)")
		listCmd.Flags().StringVar(&listSort, "sort", "", "Sort by: created, updated, status, priority, id, order (order is scoped per parent) (default: status, priority, type, title)")
		listCmd.Flags().BoolVar(&listDesc, "desc", false, "Reverse the sort order")
		listCmd.Flags().BoolVar(&listFull, "full", false, "Include bean body in JSON output")
		listCmd.Flags().StringArrayVar(&listWhere, "where", nil, "Filter by extra front matter key=value (can be repeated, AND logic)")
		listCmd.Flags().StringVar(&listView, "view", "table",
			"Arrangement: table (flat, sortable) or tree (nested)")
		listCmd.Flags().IntVar(&listMaxWidth, "max-width", 0,
			"Cap the rendered width; 0 disables the cap (default: display.max_width, else 110)")
		listCmd.Flags().BoolVar(&listTags, "tags", false, "Render each bean's tags")
	}
	root.AddCommand(listCmd)
}
