package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestIsUserFacingDefaultsClosed pins the direction IsUserFacing itself
// takes if the wiring invariant below were ever violated. In the running
// tree an unmarked command cannot occur: RegisterCoreCommands's final pass
// explicitly marks every not-yet-plumbing command user-facing, so
// "unmarked" is not a live state, not a silent fallthrough. This test
// exercises IsUserFacing directly on a bare command that never went
// through that wiring, and requires it to read as plumbing rather than
// defaulting open.
func TestIsUserFacingDefaultsClosed(t *testing.T) {
	bare := &cobra.Command{Use: "unclassified"}
	if IsUserFacing(bare) {
		t.Error("a command with no audience annotation reported user-facing, want plumbing-by-default")
	}
}

// TestCobraBuiltinsArePresentAndPlumbing guards against the marker becoming
// vacuous. TestAudienceClassifiesThePlumbingSet only classifies whatever is
// already in root.Commands(): if the InitDefaultHelpCmd/
// InitDefaultCompletionCmd calls in RegisterCoreCommands were ever deleted,
// "completion" and "help" would simply be absent from the tree, and that
// test would stay green over a smaller set. This test finds them by name
// and requires both that they exist and that they are plumbing.
func TestCobraBuiltinsArePresentAndPlumbing(t *testing.T) {
	root := sharedTestRoot(t)
	for _, name := range []string{"completion", "help"} {
		cmd, _, err := root.Find([]string{name})
		if err != nil {
			t.Fatalf("%q is missing from the command tree: %v", name, err)
		}
		if got := cmd.Annotations[audienceAnnotationKey]; got != audiencePlumbing {
			t.Errorf("%q carries audience marker %q, want explicit %q (not merely unmarked)",
				name, got, audiencePlumbing)
		}
	}
}

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
