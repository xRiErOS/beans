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
)

// roadmapData holds the structured roadmap for JSON output.
type roadmapData struct {
	Milestones  []milestoneGroup  `json:"milestones"`
	Unscheduled *unscheduledGroup `json:"unscheduled,omitempty"`
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
	Use:   "roadmap",
	Short: "Generate a Markdown roadmap from milestones and epics",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Query all beans via GraphQL resolver
		resolver := &beangraph.CoreResolver{Core: core}
		allBeans, err := resolver.Beans(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("querying beans: %w", err)
		}

		// Build the roadmap
		data := buildRoadmap(allBeans, roadmapIncludeDone, roadmapStatus, roadmapNoStatus)

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

		fmt.Print(roadmapOutput(data, isTTY, cols, links, linkPrefix))
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
func roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string) string {
	if isTTY {
		return renderRoadmapPretty(data, roadmapClampWidth(cols))
	}
	return renderRoadmapMarkdown(data, links, linkPrefix)
}

// buildRoadmap constructs the roadmap data structure from beans.
func buildRoadmap(allBeans []*bean.Bean, includeDone bool, statusFilter, noStatusFilter []string) *roadmapData {
	// Index all beans by ID for lookups
	byID := make(map[string]*bean.Bean)
	for _, b := range allBeans {
		byID[b.ID] = b
	}

	// Build children index: parent ID -> children
	// This maps each bean ID to the beans that have it as a parent
	children := make(map[string][]*bean.Bean)
	for _, b := range allBeans {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b)
		}
	}

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

	// Sort milestones by status order, then by created date
	sortByStatusThenCreated(milestones, cfg)

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

	// Sort unscheduled epics by title
	sort.Slice(unscheduledEpics, func(i, j int) bool {
		return unscheduledEpics[i].Epic.Title < unscheduledEpics[j].Epic.Title
	})

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
	sort.Slice(unscheduledFeatures, func(i, j int) bool {
		return unscheduledFeatures[i].Feature.Title < unscheduledFeatures[j].Feature.Title
	})

	// Find orphan items (not milestone, not epic, no parent or parent is not milestone/epic)
	var orphanItems []*bean.Bean
	for _, b := range allBeans {
		// Skip milestones and epics -- always containers, never flat leaves.
		if b.Type == "milestone" || b.Type == "epic" {
			continue
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
	sortByTypeThenStatus(orphanItems, cfg)

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

	// Sort epics and features by their title
	sort.Slice(group.Epics, func(i, j int) bool {
		return group.Epics[i].Epic.Title < group.Epics[j].Epic.Title
	})
	sort.Slice(group.Features, func(i, j int) bool {
		return group.Features[i].Feature.Title < group.Features[j].Feature.Title
	})

	// Sort other items
	sortByTypeThenStatus(filteredOther, cfg)
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
	sortByTypeThenStatus(leafItems, cfg)
	eg.Items = leafItems

	sort.Slice(eg.Features, func(i, j int) bool {
		return eg.Features[i].Feature.Title < eg.Features[j].Feature.Title
	})
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

// buildFeatureGroup builds a feature group: all leaf descendants found
// anywhere beneath the feature, flattened and sorted.
func buildFeatureGroup(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool) featureGroup {
	items := collectLeafDescendants(feature.ID, children, includeDone)
	sortByTypeThenStatus(items, cfg)
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

// sortByStatusThenCreated sorts beans by status order, then by created date.
func sortByStatusThenCreated(beans []*bean.Bean, cfg interface{ StatusNames() []string }) {
	statusOrder := make(map[string]int)
	for i, s := range cfg.StatusNames() {
		statusOrder[s] = i
	}

	sort.Slice(beans, func(i, j int) bool {
		oi, oj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
		if oi != oj {
			return oi < oj
		}
		// Then by created date (oldest first for milestones)
		if beans[i].CreatedAt != nil && beans[j].CreatedAt != nil {
			return beans[i].CreatedAt.Before(*beans[j].CreatedAt)
		}
		return beans[i].ID < beans[j].ID
	})
}

// sortByTypeThenStatus sorts beans by type order, then status order, then by ID.
func sortByTypeThenStatus(beans []*bean.Bean, cfg interface {
	StatusNames() []string
	TypeNames() []string
}) {
	statusOrder := make(map[string]int)
	for i, s := range cfg.StatusNames() {
		statusOrder[s] = i
	}
	typeOrder := make(map[string]int)
	for i, t := range cfg.TypeNames() {
		typeOrder[t] = i
	}

	sort.Slice(beans, func(i, j int) bool {
		// First by type
		ti, tj := typeOrder[beans[i].Type], typeOrder[beans[j].Type]
		if ti != tj {
			return ti < tj
		}
		// Then by status
		si, sj := statusOrder[beans[i].Status], statusOrder[beans[j].Status]
		if si != sj {
			return si < sj
		}
		return beans[i].ID < beans[j].ID
	})
}

// renderRoadmapMarkdown renders the roadmap as Markdown using the template.
func renderRoadmapMarkdown(data *roadmapData, links bool, linkPrefix string) string {
	// Create template with closures that capture link settings
	tmpl := template.Must(
		template.New("roadmap").Funcs(template.FuncMap{
			"firstParagraph": firstParagraph,
			"typeBadge":      typeBadge,
			"beanRef": func(b *bean.Bean) string {
				return renderBeanRef(b, links, linkPrefix)
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
	root.AddCommand(roadmapCmd)
}
