package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
	"github.com/spf13/cobra"
)

// setupOrderTest installs a throwaway core and default config into the
// package globals orderCmd.RunE reads, creates a parent bean plus n
// children (in order, with ascending single-letter Order keys "A".."?"),
// and returns the parent and its children.
func setupOrderTest(t *testing.T, n int) (parent *bean.Bean, children []*bean.Bean) {
	t.Helper()
	tmpDir := t.TempDir()
	beansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	testCfg := config.Default()
	testCore := beancore.New(beansDir, testCfg)
	if err := testCore.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	oldCore, oldCfg := core, cfg
	core, cfg = testCore, testCfg
	t.Cleanup(func() { core, cfg = oldCore, oldCfg })

	parent = &bean.Bean{
		ID:     "beans-parent",
		Slug:   bean.Slugify("Parent"),
		Title:  "Parent",
		Status: "todo",
		Type:   "epic",
	}
	if err := core.Create(parent); err != nil {
		t.Fatalf("core.Create(parent) error = %v", err)
	}

	letters := "ABCDEFGHIJ"
	for i := 0; i < n; i++ {
		c := &bean.Bean{
			ID:     "beans-child" + string(letters[i]),
			Slug:   bean.Slugify("Child " + string(letters[i])),
			Title:  "Child " + string(letters[i]),
			Status: "todo",
			Type:   "task",
			Parent: parent.ID,
			Order:  string(letters[i]),
		}
		if err := core.Create(c); err != nil {
			t.Fatalf("core.Create(child) error = %v", err)
		}
		children = append(children, c)
	}

	return parent, children
}

// resetOrderFlags clears every order* package global the tests below touch.
func resetOrderFlags(t *testing.T) {
	t.Helper()
	oldAfter, oldBefore, oldFirst, oldLast := orderAfter, orderBefore, orderFirst, orderLast
	orderAfter, orderBefore, orderFirst, orderLast = "", "", false, false
	t.Cleanup(func() {
		orderAfter, orderBefore, orderFirst, orderLast = oldAfter, oldBefore, oldFirst, oldLast
	})
}

// allBeanFileContents walks the beans dir and returns a map of relative path
// to file content, so tests can diff "what changed on disk" without relying
// on mtime resolution.
func allBeanFileContents(t *testing.T, root string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("walking beans dir: %v", err)
	}
	return out
}

