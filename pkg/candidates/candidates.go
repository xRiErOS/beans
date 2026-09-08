// Package candidates produces the option sets shown by the beans TUI's
// pickers (parent, blocking, status, type, priority, tag). It is
// independent of any rendering framework: every producer here returns
// plain data, never a bubbletea/lipgloss view.
package candidates

import (
	"context"
	"sort"
	"strings"

	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/beangraph"
	"github.com/xRiErOS/beans/pkg/config"
)

// ParentCandidates returns the beans eligible to become the parent of the
// beans identified by beanIDs (whose types are beanTypes), sorted by type
// order then title (case-insensitive). A bean is eligible only if its type
// is a valid parent type for every selected bean, and it is neither one of
// the selected beans nor a descendant of any of them (which would create a
// cycle).
func ParentCandidates(ctx context.Context, resolver *beangraph.CoreResolver, cfg *config.Config, beanIDs, beanTypes []string) ([]*bean.Bean, error) {
	var validParentTypes []string
	for i, beanType := range beanTypes {
		typeParents := beancore.ValidParentTypes(cfg, beanType)
		if i == 0 {
			validParentTypes = typeParents
		} else {
			validParentTypes = intersectStrings(validParentTypes, typeParents)
		}
	}

	allBeans, err := resolver.Beans(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Collect all descendants of all selected beans (to prevent cycles).
	allDescendants := make(map[string]bool)
	for _, beanID := range beanIDs {
		for descID := range collectDescendants(beanID, allBeans) {
			allDescendants[descID] = true
		}
	}

	selectedSet := make(map[string]bool)
	for _, id := range beanIDs {
		selectedSet[id] = true
	}

	// Filter to eligible parents:
	// 1. Must be of a valid parent type for ALL selected beans
	// 2. Must not be any of the selected beans
	// 3. Must not be a descendant of any selected bean (to prevent cycles)
	var eligibleBeans []*bean.Bean
	for _, b := range allBeans {
		if selectedSet[b.ID] {
			continue
		}
		if allDescendants[b.ID] {
			continue
		}
		isValidType := false
		for _, validType := range validParentTypes {
			if b.Type == validType {
				isValidType = true
				break
			}
		}
		if !isValidType {
			continue
		}
		eligibleBeans = append(eligibleBeans, b)
	}

	sortByTypeThenTitle(eligibleBeans, cfg)
	return eligibleBeans, nil
}

// BlockingCandidates returns every bean other than beanID, sorted by type
// order then title (case-insensitive). Unlike ParentCandidates, it does
// not exclude descendants of beanID: a bean may block or be blocked by its
// own descendants.
func BlockingCandidates(ctx context.Context, resolver *beangraph.CoreResolver, cfg *config.Config, beanID string) ([]*bean.Bean, error) {
	allBeans, err := resolver.Beans(ctx, nil)
	if err != nil {
		return nil, err
	}

	var eligibleBeans []*bean.Bean
	for _, b := range allBeans {
		if b.ID != beanID {
			eligibleBeans = append(eligibleBeans, b)
		}
	}

	sortByTypeThenTitle(eligibleBeans, cfg)
	return eligibleBeans, nil
}

// sortByTypeThenTitle sorts beans by cfg's type order, then by
// case-insensitive title.
func sortByTypeThenTitle(beans []*bean.Bean, cfg *config.Config) {
	typeNames := cfg.TypeNames()
	typeOrder := make(map[string]int)
	for i, t := range typeNames {
		typeOrder[t] = i
	}
	sort.Slice(beans, func(i, j int) bool {
		ti, tj := typeOrder[beans[i].Type], typeOrder[beans[j].Type]
		if ti != tj {
			return ti < tj
		}
		return strings.ToLower(beans[i].Title) < strings.ToLower(beans[j].Title)
	})
}

// StatusCandidates returns the available bean statuses.
func StatusCandidates() []config.StatusConfig {
	return config.DefaultStatuses
}

// TypeCandidates returns the available bean types.
func TypeCandidates() []config.TypeConfig {
	return config.DefaultTypes
}

// PriorityCandidates returns the available bean priorities.
func PriorityCandidates() []config.PriorityConfig {
	return config.DefaultPriorities
}

// TagCount holds a tag and its usage count across all beans.
type TagCount struct {
	Tag   string
	Count int
}

// TagCandidates returns every tag in use across all beans with its usage
// count, sorted by count descending, then by tag ascending.
func TagCandidates(ctx context.Context, resolver *beangraph.CoreResolver) ([]TagCount, error) {
	beans, err := resolver.Beans(ctx, nil)
	if err != nil {
		return nil, err
	}

	tagCounts := make(map[string]int)
	for _, b := range beans {
		for _, tag := range b.Tags {
			tagCounts[tag]++
		}
	}

	tags := make([]TagCount, 0, len(tagCounts))
	for tag, count := range tagCounts {
		tags = append(tags, TagCount{Tag: tag, Count: count})
	}

	sort.Slice(tags, func(i, j int) bool {
		if tags[i].Count != tags[j].Count {
			return tags[i].Count > tags[j].Count
		}
		return tags[i].Tag < tags[j].Tag
	})

	return tags, nil
}

// intersectStrings returns the intersection of two string slices.
func intersectStrings(a, b []string) []string {
	set := make(map[string]bool)
	for _, s := range a {
		set[s] = true
	}
	var result []string
	for _, s := range b {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}

// collectDescendants returns a set of all bean IDs that are descendants of
// the given bean.
func collectDescendants(beanID string, allBeans []*bean.Bean) map[string]bool {
	descendants := make(map[string]bool)

	// Build parent->children map
	children := make(map[string][]string)
	for _, b := range allBeans {
		if b.Parent != "" {
			children[b.Parent] = append(children[b.Parent], b.ID)
		}
	}

	// BFS to collect all descendants
	queue := children[beanID]
	for len(queue) > 0 {
		childID := queue[0]
		queue = queue[1:]
		if !descendants[childID] {
			descendants[childID] = true
			queue = append(queue, children[childID]...)
		}
	}

	return descendants
}
