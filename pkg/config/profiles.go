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
//
// Task 9 fix round 2/3: `beans init --profile` also sets
// Config.TypesExclusive, which switches TypeList() off from merging onto
// DefaultTypes. With no defaults underneath, every entry here has to be
// self-sufficient - it is the whole rendered type, not an override layered
// onto a built-in. The rule for filling in a field this table does not name
// itself: DefaultTypes fills only what is left empty here. A name
// DefaultTypes already carries (milestone, epic, feature, bug, task) takes
// its Color, Emphasis and Description from DefaultTypes wherever this table
// leaves that field unset - one name, one meaning (D16) - but a field this
// table does set of its own accord (complex's "feature" Description, which
// carries the D13 value-origin wording) is never overwritten by that fill.
// The five genuinely new names (bucket, release, improvement, chore, story)
// keep the bespoke descriptions this table always gave them and stay
// without a colour.
var profiles = map[string][]TypeOverride{
	// classic reproduces the built-in set. It is what an existing store runs
	// on, so beans init --profile classic must be indistinguishable from
	// beans init: same ranks, same colours, same emphasis, same descriptions
	// as DefaultTypes.
	"classic": {
		{Name: "milestone", Rank: intPtr(1), Short: "M", Color: "mauve", Emphasis: boolPtr(true),
			Description: "A target release or checkpoint; group work that should ship together"},
		{Name: "epic", Rank: intPtr(2), Short: "E", Color: "blue", Emphasis: boolPtr(true),
			Description: "A thematic container for related work; should have child beans, not be worked on directly"},
		{Name: "feature", Rank: intPtr(3), Short: "F", Color: "sapphire",
			Description: "A user-facing capability or enhancement"},
		{Name: "bug", Rank: intPtr(LeafRank), Short: "B", Color: "maroon",
			Description: "Something that is broken and needs fixing"},
		{Name: "task", Rank: intPtr(LeafRank), Short: "T",
			Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
	},
	// todo is a flat list: one leaf type, no containers, grouping by tag.
	"todo": {
		{Name: "task", Rank: intPtr(LeafRank), Short: "T",
			Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
	},
	// simple splits rank 2 by subject matter: epic is the thematic container,
	// feature the user-facing capability.
	"simple": {
		{Name: "milestone", Rank: intPtr(1), Short: "M", Color: "mauve", Emphasis: boolPtr(true),
			Description: "A target release or checkpoint; group work that should ship together"},
		{Name: "bucket", Rank: intPtr(1), Short: "K", Roadmap: boolPtr(false),
			Description: "A parking lot for topics that might be picked up some day; carries work like a milestone but stays out of the roadmap and the milestone list"},
		{Name: "epic", Rank: intPtr(2), Short: "E", Color: "blue", Emphasis: boolPtr(true),
			Description: "A thematic container for related work; should have child beans, not be worked on directly"},
		{Name: "feature", Rank: intPtr(2), Short: "F", Color: "sapphire",
			Description: "A user-facing capability or enhancement"},
		{Name: "bug", Rank: intPtr(LeafRank), Short: "B", Color: "maroon",
			Description: "Something that is broken and needs fixing"},
		{Name: "task", Rank: intPtr(LeafRank), Short: "T",
			Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
	},
	// complex splits rank 2 by where the value comes from: first customer
	// benefit yes or no, then within "yes" new against existing.
	"complex": {
		{Name: "release", Rank: intPtr(1), Short: "R",
			Description: "A version that ships; gets a release tag and groups everything that goes out together"},
		{Name: "bucket", Rank: intPtr(1), Short: "K", Roadmap: boolPtr(false),
			Description: "A parking lot for topics that might be picked up some day; carries work like a release but stays out of the roadmap and the milestone list"},
		// feature keeps its own bespoke description here (not DefaultTypes'
		// generic one): in complex, rank 2 carries the value-origin axis
		// (feature / improvement / chore), and D13 needs this wording to
		// say so. DefaultTypes fills only the fields the brief left empty -
		// here that is Color, not Description.
		{Name: "feature", Rank: intPtr(2), Short: "F", Color: "sapphire",
			Description: "A new capability with customer benefit"},
		{Name: "improvement", Rank: intPtr(2), Short: "I",
			Description: "Work that makes an existing capability better and the customer notices; a refactoring nobody sees is a chore instead"},
		{Name: "chore", Rank: intPtr(2), Short: "C",
			Description: "Internal work without customer benefit — tooling, upgrades, documentation, refactoring; new or existing alike"},
		{Name: "story", Rank: intPtr(LeafRank), Short: "S",
			Description: "A slice of work a user can see; done when it can be demonstrated"},
		{Name: "bug", Rank: intPtr(LeafRank), Short: "B", Color: "maroon",
			Description: "Something that is broken and needs fixing"},
		{Name: "task", Rank: intPtr(LeafRank), Short: "T",
			Description: "A concrete piece of work to complete (eg. a chore, or a sub-task for a feature)"},
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
