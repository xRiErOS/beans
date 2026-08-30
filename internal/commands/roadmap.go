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
	"github.com/hmans/beans/pkg/config"
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
	roadmapView        string
	roadmapFormat      string
	roadmapWidthFlag   int
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

// milestoneGroup represents a milestone and its contents.
type milestoneGroup struct {
	Milestone *bean.Bean     `json:"milestone"`
	Epics     []epicGroup    `json:"epics,omitempty"`
	Features  []featureGroup `json:"features,omitempty"`
	Other     []*bean.Bean   `json:"other,omitempty"`
}

// epicGroup represents an epic and its child items.
type epicGroup struct {
	Epic     *bean.Bean     `json:"epic"`
	Items    []*bean.Bean   `json:"items,omitempty"`
	Features []featureGroup `json:"features,omitempty"`
}

// featureGroup represents a feature and the leaf items found anywhere
// beneath it (leafs below nested features are flattened into this list).
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

		form, ok := ui.ParseForm(roadmapView)
		if !ok {
			return fmt.Errorf("invalid --view %q: must be one of \"tree\", \"table\"", roadmapView)
		}

		var formatOverride roadmapFormatOverride
		switch roadmapFormat {
		case "":
			formatOverride = roadmapFormatAuto
		case "tty":
			formatOverride = roadmapFormatTTY
		case "markdown":
			formatOverride = roadmapFormatMarkdown
		default:
			return fmt.Errorf("invalid --format %q: must be one of \"tty\", \"markdown\"", roadmapFormat)
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
		cols := resolveWidth(roadmapWidthFlag, cmd.Flags().Changed("max-width"), cfg)

		fmt.Print(roadmapOutput(data, isTTY, formatOverride, cols, links, linkPrefix, roadmapTags, form, cfg))
		return nil
	},
}

// roadmapFormatOverride lets --format force which branch roadmapOutput takes,
// independent of isTTY detection. roadmapFormatAuto preserves today's
// detection-only behaviour exactly -- this step does not change what either
// branch renders, only how the branch is chosen.
type roadmapFormatOverride int

const (
	roadmapFormatAuto roadmapFormatOverride = iota
	roadmapFormatTTY
	roadmapFormatMarkdown
)

// roadmapOutput is the testable TTY switch (EARS-1/EARS-2/EARS-5): TTY gets
// the shared layout engine (ui.Render) in the requested form, everything else
// (pipe, redirect, non-terminal) gets renderRoadmapMarkdown unchanged --
// byte-identical to calling renderRoadmapMarkdown directly (Q07/D02). cols is
// clamped via roadmapClampWidth regardless of what the caller passed in; a
// caller that could not determine a terminal width passes 0, which lands on
// the 80-column floor (D08). format overrides the isTTY-derived choice when
// it is not roadmapFormatAuto, so a caller can force either branch (e.g. for
// tests, or a user explicitly asking for --format markdown/tty).
func roadmapOutput(data *roadmapData, isTTY bool, format roadmapFormatOverride, cols int, links bool, linkPrefix string, showTags bool, form ui.Form, cfg *config.Config) string {
	renderTTY := isTTY
	switch format {
	case roadmapFormatTTY:
		renderTTY = true
	case roadmapFormatMarkdown:
		renderTTY = false
	}
	if renderTTY {
		return ui.Render(roadmapRows(data), form, "Roadmap", roadmapClampWidth(cols), showTags, cfg)
	}
	return renderRoadmapMarkdown(data, links, linkPrefix, showTags)
}

