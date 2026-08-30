package commands

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/hmans/beans/internal/ui"
	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beangraph"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

//go:embed roadmap.tmpl
var roadmapTemplateContent string

var (
	roadmapJSON        bool
	roadmapIncludeDone bool
	roadmapStatus      []string
	roadmapNoStatus    []string
	roadmapNoLinks     bool
	roadmapLinkPrefix  string
	roadmapDepth       int
	roadmapTags        bool
)

// roadmapData holds the structured roadmap for JSON output.
type roadmapData struct {
	// Root is set instead of Milestones/Unscheduled when the roadmap is
	// scoped to a single epic or feature (buildScopedRoadmap). A
	// milestone-scoped roadmap reuses Milestones with a single entry
	// instead, since that requires no new rendering path.
	Root        *rootGroup        `json:"root,omitempty"`
	Milestones  []milestoneGroup  `json:"milestones"`
	Unscheduled *unscheduledGroup `json:"unscheduled,omitempty"`
}

// rootGroup holds the scoped roadmap when rooted at an epic or feature.
// Exactly one of Epic/Feature is set.
type rootGroup struct {
	Epic    *epicGroup    `json:"epic,omitempty"`
	Feature *featureGroup `json:"feature,omitempty"`
}

// unscheduledGroup represents items not assigned to any milestone.
type unscheduledGroup struct {
	Epics    []epicGroup    `json:"epics,omitempty"`
	Features []featureGroup `json:"features,omitempty"`
	Other    []*bean.Bean   `json:"other,omitempty"`
}

// milestoneGroup represents a rank-1 container and its contents. The JSON
// keys are slot names, not type names: a configured rank-2 type appears
// under "epics" and a rank-3 type under "features", whatever they are
// called.
type milestoneGroup struct {
	Milestone *bean.Bean     `json:"milestone"`
	Epics     []epicGroup    `json:"epics,omitempty"`
	Features  []featureGroup `json:"features,omitempty"`
	Other     []*bean.Bean   `json:"other,omitempty"`
}

// epicGroup represents a rank-2 container and its child items. The "epic"
// JSON key is a slot name, not a type name: it holds whatever type is
// configured at rank 2.
type epicGroup struct {
	Epic     *bean.Bean     `json:"epic"`
	Items    []*bean.Bean   `json:"items,omitempty"`
	Features []featureGroup `json:"features,omitempty"`
}

// featureGroup represents a rank-3 container and the leaf items found
// anywhere beneath it (leafs below nested rank-3 containers are flattened
// into this list). The "feature" JSON key is a slot name, not a type name:
// it holds whatever type is configured at rank 3.
type featureGroup struct {
	Feature *bean.Bean   `json:"feature"`
	Items   []*bean.Bean `json:"items,omitempty"`
}

