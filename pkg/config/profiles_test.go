package config

import "testing"

func rankOfIn(list []TypeOverride, name string) int {
	for _, t := range list {
		if t.Name == name && t.Rank != nil {
			return *t.Rank
		}
	}
	return 0
}

func TestProfileNamesCoversAllFour(t *testing.T) {
	got := ProfileNames()
	want := []string{"classic", "complex", "simple", "todo"}
	if len(got) != len(want) {
		t.Fatalf("ProfileNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ProfileNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestTodoProfileIsASingleLeaf(t *testing.T) {
	list, ok := ProfileTypes("todo")
	if !ok {
		t.Fatal("ProfileTypes(\"todo\") not found")
	}
	if len(list) != 1 {
		t.Fatalf("todo has %d types, want 1", len(list))
	}
	if list[0].Name != "task" || rankOfIn(list, "task") != LeafRank {
		t.Errorf("todo carries %+v, want a single task on the leaf rank", list[0])
	}
}

func TestSimpleProfileMatchesTheDesign(t *testing.T) {
	list, _ := ProfileTypes("simple")
	for name, want := range map[string]int{
		"milestone": 1, "bucket": 1,
		"epic": 2, "feature": 2,
		"task": LeafRank, "bug": LeafRank,
	} {
		if got := rankOfIn(list, name); got != want {
			t.Errorf("simple: rank of %q = %d, want %d", name, got, want)
		}
	}
	if len(list) != 6 {
		t.Errorf("simple has %d types, want 6", len(list))
	}
}

func TestComplexProfileMatchesTheDesign(t *testing.T) {
	list, _ := ProfileTypes("complex")
	for name, want := range map[string]int{
		"release": 1, "bucket": 1,
		"feature": 2, "improvement": 2, "chore": 2,
		"story": LeafRank, "bug": LeafRank, "task": LeafRank,
	} {
		if got := rankOfIn(list, name); got != want {
			t.Errorf("complex: rank of %q = %d, want %d", name, got, want)
		}
	}
}

// Task 9 fix round 3: DefaultTypes fills a field only where this profile
// leaves it empty. The brief gives complex's "feature" its own Description
// ("A new capability with customer benefit") because in complex, rank 2
// carries the value-origin axis (feature / improvement / chore) that D13
// depends on - the generic DefaultTypes text for "feature" does not say
// that, and must not silently replace it.
func TestComplexFeatureKeepsItsBespokeDescription(t *testing.T) {
	list, _ := ProfileTypes("complex")
	for _, ty := range list {
		if ty.Name != "feature" {
			continue
		}
		want := "A new capability with customer benefit"
		if ty.Description != want {
			t.Errorf("complex feature Description = %q, want %q", ty.Description, want)
		}
		return
	}
	t.Fatal("complex profile has no feature type")
}

func TestBucketOptsOutOfTheRoadmapInEveryProfile(t *testing.T) {
	for _, profile := range []string{"simple", "complex"} {
		list, _ := ProfileTypes(profile)
		found := false
		for _, t2 := range list {
			if t2.Name != "bucket" {
				continue
			}
			found = true
			if t2.Roadmap == nil || *t2.Roadmap {
				t.Errorf("%s: bucket must carry roadmap: false", profile)
			}
		}
		if !found {
			t.Errorf("%s: no bucket type", profile)
		}
	}
}

func TestEveryProfileKeepsOneNameOnOneRank(t *testing.T) {
	ranks := map[string]map[int]string{}
	for _, profile := range ProfileNames() {
		list, _ := ProfileTypes(profile)
		for _, ty := range list {
			if ty.Rank == nil {
				continue
			}
			if ranks[ty.Name] == nil {
				ranks[ty.Name] = map[int]string{}
			}
			ranks[ty.Name][*ty.Rank] = profile
		}
	}
	for name, seen := range ranks {
		// feature is the one documented exception: classic keeps it on rank 3
		// because the existing stores carry epic -> feature beans (D16).
		if name == "feature" {
			continue
		}
		if len(seen) > 1 {
			t.Errorf("type %q sits on %d different ranks across profiles: %v", name, len(seen), seen)
		}
	}
}

func TestUnknownProfileIsReported(t *testing.T) {
	if _, ok := ProfileTypes("enterprise"); ok {
		t.Error("ProfileTypes(\"enterprise\") reported success for an unknown profile")
	}
}

// Task 9 fix round 2: init --profile now sets TypesExclusive, so a profile's
// types are the whole rendered table, not overrides layered onto
// DefaultTypes. Every entry - including the five names DefaultTypes already
// carries - must ship its own description; there is no defaults fallback
// left to inherit one from.
func TestEveryProfileTypeCarriesADescription(t *testing.T) {
	for _, profile := range ProfileNames() {
		list, _ := ProfileTypes(profile)
		for _, ty := range list {
			if ty.Description == "" {
				t.Errorf("%s: type %q ships without a description", profile, ty.Name)
			}
		}
	}
}
