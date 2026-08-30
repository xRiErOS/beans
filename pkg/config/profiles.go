package config

import "sort"

// intPtr keeps the profile tables readable; the override fields are pointers
// so that a partial override cannot reset a value it does not name. boolPtr
// already exists in config.go (used by Default) and is reused here.
func intPtr(i int) *int { return &i }

// profiles holds the type sets `beans init --profile` writes out. The list is
// expanded into the .beans.yml at init time rather than referenced by name:
// a later change to a profile definition must not silently move the types of
// an existing project.
var profiles = map[string][]TypeOverride{
	// classic reproduces the built-in set. It is what an existing store runs
	// on, so its ranks must match DefaultTypes exactly.
	"classic": {
		{Name: "milestone", Rank: intPtr(1), Short: "M"},
		{Name: "epic", Rank: intPtr(2), Short: "E"},
		{Name: "feature", Rank: intPtr(3), Short: "F"},
		{Name: "bug", Rank: intPtr(LeafRank), Short: "B"},
		{Name: "task", Rank: intPtr(LeafRank), Short: "T"},
	},
	// todo is a flat list: one leaf type, no containers, grouping by tag.
	"todo": {
		{Name: "task", Rank: intPtr(LeafRank), Short: "T"},
	},
	// simple splits rank 2 by subject matter: epic is the thematic container,
	// feature the user-facing capability.
	"simple": {
		{Name: "milestone", Rank: intPtr(1), Short: "M"},
		{Name: "bucket", Rank: intPtr(1), Short: "K", Roadmap: boolPtr(false),
			Description: "A parking lot for topics that might be picked up some day; carries work like a milestone but stays out of the roadmap and the milestone list"},
		{Name: "epic", Rank: intPtr(2), Short: "E"},
		{Name: "feature", Rank: intPtr(2), Short: "F"},
		{Name: "bug", Rank: intPtr(LeafRank), Short: "B"},
		{Name: "task", Rank: intPtr(LeafRank), Short: "T"},
	},
	// complex splits rank 2 by where the value comes from: first customer
	// benefit yes or no, then within "yes" new against existing.
	"complex": {
		{Name: "release", Rank: intPtr(1), Short: "R",
			Description: "A version that ships; gets a release tag and groups everything that goes out together"},
		{Name: "bucket", Rank: intPtr(1), Short: "K", Roadmap: boolPtr(false),
			Description: "A parking lot for topics that might be picked up some day; carries work like a release but stays out of the roadmap and the milestone list"},
		{Name: "feature", Rank: intPtr(2), Short: "F",
			Description: "A new capability with customer benefit"},
		{Name: "improvement", Rank: intPtr(2), Short: "I",
			Description: "Work that makes an existing capability better and the customer notices; a refactoring nobody sees is a chore instead"},
		{Name: "chore", Rank: intPtr(2), Short: "C",
			Description: "Internal work without customer benefit — tooling, upgrades, documentation, refactoring; new or existing alike"},
		{Name: "story", Rank: intPtr(LeafRank), Short: "S",
			Description: "A slice of work a user can see; done when it can be demonstrated"},
		{Name: "bug", Rank: intPtr(LeafRank), Short: "B"},
		{Name: "task", Rank: intPtr(LeafRank), Short: "T"},
	},
}

// ProfileNames returns the available profile names, sorted.
func ProfileNames() []string {
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ProfileTypes returns the expanded type list of a profile.
func ProfileTypes(name string) ([]TypeOverride, bool) {
	list, ok := profiles[name]
	if !ok {
		return nil, false
	}
	out := make([]TypeOverride, len(list))
	copy(out, list)
	return out, true
}