var roadmapCmd = &cobra.Command{
	Use:   "roadmap [id]",
	Short: "Generate a Markdown roadmap from milestones and epics",
	Long: `Displays a roadmap of milestones, epics, and their child items.

With no argument, renders the entire roadmap. With an ID argument (a milestone,
epic, or feature), scopes the output to that item's subtree only.

The --status and --no-status flags cannot be combined with an ID argument.

--depth limits how many levels below the roadmap root are rendered, following
the tree -L n convention: the root itself never counts. Without an ID argument the root is
the roadmap as a whole, so --depth 1 lists milestones only. With an ID
argument the root is that item, so --depth 1 lists its direct children.

--tags renders each item's tags on a line of their own beneath its title.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Query all beans via GraphQL resolver
		resolver := &beangraph.CoreResolver{Core: core}
		allBeans, err := resolver.Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		if err := validateRoadmapDepth(roadmapDepth, cmd.Flags().Changed("depth")); err != nil {
			return err
		}

		// Build the roadmap
		var data *roadmapData
		scoped := len(args) == 1
		if scoped {
			if len(roadmapStatus) > 0 || len(roadmapNoStatus) > 0 {
				return fmt.Errorf("--status/--no-status cannot be combined with a roadmap root ID")
			}
			root, err := core.Get(args[0])
			if err != nil {
				return fmt.Errorf("unknown bean: %s", args[0])
			}
			if err := validateRoadmapRootType(root); err != nil {
				return err
			}
			data = buildScopedRoadmap(allBeans, roadmapIncludeDone, root)
		} else {
			data = buildRoadmap(allBeans, roadmapIncludeDone, roadmapStatus, roadmapNoStatus)
		}
		pruneRoadmapDepth(data, roadmapDepth, scoped)

		// JSON output
		if roadmapJSON {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(data)
		}

		// TTY-aware rendering: gerendert am Terminal, Markdown bei Pipe/Redirect
		// (D02). The width is only queried when stdout is a terminal (EARS-6) --
		// a non-TTY caller never pays for term.GetSize, and its result would be
		// discarded by roadmapOutput's markdown branch anyway.
		links := !roadmapNoLinks
		linkPrefix := roadmapLinkPrefix
		if links && linkPrefix == "" {
			// Default: relative path from cwd to .beans directory
			linkPrefix = defaultLinkPrefix()
		}

		isTTY := term.IsTerminal(int(os.Stdout.Fd()))
		cols := 0
		if isTTY {
			if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
				cols = w
			}
		}

		fmt.Print(roadmapOutput(data, isTTY, cols, links, linkPrefix, roadmapTags))
		return nil
	},
}

// roadmapOutput is the testable TTY switch (EARS-1/EARS-2/EARS-5): TTY gets
// the plain-text tree via renderRoadmapPretty, everything else (pipe,
// redirect, non-terminal) gets renderRoadmapMarkdown unchanged -- byte-
// identical to calling renderRoadmapMarkdown directly (Q07/D02). cols is
// clamped via roadmapClampWidth regardless of what the caller passed in; a
// caller that could not determine a terminal width passes 0, which lands on
// the 80-column floor (D08).
func roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string, showTags bool) string {
	if isTTY {
		return renderRoadmapPretty(data, roadmapClampWidth(cols), showTags)
	}
	return renderRoadmapMarkdown(data, links, linkPrefix, showTags)
}

// buildRoadmap constructs the roadmap data structure from beans.
func buildRoadmap(allBeans []*bean.Bean, includeDone bool, statusFilter, noStatusFilter []string) *roadmapData {
	// Index all beans by ID for lookups
	byID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		byID[b.ID] = b
	}

	children := childrenIndex(allBeans)
	hidden := hiddenSubtrees(allBeans, children)

	// Find milestones, applying status filters
	var milestones []*bean.Bean
	for _, b := range allBeans {
		if !isRank(b, 1) {
			continue
		}
		if hidden[b.ID] {
			continue
		}
		// Apply status filters to milestones
		if len(statusFilter) > 0 && !containsStatus(statusFilter, b.Status) {
			continue
		}
		if len(noStatusFilter) > 0 && containsStatus(noStatusFilter, b.Status) {
			continue
		}
		milestones = append(milestones, b)
	}

	// Sort milestones by status, then dependency, then manual order, then priority, then created date
	bean.SortRoadmapContainers(milestones, cfg.StatusNames(), cfg.PriorityNames())

	// Build milestone groups
	var milestoneGroups []milestoneGroup
	for _, m := range milestones {
		group := buildMilestoneGroup(m, children, includeDone, hidden)
		// Only include milestones that have visible content
		if len(group.Epics) > 0 || len(group.Features) > 0 || len(group.Other) > 0 {
			milestoneGroups = append(milestoneGroups, group)
		}
	}

	// Build unscheduled group: items not under any milestone
	// Track which beans are under a milestone (directly or via epic)
	underMilestone := make(map[string]bool)
	for _, m := range milestones {
		underMilestone[m.ID] = true
		for _, child := range children[m.ID] {
			underMilestone[child.ID] = true
			// Also mark children of epics under this milestone
			if isRank(child, 2) {
				for _, epicChild := range children[child.ID] {
					underMilestone[epicChild.ID] = true
				}
			}
		}
	}

	// Find unscheduled epics (epics not under a milestone)
	var unscheduledEpics []epicGroup
	for _, b := range allBeans {
		if !isRank(b, 2) {
			continue
		}
		if hidden[b.ID] {
			continue
		}
		if underMilestone[b.ID] {
			continue
		}
		eg := buildEpicGroup(b, children, includeDone, hidden)
		if len(eg.Items) > 0 || len(eg.Features) > 0 {
			unscheduledEpics = append(unscheduledEpics, eg)
		}
	}

	// Sort unscheduled epics
	sortEpicGroups(unscheduledEpics, cfg.StatusNames(), cfg.PriorityNames())

	// Find unscheduled features: feature-typed beans that are not under a
	// milestone or epic (orphan features, e.g. created without --parent).
	var unscheduledFeatures []featureGroup
	for _, b := range allBeans {
		if !isRank(b, 3) {
			continue
		}
		if hidden[b.ID] {
			continue
		}
		if underMilestone[b.ID] {
			continue
		}
		// Skip features that are themselves children of an unscheduled epic
		// or of another feature above -- those are already rendered as part
		// of that ancestor's Features list / flattened into its Items via
		// collectLeafDescendants. (feature-under-feature is rejected by
		// ValidateParent via the CLI, but beans are hand-editable markdown --
		// this guard keeps hand-edited data from double-rendering.)
		if b.Parent != "" {
			if parent, ok := byID[b.Parent]; ok && (isRank(parent, 2) || isRank(parent, 3)) {
				continue
			}
		}
		fg := buildFeatureGroup(b, children, includeDone, hidden)
		if len(fg.Items) > 0 {
			unscheduledFeatures = append(unscheduledFeatures, fg)
		}
	}
	sortFeatureGroups(unscheduledFeatures, cfg.StatusNames(), cfg.PriorityNames())

	// Find orphan items (not milestone, not epic, no parent or parent is not milestone/epic)
	var orphanItems []*bean.Bean
	for _, b := range allBeans {
		// A hidden container's whole subtree stays out of the roadmap --
		// including here, where its orphaned children would otherwise
		// resurface as unscheduled items.
		if hidden[b.ID] {
			continue
		}
		// Skip milestones and epics -- always containers, never flat leaves.
		if isRank(b, 1) {
			continue
		}
		if isRank(b, 2) {
			// Epics with >=1 leaf descendant or feature are rendered via the
			// unscheduledEpics loop above as an epicGroup; skip them
			// here to avoid double-rendering. Childless epics (beans-36fa)
			// are not containers -- fall through and treat them as a flat
			// leaf like any other orphan item below.
			if eg, _ := classifyEpicChild(b, children, includeDone, hidden); eg != nil {
				continue
			}
		}
		if isRank(b, 3) {
			// Features with >=1 leaf descendant are rendered via the
			// unscheduledFeatures loop above as a featureGroup; skip them
			// here to avoid double-rendering. Childless features (D01,
			// beans-n8zw) are not containers -- fall through and treat
			// them as a flat leaf like any other orphan item below.
			if fg, _ := classifyFeatureChild(b, children, includeDone, hidden); fg != nil {
				continue
			}
		}
		// Skip if already under a milestone
		if underMilestone[b.ID] {
			continue
		}
		// Skip if has a parent (it's under an unscheduled epic, handled above)
		if b.Parent != "" {
			continue
		}
		// Apply done filter
		if !includeDone && cfg.IsArchiveStatus(b.Status) {
			continue
		}
		orphanItems = append(orphanItems, b)
	}

	// Sort orphan items
	bean.SortRoadmapLeaves(orphanItems, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

	// Build unscheduled group if there's content
	var unscheduled *unscheduledGroup
	if len(unscheduledEpics) > 0 || len(unscheduledFeatures) > 0 || len(orphanItems) > 0 {
		unscheduled = &unscheduledGroup{
			Epics:    unscheduledEpics,
			Features: unscheduledFeatures,
			Other:    orphanItems,
		}
	}

	return &roadmapData{
		Milestones:  milestoneGroups,
		Unscheduled: unscheduled,
	}
}

// isRank reports whether a bean sits on the given hierarchy rank.
func isRank(b *bean.Bean, rank int) bool {
	return cfg.RankOf(b.Type) == rank
}

// isContainerRank reports whether a bean sits on one of the three container
// ranks. Leaves (rank 4) are rendered inside a container, never as one.
func isContainerRank(b *bean.Bean) bool {
	r := cfg.RankOf(b.Type)
	return r >= 1 && r <= 3
}

// childrenIndex maps each bean ID to the beans that have it as a parent.
func childrenIndex(allBeans []*bean.Bean) map[string][]*bean.Bean {
	children := make(map[string][]*bean.Bean)
	for _, b := range allBeans {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b)
		}
	}
	return children
}

// markSubtree records id and every descendant of it in seen.
func markSubtree(id string, children map[string][]*bean.Bean, seen map[string]bool) {
	if seen[id] {
		return
	}
	seen[id] = true
	for _, child := range children[id] {
		markSubtree(child.ID, children, seen)
	}
}

// hiddenSubtrees returns the set of bean IDs that must vanish from an
// aggregate view (D15): every container-ranked bean whose type opted out
// via cfg.IsRoadmapType, plus everything beneath it, at any depth --
// markSubtree walks every descendant regardless of its own type, so a
// visible-typed epic under a hidden milestone is hidden too.
func hiddenSubtrees(allBeans []*bean.Bean, children map[string][]*bean.Bean) map[string]bool {
	hidden := make(map[string]bool)
	for _, b := range allBeans {
		if isContainerRank(b) && !cfg.IsRoadmapType(b.Type) {
			markSubtree(b.ID, children, hidden)
		}
	}
	return hidden
}

// hiddenSubtreesWithinScope is hiddenSubtrees restricted to a scoped-by-ID
// root: only a hidden container that is a strict descendant of root can
// seed hiding. This is deliberately narrower than "every bean except root":
// an ancestor of root -- however deeply hidden -- is never a strict
// descendant of root, so it can never seed hiding here, even though it
// would have seeded it (and marked root's whole subtree) in an
// allBeans-wide scan. root itself is also never a seed, even if its own
// type opted out -- naming a container by ID is a direct lookup, not the
// aggregate view the visibility flag governs, and the bypass is for
// exactly that named node. A hidden container nested anywhere beneath root
// still seeds normally and takes its own subtree down with it, exactly as
// in the unscoped roadmap.
func hiddenSubtreesWithinScope(allBeans []*bean.Bean, children map[string][]*bean.Bean, rootID string) map[string]bool {
	descendants := make(map[string]bool)
	markSubtree(rootID, children, descendants)

	hidden := make(map[string]bool)
	for _, b := range allBeans {
		if b.ID == rootID || !descendants[b.ID] {
			continue
		}
		if isContainerRank(b) && !cfg.IsRoadmapType(b.Type) {
			markSubtree(b.ID, children, hidden)
		}
	}
	return hidden
}

// buildScopedRoadmap builds a roadmapData scoped to a single milestone,
// epic, or feature root. Callers must have already validated root's type
// via validateRoadmapRootType; any other type panics, since that would be a
// caller bug, not user input.
//
// The roadmap-visibility flag (cfg.IsRoadmapType, D15) governs the aggregate
// views -- the unscoped roadmap and beans milestones -- but a direct-by-ID
// lookup of a container the user named here bypasses hiding for the named
// root only: hiddenSubtreesWithinScope seeds hiding only from root's strict
// descendants, so no ancestor of root (hidden or not) can mark anything
// inside the scope, while a hidden container nested anywhere beneath root
// still seeds normally and is suppressed exactly as in the unscoped
// roadmap.
func buildScopedRoadmap(allBeans []*bean.Bean, includeDone bool, root *bean.Bean) *roadmapData {
	children := childrenIndex(allBeans)
	hidden := hiddenSubtreesWithinScope(allBeans, children, root.ID)
	switch cfg.RankOf(root.Type) {
	case 1:
		group := buildMilestoneGroup(root, children, includeDone, hidden)
		return &roadmapData{Milestones: []milestoneGroup{group}}
	case 2:
		eg := buildEpicGroup(root, children, includeDone, hidden)
		return &roadmapData{Root: &rootGroup{Epic: &eg}}
	case 3:
		fg := buildFeatureGroup(root, children, includeDone, hidden)
		return &roadmapData{Root: &rootGroup{Feature: &fg}}
	default:
		// validateRoadmapRootType is checked by every caller before this
		// runs, so reaching a non-container rank here would be a caller
		// bug, not user input.
		panic("buildScopedRoadmap: unsupported root type " + root.Type)
	}
}

// validateRoadmapRootType returns an error if b does not sit on one of the
// three container ranks (rank 1 through 3).
func validateRoadmapRootType(b *bean.Bean) error {
	if isContainerRank(b) {
		return nil
	}
	var containers []string
	for rank := 1; rank <= 3; rank++ {
		containers = append(containers, cfg.TypesAtRank(rank)...)
	}
	if len(containers) == 0 {
		return fmt.Errorf("this project defines no container types (ranks 1-3), so %s (%s) cannot be a roadmap root",
			b.Type, b.ID)
	}
	return fmt.Errorf("roadmap root must be one of %s, got %s (%s)",
		strings.Join(containers, ", "), b.Type, b.ID)
}

// buildMilestoneGroup builds a milestone group with its epics and other
// items. hidden marks every bean whose whole subtree opted out of the
// aggregate views (D15): such a child is dropped outright, not folded into
// Other -- hiding removes, it does not reclassify. Pass nil to render
// unfiltered (buildScopedRoadmap's direct-by-ID lookup bypasses hiding).
func buildMilestoneGroup(m *bean.Bean, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool) milestoneGroup {
	group := milestoneGroup{Milestone: m}

	// Get direct children of this milestone, dropping any whose subtree is
	// hidden before they can be classified as an epic or folded into Other.
	var directChildren []*bean.Bean
	for _, child := range children[m.ID] {
		if hidden[child.ID] {
			continue
		}
		directChildren = append(directChildren, child)
	}

	// Separate epics from other items
	var epics []*bean.Bean
	var rest []*bean.Bean
	for _, child := range directChildren {
		if isRank(child, 2) {
			epics = append(epics, child)
		} else {
			rest = append(rest, child)
		}
	}

	// Build epic groups
	for _, epic := range epics {
		eg := buildEpicGroup(epic, children, includeDone, hidden)
		if len(eg.Items) > 0 || len(eg.Features) > 0 {
			group.Epics = append(group.Epics, eg)
		}
	}

	// Split the milestone's non-epic direct children into leaf items and
	// feature-typed children (which need their own recursive resolution).
	other, featureChildren := splitByContainerType(rest)

	for _, feature := range featureChildren {
		fg, leaf := classifyFeatureChild(feature, children, includeDone, hidden)
		if fg != nil {
			group.Features = append(group.Features, *fg)
		}
		if leaf != nil {
			// Childless feature (D01, beans-n8zw): not a container, render
			// as a flat leaf alongside the milestone's other direct items.
			other = append(other, leaf)
		}
	}

	// Filter the remaining flat "Other" items by done status.
	var filteredOther []*bean.Bean
	for _, child := range other {
		if includeDone || !cfg.IsArchiveStatus(child.Status) {
			filteredOther = append(filteredOther, child)
		}
	}

	// Sort epics and features
	sortEpicGroups(group.Epics, cfg.StatusNames(), cfg.PriorityNames())
	sortFeatureGroups(group.Features, cfg.StatusNames(), cfg.PriorityNames())

	// Sort other items
	bean.SortRoadmapLeaves(filteredOther, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())
	group.Other = filteredOther

	return group
}

// buildEpicGroup builds an epic group: its direct leaf children plus a
// recursively-resolved featureGroup for each direct feature child. hidden is
// as in buildMilestoneGroup: a hidden child's whole subtree is dropped, not
// folded into Items. Pass nil to render unfiltered (buildScopedRoadmap's
// direct-by-ID lookup bypasses hiding).
func buildEpicGroup(epic *bean.Bean, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool) epicGroup {
	var visibleChildren []*bean.Bean
	for _, child := range children[epic.ID] {
		if hidden[child.ID] {
			continue
		}
		visibleChildren = append(visibleChildren, child)
	}
	leafs, featureChildren := splitByContainerType(visibleChildren)

	eg := epicGroup{Epic: epic}
	for _, feature := range featureChildren {
		fg, leaf := classifyFeatureChild(feature, children, includeDone, hidden)
		if fg != nil {
			eg.Features = append(eg.Features, *fg)
		}
		if leaf != nil {
			// Childless feature (D01, beans-n8zw): not a container, render
			// as a flat leaf alongside the epic's other direct items.
			leafs = append(leafs, leaf)
		}
	}

	leafItems := filterChildren(leafs, includeDone)
	bean.SortRoadmapLeaves(leafItems, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())
	eg.Items = leafItems

	sortFeatureGroups(eg.Features, cfg.StatusNames(), cfg.PriorityNames())
	return eg
}

// classifyFeatureChild resolves a direct feature-typed child bean per D01
// (beans-n8zw): a feature is a container IFF it has >=1 leaf descendant
// (collectLeafDescendants, respecting includeDone). If it has descendants,
// the resolved featureGroup is returned for container rendering (existing
// behavior, unchanged). If it has none, the feature bean itself is returned
// as leaf so the caller can fold it into its own flat-leaf list -- and go
// through the exact same archive-status filtering every other leaf in that
// list goes through, instead of being silently dropped.
func classifyFeatureChild(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool) (fg *featureGroup, leaf *bean.Bean) {
	built := buildFeatureGroup(feature, children, includeDone, hidden)
	if len(built.Items) > 0 {
		return &built, nil
	}
	return nil, feature
}

// classifyEpicChild resolves a direct epic-typed child bean per beans-36fa:
// an epic is a container IFF it has >=1 leaf descendant or any feature.
// (buildEpicGroup, respecting includeDone). If it has descendants/features,
// the resolved epicGroup is kept for container rendering. If it has none,
// the epic bean itself is returned as leaf so the caller can fold it into
// its own flat-leaf list -- and go through the exact same archive-status
// filtering every other leaf in that list goes through, instead of being
// silently dropped.
func classifyEpicChild(epic *bean.Bean, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool) (eg *epicGroup, leaf *bean.Bean) {
	built := buildEpicGroup(epic, children, includeDone, hidden)
	if len(built.Items) > 0 || len(built.Features) > 0 {
		return &built, nil
	}
	return nil, epic
}


// buildFeatureGroup builds a feature group: all leaf descendants found
// anywhere beneath the feature, flattened and sorted. hidden is as in
// buildMilestoneGroup: a hidden descendant, and everything beneath it, is
// dropped rather than flattened into Items. Pass nil to render unfiltered
// (buildScopedRoadmap's direct-by-ID lookup bypasses hiding).
func buildFeatureGroup(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool) featureGroup {
	items := collectLeafDescendants(feature.ID, children, includeDone, hidden)
	bean.SortRoadmapLeaves(items, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())
	return featureGroup{Feature: feature, Items: items}
}

// filterChildren filters children based on done status.
func filterChildren(children []*bean.Bean, includeDone bool) []*bean.Bean {
	if includeDone {
		// Return a copy to avoid modifying the original
		result := make([]*bean.Bean, len(children))
		copy(result, children)
		return result
	}

	var filtered []*bean.Bean
	for _, b := range children {
		if !cfg.IsArchiveStatus(b.Status) {
			filtered = append(filtered, b)
		}
	}
	return filtered
}

// splitByContainerType separates a bean's direct children into leafs
// (anything that isn't a feature) and feature-typed children.
func splitByContainerType(beans []*bean.Bean) (leafs []*bean.Bean, features []*bean.Bean) {
	for _, b := range beans {
		if isRank(b, 3) {
			features = append(features, b)
		} else {
			leafs = append(leafs, b)
		}
	}
	return leafs, features
}

// collectLeafDescendants recursively walks everything below parentID and
// returns the leaf beans found at any depth, flattened. Feature-typed
// descendants are transparent containers: their own children are walked
// too, but the feature bean itself is never included in the result. hidden
// is as in buildMilestoneGroup: a hidden descendant, and everything beneath
// it, is dropped rather than flattened in. Pass nil to render unfiltered
// (buildScopedRoadmap's direct-by-ID lookup bypasses hiding).
// beans.yml's ValidateParent forbids feature-under-feature via the CLI, so
// this only recurses more than one level on hand-edited data -- the
// visited guard exists purely so a hand-authored parent cycle can't crash
// roadmap with a stack overflow (the old, non-recursive code was immune).
func collectLeafDescendants(parentID string, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool) []*bean.Bean {
	return collectLeafDescendantsVisited(parentID, children, includeDone, hidden, map[string]bool{})
}

func collectLeafDescendantsVisited(parentID string, children map[string][]*bean.Bean, includeDone bool, hidden map[string]bool, visited map[string]bool) []*bean.Bean {
	if visited[parentID] {
		return nil
	}
	visited[parentID] = true

	var leafs []*bean.Bean
	for _, child := range children[parentID] {
		if hidden[child.ID] {
			continue
		}
		if isRank(child, 3) {
			leafs = append(leafs, collectLeafDescendantsVisited(child.ID, children, includeDone, hidden, visited)...)
			continue
		}
		if !includeDone && cfg.IsArchiveStatus(child.Status) {
			continue
		}
		leafs = append(leafs, child)
	}
	return leafs
}

// containsStatus checks if a status is in the list.
func containsStatus(statuses []string, status string) bool {
	return slices.Contains(statuses, status)
}

// sortEpicGroups sorts epicGroups by their Epic bean, following the same
// status → dependency → order → priority → created_at chain as
// bean.SortRoadmapContainers.
func sortEpicGroups(groups []epicGroup, statusNames, priorityNames []string) {
	items := make([]*bean.Bean, len(groups))
	for i, g := range groups {
		items[i] = g.Epic
	}
	bean.SortRoadmapContainers(items, statusNames, priorityNames)
	rank := make(map[string]int, len(items))
	for i, b := range items {
		rank[b.ID] = i
	}
	sort.SliceStable(groups, func(i, j int) bool { return rank[groups[i].Epic.ID] < rank[groups[j].Epic.ID] })
}

// sortFeatureGroups sorts featureGroups by their Feature bean, following the
// same status → dependency → order → priority → created_at chain as
// bean.SortRoadmapContainers.
func sortFeatureGroups(groups []featureGroup, statusNames, priorityNames []string) {
	items := make([]*bean.Bean, len(groups))
	for i, g := range groups {
		items[i] = g.Feature
	}
	bean.SortRoadmapContainers(items, statusNames, priorityNames)
	rank := make(map[string]int, len(items))
	for i, b := range items {
		rank[b.ID] = i
	}
	sort.SliceStable(groups, func(i, j int) bool { return rank[groups[i].Feature.ID] < rank[groups[j].Feature.ID] })
}

// renderRoadmapMarkdown renders the roadmap as Markdown using the template.
func renderRoadmapMarkdown(data *roadmapData, links bool, linkPrefix string, showTags bool) string {
	// Create template with closures that capture link and tag settings
	tmpl := template.Must(
		template.New("roadmap").Funcs(template.FuncMap{
			"firstParagraph": firstParagraph,
			"typeBadge":      typeBadge,
			"beanRef": func(b *bean.Bean) string {
				return renderBeanRef(b, links, linkPrefix)
			},
			"tagLine": func(b *bean.Bean, indent string) string {
				return renderTagLine(b, indent, showTags)
			},
		}).Parse(roadmapTemplateContent),
	)

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		panic(err)
	}
	return sb.String()
}

// renderBeanRef renders a bean ID, optionally as a markdown link.
func renderBeanRef(b *bean.Bean, asLink bool, linkPrefix string) string {
	if !asLink {
		return "(" + b.ID + ")"
	}
	if linkPrefix == "" {
		return fmt.Sprintf("([%s](%s))", b.ID, b.Path)
	}
	// Ensure prefix ends with / for clean concatenation
	if !strings.HasSuffix(linkPrefix, "/") {
		linkPrefix += "/"
	}
	return fmt.Sprintf("([%s](%s%s))", b.ID, linkPrefix, b.Path)
}

// typeBadge returns a shields.io badge markdown for the bean type.
func typeBadge(b *bean.Bean) string {
	if b.Type == "" {
		return ""
	}
	// One colour source: the badge shows the same tone the terminal renders,
	// resolved through the active theme. shields.io wants the hex without the
	// leading hash.
	color := "gray"
	if t := cfg.GetType(b.Type); t != nil && t.Color != "" {
		color = strings.TrimPrefix(string(ui.ResolveColor(t.Color)), "#")
	}
	return fmt.Sprintf("![%s](https://img.shields.io/badge/%s-%s?style=flat-square)", b.Type, b.Type, color)
}

// renderTagLine renders a bean's tags as their own Markdown line, mirroring
// the TTY tag row: each tag as inline code, prefixed with "#". The returned
// string starts with a newline so a template can append it directly to the
// bean's line; indent keeps a leaf's tags inside its list item. Returns ""
// when tags are off or the bean has none.
func renderTagLine(b *bean.Bean, indent string, showTags bool) string {
	if !showTags || len(b.Tags) == 0 {
		return ""
	}
	tags := make([]string, len(b.Tags))
	for i, t := range b.Tags {
		tags[i] = "`#" + t + "`"
	}
	return "\n" + indent + strings.Join(tags, " ")
}

