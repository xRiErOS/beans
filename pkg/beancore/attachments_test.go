package beancore

import (
	"os"
	"path/filepath"
	"testing"
)

// writeAttachment drops a file into attachments/<id>/ of the core's store.
func writeAttachment(t *testing.T, c *Core, id, name, content string) string {
	t.Helper()
	dir := filepath.Join(c.Root(), AttachmentsDir, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	full := filepath.Join(dir, name)
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return full
}

// D24/R-31: the bean walk loads every .md under any non-dot subdirectory, so
// before this guard a guard draft parked next to a diff was parsed into a
// bean of its own -- measured as ID "guard" out of "guard-draft.md", visible
// in `beans list`. attachments/ therefore has to be skipped by the walk, and
// it deliberately stays non-dot-prefixed because R-21 has it cited from bean
// bodies.
func TestLoad_ignoresMarkdownUnderAttachments(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--host.md": "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	writeAttachment(t, c, "tp-aaaa", "guard-draft.md", "# Guard draft\n\nnotes\n")
	writeAttachment(t, c, "tp-aaaa", "review.json", "{\"findings\":[]}\n")

	if err := c.Load(); err != nil {
		t.Fatalf("Load after attachment: %v", err)
	}
	all := c.All()
	if len(all) != 1 {
		ids := make([]string, 0, len(all))
		for _, b := range all {
			ids = append(ids, b.ID)
		}
		t.Fatalf("attachment parsed into the store: got %d beans %v, want only tp-aaaa", len(all), ids)
	}
	if all[0].ID != "tp-aaaa" {
		t.Fatalf("wrong bean survived: %q", all[0].ID)
	}
}

// D15/R-23: a rename that changes the ID must carry the attachment directory
// with it. Measured before the fix: `rename <id> <new-id>` moved the .md and
// left attachments/<old>/ behind, and `check` still reported "All checks
// passed" over the dangling directory.
func TestRenameID_carriesAttachments(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--host.md": "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	writeAttachment(t, c, "tp-aaaa", "review.json", "{\"findings\":[]}\n")

	plan, err := c.PlanRenameID("tp-aaaa", "tp-bbbb")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.AttachmentMoves) != 1 {
		t.Fatalf("plan carries %d attachment moves, want 1", len(plan.AttachmentMoves))
	}
	if got, want := plan.AttachmentMoves[0].OldPath, filepath.Join(AttachmentsDir, "tp-aaaa"); got != want {
		t.Errorf("OldPath = %q, want %q", got, want)
	}
	if got, want := plan.AttachmentMoves[0].NewPath, filepath.Join(AttachmentsDir, "tp-bbbb"); got != want {
		t.Errorf("NewPath = %q, want %q", got, want)
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(c.Root(), AttachmentsDir, "tp-bbbb", "review.json")); err != nil {
		t.Errorf("attachment did not follow the rename: %v", err)
	}
	if _, err := os.Stat(filepath.Join(c.Root(), AttachmentsDir, "tp-aaaa")); !os.IsNotExist(err) {
		t.Errorf("stale attachment directory survived: err=%v", err)
	}
}

// D15/R-23: the prefix rebrand is the mode that actually ran in this hull,
// across 34 beans. Its drift was the silent one -- check green over a
// directory pointing at an ID that no longer existed.
func TestRebrand_carriesAttachments(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--host.md":  "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
		"tp-cccc--other.md": "---\n# tp-cccc\ntitle: Other Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	writeAttachment(t, c, "tp-aaaa", "review.json", "{\"findings\":[]}\n")

	plan, err := c.PlanRebrand("rev-")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.AttachmentMoves) != 1 {
		t.Fatalf("plan carries %d attachment moves, want 1 (only tp-aaaa has one)", len(plan.AttachmentMoves))
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(c.Root(), AttachmentsDir, "rev-aaaa", "review.json")); err != nil {
		t.Errorf("attachment did not follow the rebrand: %v", err)
	}
	if _, err := os.Stat(filepath.Join(c.Root(), AttachmentsDir, "tp-aaaa")); !os.IsNotExist(err) {
		t.Errorf("stale attachment directory survived the rebrand: err=%v", err)
	}
}

// A slug rename does not touch the ID, so there is nothing to move. Asserted
// because the plan builder must not invent a no-op move that the renderer
// would then print.
func TestRenameSlug_leavesAttachmentsAlone(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--old-slug.md": "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	writeAttachment(t, c, "tp-aaaa", "review.json", "{\"findings\":[]}\n")

	newSlug := "new-slug"
	plan, err := c.PlanRenameSlug("tp-aaaa", &newSlug, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.AttachmentMoves) != 0 {
		t.Fatalf("slug rename planned %d attachment moves, want 0", len(plan.AttachmentMoves))
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(c.Root(), AttachmentsDir, "tp-aaaa", "review.json")); err != nil {
		t.Errorf("slug rename disturbed the attachment: %v", err)
	}
}

// A bean without an attachment must not produce a move, otherwise every
// rename would try to move a directory that does not exist.
func TestRenameID_noMoveWithoutAttachment(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--host.md": "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	plan, err := c.PlanRenameID("tp-aaaa", "tp-bbbb")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.AttachmentMoves) != 0 {
		t.Fatalf("planned %d attachment moves for a bean without attachments, want 0", len(plan.AttachmentMoves))
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatalf("rename without attachments failed: %v", err)
	}
}

// T02/R-23: the orphan check is what turns the invariant from an assertion
// into something verified. Measured before the fix: check reported "All
// checks passed" over an attachment directory whose bean no longer existed.
// An archived bean stays loaded (the walk enters archive/ deliberately), so
// its attachment must NOT be reported.
func TestOrphanAttachments(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--host.md":             "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
		"archive/tp-dddd--archived.md": "---\n# tp-dddd\ntitle: Archived Bean\nstatus: completed\ntype: task\n---\nBody.\n",
	})
	writeAttachment(t, c, "tp-aaaa", "review.json", "{}\n")
	writeAttachment(t, c, "tp-dddd", "review.json", "{}\n")
	writeAttachment(t, c, "tp-9999", "review.json", "{}\n")
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}

	orphans, err := c.OrphanAttachments()
	if err != nil {
		t.Fatal(err)
	}
	if len(orphans) != 1 || orphans[0] != "tp-9999" {
		t.Fatalf("orphans = %v, want exactly [tp-9999]", orphans)
	}
}

// No attachments directory at all is the normal case for every existing
// store and must not error.
func TestOrphanAttachments_absentDirectory(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--host.md": "---\n# tp-aaaa\ntitle: Host Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	orphans, err := c.OrphanAttachments()
	if err != nil {
		t.Fatalf("absent attachments dir errored: %v", err)
	}
	if len(orphans) != 0 {
		t.Fatalf("orphans = %v, want none", orphans)
	}
}