// roadmapRows bridges the grouped roadmapData produced by buildRoadmap /
// buildScopedRoadmap into the flat row list ui.Render expects. It mirrors the
// walk renderRoadmapPretty performs (milestone -> epics -> features -> leafs,
// then epic/feature/other for the unscheduled bucket), but builds
// ui.FlatItems instead of writing text directly, and does no sorting of its
// own -- order comes entirely from the builder's slices.
//
// Depth counts tree levels, not the old renderer's indent-in-spaces: each
// milestone (or, for the unscheduled bucket, each of its top-level epics/
// features/other) is its own depth-0 root. ui.RowsFromFlatItems resets its
// ancestry stack whenever depth drops back to 0, so consecutive roots never
// bleed tree lines into each other -- which is also why a milestone's own
// IsLast is irrelevant to what gets drawn (Connector/Stem never consult
// AncestorsLast[0]) and is set to true throughout rather than tracked.
//
// Only the unscheduled bucket gets a Section heading ("No Milestone"): each
// milestone is itself a Bean row (type "milestone"), which is heading enough
// on its own, matching the one documented use of Row.Section (ui/columns.go).
func roadmapRows(data *roadmapData) []ui.Row {
	var items []ui.FlatItem

	if data.Root != nil {
		if data.Root.Epic != nil {
			appendRoadmapEpicGroup(&items, *data.Root.Epic, 0, true)
		}
		if data.Root.Feature != nil {
			appendRoadmapFeatureGroup(&items, *data.Root.Feature, 0, true)
		}
		return ui.RowsFromFlatItems(items)
	}

	for _, mg := range data.Milestones {
		appendRoadmapMilestoneGroup(&items, mg)
	}

	unscheduledAt := -1
	if data.Unscheduled != nil {
		unscheduledAt = len(items)
		appendRoadmapUnscheduledGroup(&items, *data.Unscheduled)
	}

	rows := ui.RowsFromFlatItems(items)
	if unscheduledAt >= 0 && unscheduledAt < len(rows) {
		rows[unscheduledAt].Section = "No Milestone"
	}
	return rows
}

// appendRoadmapMilestoneGroup appends a milestone (depth 0) followed by its
// epics, features, and other leaf items (depth 1), in that order -- matching
// renderRoadmapPretty's mg.Epics / mg.Features / mg.Other walk.
func appendRoadmapMilestoneGroup(items *[]ui.FlatItem, mg milestoneGroup) {
	*items = append(*items, ui.FlatItem{Bean: mg.Milestone, Depth: 0, IsLast: true})

	total := len(mg.Epics) + len(mg.Features) + len(mg.Other)
	i := 0
	for _, eg := range mg.Epics {
		i++
		appendRoadmapEpicGroup(items, eg, 1, i == total)
	}
	for _, fg := range mg.Features {
		i++
		appendRoadmapFeatureGroup(items, fg, 1, i == total)
	}
	for _, leaf := range mg.Other {
		i++
		*items = append(*items, ui.FlatItem{Bean: leaf, Depth: 1, IsLast: i == total})
	}
}

// appendRoadmapUnscheduledGroup appends the unscheduled bucket's epics,
// features, and other leaf items at depth 0 -- each is its own root, exactly
// like a milestone-less scoped root, since there is no milestone bean to
// hang them under.
func appendRoadmapUnscheduledGroup(items *[]ui.FlatItem, ug unscheduledGroup) {
	total := len(ug.Epics) + len(ug.Features) + len(ug.Other)
	i := 0
	for _, eg := range ug.Epics {
		i++
		appendRoadmapEpicGroup(items, eg, 0, i == total)
	}
	for _, fg := range ug.Features {
		i++
		appendRoadmapFeatureGroup(items, fg, 0, i == total)
	}
	for _, leaf := range ug.Other {
		i++
		*items = append(*items, ui.FlatItem{Bean: leaf, Depth: 0, IsLast: i == total})
	}
}

// appendRoadmapEpicGroup appends an epic at depth, followed by its direct
// leaf items and nested feature groups at depth+1 -- items before features,
// per renderRoadmapEpicGroup / roadmap.tmpl.
func appendRoadmapEpicGroup(items *[]ui.FlatItem, eg epicGroup, depth int, isLast bool) {
	*items = append(*items, ui.FlatItem{Bean: eg.Epic, Depth: depth, IsLast: isLast})

	childDepth := depth + 1
	total := len(eg.Items) + len(eg.Features)
	i := 0
	for _, leaf := range eg.Items {
		i++
		*items = append(*items, ui.FlatItem{Bean: leaf, Depth: childDepth, IsLast: i == total})
	}
	for _, fg := range eg.Features {
		i++
		appendRoadmapFeatureGroup(items, fg, childDepth, i == total)
	}
}