// defaultLinkPrefix returns the relative path from cwd to the .beans directory.
func defaultLinkPrefix() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(cwd, core.Root())
	if err != nil {
		return ""
	}
	// Convert to forward slashes for URL compatibility
	return filepath.ToSlash(rel)
}

// firstParagraph extracts the first paragraph from a body text.
func firstParagraph(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	// Find the first blank line (paragraph separator)
	lines := strings.Split(body, "\n")
	var para []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			break
		}
		// Skip markdown headers
		if strings.HasPrefix(line, "#") {
			continue
		}
		para = append(para, strings.TrimSpace(line))
	}

	result := strings.Join(para, " ")
	// Truncate if too long
	if len(result) > 200 {
		result = result[:197] + "..."
	}
	return result
}

func RegisterRoadmapCmd(root *cobra.Command) {
	roadmapCmd.Flags().BoolVar(&roadmapJSON, "json", false, "Output as JSON")
	roadmapCmd.Flags().BoolVar(&roadmapIncludeDone, "include-done", false, "Include completed items")
	roadmapCmd.Flags().StringArrayVar(&roadmapStatus, "status", nil, "Filter milestones by status (can be repeated)")
	roadmapCmd.Flags().StringArrayVar(&roadmapNoStatus, "no-status", nil, "Exclude milestones by status (can be repeated)")
	roadmapCmd.Flags().BoolVar(&roadmapNoLinks, "no-links", false, "Don't render bean IDs as markdown links")
	roadmapCmd.Flags().StringVar(&roadmapLinkPrefix, "link-prefix", "", "URL prefix for links")
	roadmapCmd.Flags().IntVar(&roadmapDepth, "depth", 0, "Limit output to n levels below the roadmap root (default: no limit)")
	roadmapCmd.Flags().BoolVar(&roadmapTags, "tags", false, "Render each item's tags")
	root.AddCommand(roadmapCmd)
}
