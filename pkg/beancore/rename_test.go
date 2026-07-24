package beancore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/config"
)

// newTestCore builds a Core over a temp .beans dir with one bean file.
func newTestCore(t *testing.T, prefix string, files map[string]string) *Core {
	t.Helper()
	repo := t.TempDir()
	beansDir := filepath.Join(repo, ".beans")
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(beansDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultWithPrefix(prefix)
	cfg.SetConfigDir(repo) // ensure Save/ConfigDir target repo root
	c := New(beansDir, cfg)
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestApplyRenameSlug_setsSlug(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--old-slug.md": "# tp-aaaa\n---\ntitle: Test Bean\nstatus: todo\ntype: task\n---\nBody.\n",
	})
	plan, err := c.PlanRenameSlug("tp-aaaa", strPtr("new-slug"), false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != "slug" || len(plan.Changes) != 1 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if plan.Changes[0].NewPath != "tp-aaaa--new-slug.md" { // .beans-relative
		t.Errorf("NewPath = %q", plan.Changes[0].NewPath)
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	// old gone, new present
	if _, err := os.Stat(filepath.Join(c.Root(), "tp-aaaa--old-slug.md")); !os.IsNotExist(err) {
		t.Error("old file still present")
	}
	if _, err := os.Stat(filepath.Join(c.Root(), "tp-aaaa--new-slug.md")); err != nil {
		t.Errorf("new file missing: %v", err)
	}
}

func TestApplyRenameSlug_idAndRefsUnchanged(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--parent.md": "# tp-aaaa\n---\ntitle: Parent\nstatus: todo\ntype: epic\n---\n",
		"tp-bbbb--child.md":  "# tp-bbbb\n---\ntitle: Child\nstatus: todo\ntype: task\nparent: tp-aaaa\n---\n",
	})
	plan, err := c.PlanRenameSlug("tp-aaaa", strPtr("renamed"), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	// ID unchanged, in-memory Path/Slug updated
	b, err := c.Get("tp-aaaa")
	if err != nil {
		t.Fatalf("bean still resolvable under same ID: %v", err)
	}
	if b.Slug != "renamed" {
		t.Errorf("Slug = %q, want renamed", b.Slug)
	}
	if b.Path != "tp-aaaa--renamed.md" {
		t.Errorf("Path = %q, want tp-aaaa--renamed.md", b.Path)
	}
	// child's ref to parent is untouched (still references tp-aaaa, the ID)
	content, err := os.ReadFile(filepath.Join(c.Root(), "tp-bbbb--child.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(content), "parent: tp-aaaa") {
		t.Errorf("child ref rewritten unexpectedly: %s", content)
	}
	// on-disk id comment for the renamed bean is unchanged
	renamedContent, err := os.ReadFile(filepath.Join(c.Root(), "tp-aaaa--renamed.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(string(renamedContent), "# tp-aaaa") {
		t.Errorf("id comment changed unexpectedly: %s", renamedContent)
	}
}

func TestPlanRenameSlug_noopWhenPathsIdentical(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--same.md": "# tp-aaaa\n---\ntitle: Test Bean\nstatus: todo\ntype: task\n---\n",
	})
	plan, err := c.PlanRenameSlug("tp-aaaa", strPtr("same"), false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes[0].OldPath != plan.Changes[0].NewPath {
		t.Fatalf("expected identical paths, got %q vs %q", plan.Changes[0].OldPath, plan.Changes[0].NewPath)
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(c.Root(), "tp-aaaa--same.md")); err != nil {
		t.Errorf("file should still be present: %v", err)
	}
}

func strPtr(s string) *string { return &s }

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}
