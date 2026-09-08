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
// argument. Checking err != nil alone would also pass for an unrelated
// runtime failure inside RunE, which is not the defect this pins. Instead
// compare the end-to-end failure against archive's own Args policy run in
// isolation: they must be the identical error, proving the invocation was
// rejected by the declared usage/arity policy itself, not by anything that
// ran after it.
func TestArchiveRejectsPositionalArgumentEndToEnd(t *testing.T) {
	root := sharedTestRoot(t)
	archiveCmd, _, err := root.Find([]string{"archive"})
	if err != nil {
		t.Fatalf("finding archive: %v", err)
	}
	wantErr := archiveCmd.Args(archiveCmd, []string{"beans-nope"})
	if wantErr == nil {
		t.Fatal("archive's own Args policy accepted the argument; nothing to compare the end-to-end failure against")
	}

	_, _, gotErr := runRootWithArgs(t, "archive", "beans-nope")
	if gotErr == nil {
		t.Fatal("expected `beans archive <id>` to fail, it silently accepted the argument")
	}
	if gotErr.Error() != wantErr.Error() {
		t.Errorf("archive failed for a different reason than its arity policy: got %q, want the Args-policy error %q", gotErr, wantErr)
	}
}

// TestEveryCoreVerbDeclaresArity pins AC-01 across the whole tree, not just
// the nine files this leaf touches: every one of the 26 commands
// RegisterCoreCommands puts on the root (24 Register*Cmd calls plus the
// serve/tui stubs) must declare an explicit Args policy, so a future verb
// that forgets one is caught here rather than by nobody. Cobra's own
// completion/help builtins are excluded on purpose — their arity is
// cobra's, not this leaf's, per beans-t7sv Requirement 1 AC1. This
// exclusion is a scope boundary for ARITY, not the audience/plumbing
// classification SC-03 guards; it does not compete with IsUserFacing.
func TestEveryCoreVerbDeclaresArity(t *testing.T) {
	root := sharedTestRoot(t)
	for _, cmd := range root.Commands() {
		if cmd.Name() == "completion" || cmd.Name() == "help" {
			continue
		}
		if cmd.Args == nil {
			t.Errorf("%q has no declared Args policy", cmd.Name())
		}
	}
}