// appendRoadmapFeatureGroup appends a feature at depth, followed by its
// flattened leaf items at depth+1.
func appendRoadmapFeatureGroup(items *[]ui.FlatItem, fg featureGroup, depth int, isLast bool) {
	*items = append(*items, ui.FlatItem{Bean: fg.Feature, Depth: depth, IsLast: isLast})

	childDepth := depth + 1
	n := len(fg.Items)
	for i, leaf := range fg.Items {
		*items = append(*items, ui.FlatItem{Bean: leaf, Depth: childDepth, IsLast: i == n-1})
	}
}

// buildRoadmap constructs the roadmap data structure from beans.
func buildRoadmap(allBeans []*bean.Bean, includeDone bool, statusFilter, noStatusFilter []string) *roadmapData {
	// Index all beans by ID for lookups
	byID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		byID[b.ID] = b
	}

	children := childrenIndex(allBeans)

	// Find milestones, applying status filters
	var milestones []*bean.Bean
	for _, b := range allBeans {
		if b.Type != "milestone" {
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
		group := buildMilestoneGroup(m, children, includeDone)
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
			if child.Type == "epic" {
				for _, epicChild := range children[child.ID] {
					underMilestone[epicChild.ID] = true
				}
			}
		}
	}

	// Find unscheduled epics (epics not under a milestone)
	var unscheduledEpics []epicGroup
	for _, b := range allBeans {
		if b.Type != "epic" {
			continue
		}
		if underMilestone[b.ID] {
			continue
		}
		eg := buildEpicGroup(b, children, includeDone)
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
		if b.Type != "feature" {
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
			if parent, ok := byID[b.Parent]; ok && (parent.Type == "epic" || parent.Type == "feature") {
				continue
			}
		}
		fg := buildFeatureGroup(b, children, includeDone)
		if len(fg.Items) > 0 {
			unscheduledFeatures = append(unscheduledFeatures, fg)
		}
	}
	sortFeatureGroups(unscheduledFeatures, cfg.StatusNames(), cfg.PriorityNames())

	// Find orphan items (not milestone, not epic, no parent or parent is not milestone/epic)
	var orphanItems []*bean.Bean
	for _, b := range allBeans {
		// Skip milestones and epics -- always containers, never flat leaves.
		if b.Type == "milestone" {
			continue
		}
		if b.Type == "epic" {
			// Epics with >=1 leaf descendant or feature are rendered via the
			// unscheduledEpics loop above as an epicGroup; skip them
			// here to avoid double-rendering. Childless epics (beans-36fa)
			// are not containers -- fall through and treat them as a flat
			// leaf like any other orphan item below.
			if eg, _ := classifyEpicChild(b, children, includeDone); eg != nil {
				continue
			}
		}
		if b.Type == "feature" {
			// Features with >=1 leaf descendant are rendered via the
			// unscheduledFeatures loop above as a featureGroup; skip them
			// here to avoid double-rendering. Childless features (D01,
			// beans-n8zw) are not containers -- fall through and treat
			// them as a flat leaf like any other orphan item below.
			if fg, _ := classifyFeatureChild(b, children, includeDone); fg != nil {
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

// buildScopedRoadmap builds a roadmapData scoped to a single milestone,
// epic, or feature root. Callers must have already validated root's type
// via validateRoadmapRootType; any other type panics, since that would be a
// caller bug, not user input.
func buildScopedRoadmap(allBeans []*bean.Bean, includeDone bool, root *bean.Bean) *roadmapData {
	switch root.Type {
	case "milestone":
		data := buildRoadmap(allBeans, includeDone, nil, nil)
		for _, mg := range data.Milestones {
			if mg.Milestone.ID == root.ID {
				return &roadmapData{Milestones: []milestoneGroup{mg}}
			}
		}
		// buildRoadmap drops milestones with zero visible children -- the
		// root was still found and matched by type/ID, so render it as an
		// empty container rather than silently returning nothing.
		return &roadmapData{Milestones: []milestoneGroup{{Milestone: root}}}
	case "epic":
		eg := buildEpicGroup(root, childrenIndex(allBeans), includeDone)
		return &roadmapData{Root: &rootGroup{Epic: &eg}}
	case "feature":
		fg := buildFeatureGroup(root, childrenIndex(allBeans), includeDone)
		return &roadmapData{Root: &rootGroup{Feature: &fg}}
	default:
		panic("buildScopedRoadmap: unsupported root type " + root.Type)
	}
}

// validateRoadmapRootType returns an error if b is not a valid roadmap scope
// root (milestone, epic, or feature).
func validateRoadmapRootType(b *bean.Bean) error {
	switch b.Type {
	case "milestone", "epic", "feature":
		return nil
	default:
		return fmt.Errorf("roadmap root must be a milestone, epic, or feature, got %s (%s)", b.Type, b.ID)
	}
}

// buildMilestoneGroup builds a milestone group with its epics and other items.
func buildMilestoneGroup(m *bean.Bean, children map[string][]*bean.Bean, includeDone bool) milestoneGroup {
	group := milestoneGroup{Milestone: m}

	// Get direct children of this milestone
	directChildren := children[m.ID]

	// Separate epics from other items
	var epics []*bean.Bean
	var rest []*bean.Bean
	for _, child := range directChildren {
		if child.Type == "epic" {
			epics = append(epics, child)
		} else {
			rest = append(rest, child)
		}
	}

	// Build epic groups
	for _, epic := range epics {
		eg := buildEpicGroup(epic, children, includeDone)
		if len(eg.Items) > 0 || len(eg.Features) > 0 {
			group.Epics = append(group.Epics, eg)
		}
	}

	// Split the milestone's non-epic direct children into leaf items and
	// feature-typed children (which need their own recursive resolution).
	other, featureChildren := splitByContainerType(rest)

	for _, feature := range featureChildren {
		fg, leaf := classifyFeatureChild(feature, children, includeDone)
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
// recursively-resolved featureGroup for each direct feature child.
func buildEpicGroup(epic *bean.Bean, children map[string][]*bean.Bean, includeDone bool) epicGroup {
	leafs, featureChildren := splitByContainerType(children[epic.ID])

	eg := epicGroup{Epic: epic}
	for _, feature := range featureChildren {
		fg, leaf := classifyFeatureChild(feature, children, includeDone)
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
func classifyFeatureChild(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool) (fg *featureGroup, leaf *bean.Bean) {
	built := buildFeatureGroup(feature, children, includeDone)
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
func classifyEpicChild(epic *bean.Bean, children map[string][]*bean.Bean, includeDone bool) (eg *epicGroup, leaf *bean.Bean) {
	built := buildEpicGroup(epic, children, includeDone)
	if len(built.Items) > 0 || len(built.Features) > 0 {
		return &built, nil
	}
	return nil, epic
}

// buildFeatureGroup builds a feature group: all leaf descendants found
// anywhere beneath the feature, flattened and sorted.
func buildFeatureGroup(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool) featureGroup {
	items := collectLeafDescendants(feature.ID, children, includeDone)
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
		if b.Type == "feature" {
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
// too, but the feature bean itself is never included in the result.
// beans.yml's ValidateParent forbids feature-under-feature via the CLI, so
// this only recurses more than one level on hand-edited data -- the
// visited guard exists purely so a hand-authored parent cycle can't crash
// roadmap with a stack overflow (the old, non-recursive code was immune).
func collectLeafDescendants(parentID string, children map[string][]*bean.Bean, includeDone bool) []*bean.Bean {
	return collectLeafDescendantsVisited(parentID, children, includeDone, map[string]bool{})
}

func collectLeafDescendantsVisited(parentID string, children map[string][]*bean.Bean, includeDone bool, visited map[string]bool) []*bean.Bean {
	if visited[parentID] {
		return nil
	}
	visited[parentID] = true

	var leafs []*bean.Bean
	for _, child := range children[parentID] {
		if child.Type == "feature" {
			leafs = append(leafs, collectLeafDescendantsVisited(child.ID, children, includeDone, visited)...)
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
	// Map types to colors
	colors := map[string]string{
		"bug":       "d73a4a",
		"feature":   "0e8a16",
		"task":      "1d76db",
		"epic":      "5319e7",
		"milestone": "fbca04",
	}
	color := colors[b.Type]
	if color == "" {
		color = "gray"
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
	roadmapCmd.Flags().StringVar(&roadmapView, "view", "tree", `Layout: "tree" or "table"`)
	roadmapCmd.Flags().StringVar(&roadmapFormat, "format", "", `Output format: "tty" or "markdown" (default: detect)`)
	roadmapCmd.Flags().IntVar(&roadmapWidthFlag, "max-width", 0, "Cap rendering width in columns (0: no cap)")
	root.AddCommand(roadmapCmd)
}
