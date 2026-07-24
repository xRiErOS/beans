package beancore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
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

func TestRewriteRefs(t *testing.T) {
	tests := []struct {
		name          string
		parent        string
		blocking      []string
		blockedBy     []string
		m             map[string]string
		wantN         int
		wantParent    string
		wantBlocking  []string
		wantBlockedBy []string
	}{
		{
			name:          "mixed map: parent + one blocking entry changed, rest untouched",
			parent:        "old-1",
			blocking:      []string{"old-1", "keep-9"},
			blockedBy:     []string{"other-5"},
			m:             map[string]string{"old-1": "new-1"},
			wantN:         2,
			wantParent:    "new-1",
			wantBlocking:  []string{"new-1", "keep-9"},
			wantBlockedBy: []string{"other-5"},
		},
		{
			name:          "ref not in map left unchanged",
			parent:        "untouched-1",
			blocking:      nil,
			blockedBy:     []string{"untouched-2"},
			m:             map[string]string{"old-1": "new-1"},
			wantN:         0,
			wantParent:    "untouched-1",
			wantBlocking:  nil,
			wantBlockedBy: []string{"untouched-2"},
		},
		{
			name:          "empty parent never matched",
			parent:        "",
			blocking:      []string{"old-1"},
			blockedBy:     nil,
			m:             map[string]string{"": "should-not-apply", "old-1": "new-1"},
			wantN:         1,
			wantParent:    "",
			wantBlocking:  []string{"new-1"},
			wantBlockedBy: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bean.Bean{
				ID:        "x-2",
				Parent:    tt.parent,
				Blocking:  append([]string(nil), tt.blocking...),
				BlockedBy: append([]string(nil), tt.blockedBy...),
			}
			n := rewriteRefs(b, tt.m)
			if n != tt.wantN {
				t.Errorf("changed count = %d, want %d", n, tt.wantN)
			}
			if b.Parent != tt.wantParent {
				t.Errorf("Parent = %q, want %q", b.Parent, tt.wantParent)
			}
			if !equalStrSlices(b.Blocking, tt.wantBlocking) {
				t.Errorf("Blocking = %v, want %v", b.Blocking, tt.wantBlocking)
			}
			if !equalStrSlices(b.BlockedBy, tt.wantBlockedBy) {
				t.Errorf("BlockedBy = %v, want %v", b.BlockedBy, tt.wantBlockedBy)
			}
		})
	}
}

func TestStageAndSwap_atomicOnPreSwapFailure(t *testing.T) {
	tests := []struct {
		name    string
		writes  map[string][]byte
		removes []string
	}{
		{
			name: "write path with NUL byte fails MkdirAll during staging",
			writes: map[string][]byte{
				filepath.Join("\x00bad", "x.md"): []byte("nope"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestCore(t, "tp-", map[string]string{
				"tp-aaaa--a.md": "content-a",
				"tp-bbbb--b.md": "content-b",
			})
			err := c.stageAndSwap(tt.writes, tt.removes)
			if err == nil {
				t.Fatal("expected staging failure, got nil")
			}
			// original tree intact, byte-for-byte
			got, readErr := os.ReadFile(filepath.Join(c.Root(), "tp-aaaa--a.md"))
			if readErr != nil || string(got) != "content-a" {
				t.Errorf("original file corrupted: content=%q err=%v", got, readErr)
			}
			got2, readErr2 := os.ReadFile(filepath.Join(c.Root(), "tp-bbbb--b.md"))
			if readErr2 != nil || string(got2) != "content-b" {
				t.Errorf("original file corrupted: content=%q err=%v", got2, readErr2)
			}
			// no staging/backup siblings left behind
			entries, err := os.ReadDir(c.repoRoot())
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				name := e.Name()
				if name != ".beans" && strings.HasPrefix(name, ".beans") {
					t.Errorf("leftover sibling: %s", name)
				}
			}
		})
	}
}

func TestStageAndSwap_appliesWritesAndRemoves(t *testing.T) {
	tests := []struct {
		name          string
		writes        map[string][]byte
		removes       []string
		wantPresent   map[string]string // .beans-relative path -> expected content
		wantAbsent    []string
		wantUntouched map[string]string
	}{
		{
			name: "write applied, remove applied, untouched file survives",
			writes: map[string][]byte{
				"tp-cccc--c.md": []byte("content-c"),
			},
			removes:       []string{"tp-bbbb--b.md"},
			wantPresent:   map[string]string{"tp-cccc--c.md": "content-c"},
			wantAbsent:    []string{"tp-bbbb--b.md"},
			wantUntouched: map[string]string{"tp-aaaa--a.md": "content-a"},
		},
		{
			name:          "no writes/removes leaves tree unchanged",
			writes:        nil,
			removes:       nil,
			wantUntouched: map[string]string{"tp-aaaa--a.md": "content-a", "tp-bbbb--b.md": "content-b"},
		},
		{
			name: "non-bean file (e.g. archive/) survives the swap",
			writes: map[string][]byte{
				"tp-cccc--c.md": []byte("content-c"),
			},
			wantUntouched: map[string]string{
				"tp-aaaa--a.md":       "content-a",
				"archive/old-note.md": "kept",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestCore(t, "tp-", map[string]string{
				"tp-aaaa--a.md": "content-a",
				"tp-bbbb--b.md": "content-b",
			})
			if _, ok := tt.wantUntouched["archive/old-note.md"]; ok {
				archiveDir := filepath.Join(c.Root(), "archive")
				if err := os.MkdirAll(archiveDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(archiveDir, "old-note.md"), []byte("kept"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			if err := c.stageAndSwap(tt.writes, tt.removes); err != nil {
				t.Fatal(err)
			}
			for relPath := range tt.wantPresent {
				wantContent := tt.wantPresent[relPath]
				got, err := os.ReadFile(filepath.Join(c.Root(), relPath))
				if err != nil {
					t.Errorf("write not applied at %q: %v", relPath, err)
					continue
				}
				if string(got) != wantContent {
					t.Errorf("write %q = %q, want %q", relPath, got, wantContent)
				}
			}
			for _, relPath := range tt.wantAbsent {
				if _, err := os.Stat(filepath.Join(c.Root(), relPath)); !os.IsNotExist(err) {
					t.Errorf("removed file %q still present", relPath)
				}
			}
			for relPath, wantContent := range tt.wantUntouched {
				got, err := os.ReadFile(filepath.Join(c.Root(), relPath))
				if err != nil {
					t.Errorf("untouched file %q missing: %v", relPath, err)
					continue
				}
				if string(got) != wantContent {
					t.Errorf("untouched file %q changed: %q, want %q", relPath, got, wantContent)
				}
			}
			// no staging/backup siblings left behind after success
			entries, err := os.ReadDir(c.repoRoot())
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				name := e.Name()
				if name != ".beans" && strings.HasPrefix(name, ".beans") {
					t.Errorf("leftover sibling: %s", name)
				}
			}
		})
	}
}

func equalStrSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
