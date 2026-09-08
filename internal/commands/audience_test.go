package commands

import "testing"

// TestAudienceMarkerCoversEveryRegisteredVerb pins AC1: every command in the
// tree carries a queryable audience classification, readable without
// parsing --help output.
func TestAudienceMarkerCoversEveryRegisteredVerb(t *testing.T) {
	root := sharedTestRoot(t)
	for _, cmd := range root.Commands() {
		if cmd.Annotations[audienceAnnotationKey] == "" {
			t.Errorf("%q carries no audience marker", cmd.Name())
		}
	}
}

// TestAudienceClassifiesThePlumbingSet pins AC2: completion, help, init,
// path, and version are plumbing; every other registered verb is
// user-facing.
func TestAudienceClassifiesThePlumbingSet(t *testing.T) {
	root := sharedTestRoot(t)
	plumbing := map[string]bool{
		"completion": true,
		"help":       true,
		"init":       true,
		"path":       true,
		"version":    true,
	}
	for _, cmd := range root.Commands() {
		want := plumbing[cmd.Name()]
		got := !IsUserFacing(cmd)
		if got != want {
			t.Errorf("%q: plumbing=%v, want %v", cmd.Name(), got, want)
		}
	}
}

// TestAudienceCoversDeprecatedStubs pins AC3: the hidden serve/tui stubs
// answer the audience query through the same marker mechanism as real
// commands (a queryable Annotations entry) rather than being left out of
// it. AC2 does not name them plumbing, so — like every verb it does not
// name — they classify as user-facing; AC3 only requires that the
// classification exists and is asked the same way, not that it differs.
func TestAudienceCoversDeprecatedStubs(t *testing.T) {
	root := sharedTestRoot(t)
	for _, name := range []string{"serve", "tui"} {
		cmd, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %q: %v", name, err)
		}
		if cmd.Annotations[audienceAnnotationKey] == "" {
			t.Errorf("%q carries no audience marker", name)
		}
		if !IsUserFacing(cmd) {
			t.Errorf("%q classified plumbing, want user-facing per AC2", name)
		}
	}
}
