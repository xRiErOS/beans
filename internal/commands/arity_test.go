package commands

import "testing"

// TestNoArgsCommandsRejectPositionalArguments pins SC-01/SC-02: the eight
// no-argument verbs among the nine previously Args-less commands (archive,
// check, init, list, path, version, plus the deprecated serve/tui stubs) now
// declare an explicit arity policy instead of relying on cobra's implicit
// "accept anything" default. A stray positional argument must be a usage
// error, and zero arguments must still be accepted.
func TestNoArgsCommandsRejectPositionalArguments(t *testing.T) {
	root := sharedTestRoot(t)

	for _, name := range []string{"archive", "check", "init", "list", "path", "version", "serve", "tui"} {
		name := name
		t.Run(name, func(t *testing.T) {
			cmd, _, err := root.Find([]string{name})
			if err != nil {
				t.Fatalf("finding %q: %v", name, err)
			}
			if cmd.Args == nil {
				t.Fatalf("%q has no declared Args policy", name)
			}
			if err := cmd.Args(cmd, []string{"unexpected"}); err == nil {
				t.Errorf("%q accepted a positional argument, want rejection", name)
			}
			if err := cmd.Args(cmd, nil); err != nil {
				t.Errorf("%q rejected zero arguments: %v", name, err)
			}
		})
	}
}

// TestCreateDeclaresExplicitArity pins the ninth verb: create's title is an
// unbounded run of positional words (`beans create foo bar` -> "foo bar"),
// so its declared policy is cobra.ArbitraryArgs — an explicit opt-in to
// "accept any count" rather than an undeclared nil Args left at the same
// behaviour by accident (SC-01).
func TestCreateDeclaresExplicitArity(t *testing.T) {
	root := sharedTestRoot(t)
	cmd, _, err := root.Find([]string{"create"})
	if err != nil {
		t.Fatalf("finding create: %v", err)
	}
	if cmd.Args == nil {
		t.Fatal("create has no declared Args policy")
	}
	if err := cmd.Args(cmd, []string{"a", "b", "c"}); err != nil {
		t.Errorf("create rejected a multi-word title: %v", err)
	}
}

// TestArchiveRejectsPositionalArgumentEndToEnd is the D11/R-06 regression
// (beans-smwg): `beans archive <id>` used to exit 0 and silently drop the
// argument. The whole invocation must now fail, not just the isolated
// Args() check in TestNoArgsCommandsRejectPositionalArguments.
func TestArchiveRejectsPositionalArgumentEndToEnd(t *testing.T) {
	_, _, err := runRootWithArgs(t, "archive", "beans-nope")
	if err == nil {
		t.Fatal("expected `beans archive <id>` to fail, it silently accepted the argument")
	}
}