// AC1: --after places the bean strictly between that sibling and its
// successor.
func TestOrderCmdAfterSetsBetweenSiblings(t *testing.T) {
	_, children := setupOrderTest(t, 3) // A, B, C
	resetOrderFlags(t)

	orderAfter = children[0].ID // A

	if err := orderCmd.RunE(orderCmd, []string{children[2].ID}); err != nil { // move C
		t.Fatalf("orderCmd.RunE() error = %v", err)
	}

	got, err := core.Get(children[2].ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if !(got.Order > "A" && got.Order < "B") {
		t.Errorf("Order = %q, want strictly between %q and %q", got.Order, "A", "B")
	}
}

// AC2: --before places the bean strictly between that sibling and its
// predecessor.
func TestOrderCmdBeforeSetsBetweenSiblings(t *testing.T) {
	_, children := setupOrderTest(t, 3) // A, B, C
	resetOrderFlags(t)

	orderBefore = children[2].ID // C

	if err := orderCmd.RunE(orderCmd, []string{children[0].ID}); err != nil { // move A
		t.Fatalf("orderCmd.RunE() error = %v", err)
	}

	got, err := core.Get(children[0].ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if !(got.Order > "B" && got.Order < "C") {
		t.Errorf("Order = %q, want strictly between %q and %q", got.Order, "B", "C")
	}
}

// AC3: --first sorts before every sibling.
func TestOrderCmdFirstSortsBeforeAllSiblings(t *testing.T) {
	_, children := setupOrderTest(t, 3) // A, B, C
	resetOrderFlags(t)

	orderFirst = true

	if err := orderCmd.RunE(orderCmd, []string{children[2].ID}); err != nil { // move C
		t.Fatalf("orderCmd.RunE() error = %v", err)
	}

	got, err := core.Get(children[2].ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Order >= "A" {
		t.Errorf("Order = %q, want sorting before %q", got.Order, "A")
	}
}

// AC4: --last sorts after every sibling.
func TestOrderCmdLastSortsAfterAllSiblings(t *testing.T) {
	_, children := setupOrderTest(t, 3) // A, B, C
	resetOrderFlags(t)

	orderLast = true

	if err := orderCmd.RunE(orderCmd, []string{children[0].ID}); err != nil { // move A
		t.Fatalf("orderCmd.RunE() error = %v", err)
	}

	got, err := core.Get(children[0].ID)
	if err != nil {
		t.Fatalf("core.Get() error = %v", err)
	}
	if got.Order <= "C" {
		t.Errorf("Order = %q, want sorting after %q", got.Order, "C")
	}
}

// AC5/SC-01: moving the fifth of six siblings to second position leaves the
// six in the expected sequence and modifies exactly one file on disk —
// asserted by a write count (file-content diff), not merely the resulting
// order, so a full renumbering implementation would fail this test even if
// the sequence came out right.
func TestOrderCmdWritesExactlyOneFile(t *testing.T) {
	_, children := setupOrderTest(t, 6) // A..F
	resetOrderFlags(t)

	before := allBeanFileContents(t, core.Root())

	orderAfter = children[0].ID // A

	if err := orderCmd.RunE(orderCmd, []string{children[4].ID}); err != nil { // move E after A
		t.Fatalf("orderCmd.RunE() error = %v", err)
	}

	after := allBeanFileContents(t, core.Root())

	changed := 0
	var changedPath string
	for path, content := range before {
		if after[path] != content {
			changed++
			changedPath = path
		}
	}
	if changed != 1 {
		t.Fatalf("expected exactly 1 file to change, got %d", changed)
	}
	if changedPath != children[4].Path {
		t.Errorf("changed file = %q, want the moved bean's file %q", changedPath, children[4].Path)
	}

	// Resulting sequence: A, E, B, C, D, F.
	all, err := core.Get(children[0].Parent)
	if err != nil {
		t.Fatalf("core.Get(parent) error = %v", err)
	}
	_ = all
	siblings := make([]*bean.Bean, 0, 6)
	for _, c := range children {
		got, err := core.Get(c.ID)
		if err != nil {
			t.Fatalf("core.Get() error = %v", err)
		}
		siblings = append(siblings, got)
	}
	bean.SortByOrder(siblings)

	wantSeq := []string{children[0].ID, children[4].ID, children[1].ID, children[2].ID, children[3].ID, children[5].ID}
	for i, id := range wantSeq {
		if siblings[i].ID != id {
			t.Errorf("siblings[%d].ID = %q, want %q", i, siblings[i].ID, id)
		}
	}
}

// AC6: --after/--before naming a bean under a different parent is scoped
// per parent and fails.
func TestOrderCmdDifferentParentFails(t *testing.T) {
	parent, children := setupOrderTest(t, 2)
	resetOrderFlags(t)

	// A sibling epic with its own child, unrelated to `parent`.
	otherParent := &bean.Bean{
		ID: "beans-other", Slug: bean.Slugify("Other"), Title: "Other", Status: "todo", Type: "epic",
	}
	if err := core.Create(otherParent); err != nil {
		t.Fatalf("core.Create(otherParent) error = %v", err)
	}
	outsider := &bean.Bean{
		ID: "beans-outsider", Slug: bean.Slugify("Outsider"), Title: "Outsider",
		Status: "todo", Type: "task", Parent: otherParent.ID, Order: "A",
	}
	if err := core.Create(outsider); err != nil {
		t.Fatalf("core.Create(outsider) error = %v", err)
	}
	_ = parent

	orderAfter = outsider.ID

	err := orderCmd.RunE(orderCmd, []string{children[0].ID})
	if err == nil {
		t.Fatal("expected error for cross-parent --after")
	}
	if !contains(err.Error(), "scoped per parent") {
		t.Errorf("expected error to mention scope, got %q", err.Error())
	}
}

// AC7: more than one of --after/--before/--first/--last is a usage error.
// Cobra enforces this via MarkFlagsMutuallyExclusive, which only fires
// during flag parsing/validation on a real Execute() — not when calling
// RunE directly (as the other tests here do) — so this test drives the
// command the way the CLI actually does.
func TestOrderCmdMutuallyExclusiveFlags(t *testing.T) {
	_, children := setupOrderTest(t, 2)
	resetOrderFlags(t)

	root := &cobra.Command{Use: "beans"}
	RegisterOrderCmd(root)
	root.SetArgs([]string{"order", children[0].ID, "--first", "--last"})
	root.SetOut(new(nopWriter))
	root.SetErr(new(nopWriter))

	if err := root.Execute(); err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
}

// No placement flag at all is also a usage error (nothing to do).
func TestOrderCmdRequiresAPlacementFlag(t *testing.T) {
	_, children := setupOrderTest(t, 2)
	resetOrderFlags(t)

	if err := orderCmd.RunE(orderCmd, []string{children[0].ID}); err == nil {
		t.Fatal("expected error when no placement flag is given")
	}
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
