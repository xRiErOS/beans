package commands

import "fmt"

// validateRoadmapDepth rejects a --depth the pruner cannot honour. Only an
// explicitly passed flag is checked: the zero value means the flag was never
// set, which is "no limit" and not a user error. changed comes from
// cobra's Flags().Changed, so `--depth 0` is still rejected.
func validateRoadmapDepth(depth int, changed bool) error {
	if changed && depth < 1 {
		return fmt.Errorf("--depth must be >= 1, got %d", depth)
	}
	return nil
}

// pruneRoadmapDepth truncates data in place to maxDepth levels, following
// `tree -L n` semantics: the root never counts. Without a scope ID the root
// is the roadmap as a whole, so milestones -- and the unscheduled branch's
// top-level entries, since "Unscheduled" is a heading and not a bean -- are
// level 1. With a scope ID the root is the scoped bean itself, so its direct
// children are level 1.
//
// This runs on the finished roadmapData rather than inside buildRoadmap on
// purpose: the builder drops milestones without visible content, so pruning
// there would empty the whole roadmap at --depth 1. Working on the built
// structure also means --json, the Markdown template and the TTY renderer
// all see the same truncation.
//
// maxDepth < 1 is a no-op; the command rejects those values before we get
// here, and a zero value means the flag was never set.
func pruneRoadmapDepth(data *roadmapData, maxDepth int, scoped bool) {
	if maxDepth < 1 {
		return
	}

	// Level of the topmost rows in data. A scoped roadmap's root row sits at
	// level 0 so that its children land on 1.
	top := 1
	if scoped {
		top = 0
	}

	if data.Root != nil {
		if data.Root.Epic != nil {
			pruneEpicGroup(data.Root.Epic, top, maxDepth)
		}
		if data.Root.Feature != nil {
			pruneFeatureGroup(data.Root.Feature, top, maxDepth)
		}
		return
	}

	for i := range data.Milestones {
		mg := &data.Milestones[i]
		childLevel := top + 1
		if childLevel > maxDepth {
			mg.Epics, mg.Features, mg.Other = nil, nil, nil
			continue
		}
		for j := range mg.Epics {
			pruneEpicGroup(&mg.Epics[j], childLevel, maxDepth)
		}
		for j := range mg.Features {
			pruneFeatureGroup(&mg.Features[j], childLevel, maxDepth)
		}
	}

	if data.Unscheduled != nil {
		un := data.Unscheduled
		for i := range un.Epics {
			pruneEpicGroup(&un.Epics[i], top, maxDepth)
		}
		for i := range un.Features {
			pruneFeatureGroup(&un.Features[i], top, maxDepth)
		}
	}
}

// pruneEpicGroup cuts an epic's children when they sit below maxDepth.
// level is the level of the epic row itself.
func pruneEpicGroup(eg *epicGroup, level, maxDepth int) {
	childLevel := level + 1
	if childLevel > maxDepth {
		eg.Items, eg.Features = nil, nil
		return
	}
	for i := range eg.Features {
		pruneFeatureGroup(&eg.Features[i], childLevel, maxDepth)
	}
}

// pruneFeatureGroup cuts a feature's leaf items when they sit below
// maxDepth. level is the level of the feature row itself.
func pruneFeatureGroup(fg *featureGroup, level, maxDepth int) {
	if level+1 > maxDepth {
		fg.Items = nil
	}
}
