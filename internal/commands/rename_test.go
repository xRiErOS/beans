package commands

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/xRiErOS/beans/pkg/beancore"
	"github.com/xRiErOS/beans/pkg/config"
)

func renameTestCore(t *testing.T, prefix string, files map[string]string) *beancore.Core {
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
	cfg.SetConfigDir(repo)
	c := beancore.New(beansDir, cfg)
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestBuildRenamePlan_dispatch(t *testing.T) {
	seed := map[string]string{
		"tp-aaaa--a.md": "---\n# tp-aaaa\ntitle: A\nstatus: todo\ntype: task\n---\n",
	}
	tests := []struct {
		name     string
		args     []string
		flags    renameFlags
		wantMode string
		wantErr  bool
	}{
		{"slug", []string{"tp-aaaa"}, renameFlags{slug: "x", slugSet: true}, "slug", false},
		{"noSlug", []string{"tp-aaaa"}, renameFlags{noSlug: true}, "slug", false},
		{"reslug", []string{"tp-aaaa"}, renameFlags{reslug: true}, "slug", false},
		{"newID", []string{"tp-aaaa", "tp-zzzz"}, renameFlags{}, "id", false},
		{"suffix", []string{"tp-aaaa"}, renameFlags{suffix: "k7x2", suffixSet: true}, "id", false},
		{"prefix", nil, renameFlags{prefix: "op-", prefixSet: true}, "prefix", false},
		{"mutual-excl slug+newid", []string{"tp-aaaa", "tp-zzzz"}, renameFlags{slug: "x", slugSet: true}, "", true},
		{"prefix with positional", []string{"tp-aaaa"}, renameFlags{prefix: "op-", prefixSet: true}, "", true},
		{"no args no prefix", nil, renameFlags{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := renameTestCore(t, "tp-", seed)
			plan, err := buildRenamePlan(c, tt.args, tt.flags)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got plan %+v", plan)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if plan.Mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", plan.Mode, tt.wantMode)
			}
		})
	}
}

func TestBuildRenamePlan_suffixWrongPrefixErrors(t *testing.T) {
	c := renameTestCore(t, "tp-", map[string]string{
		"other-aaaa--a.md": "---\n# other-aaaa\ntitle: A\nstatus: todo\ntype: task\n---\n",
	})
	if _, err := buildRenamePlan(c, []string{"other-aaaa"}, renameFlags{suffix: "k7x2", suffixSet: true}); err == nil {
		t.Fatal("expected error: --suffix on an id lacking the configured prefix")
	}
}

// TestApplyRenameWithGuards_reChecksGuardsBeforeApply is the I01 regression
// test: for a "prefix" rebrand, the server/worktree guards must be
// re-checked immediately before the real apply/staging (not only at Plan
// time), closing the TOCTOU window between building the plan (e.g. while a
// user considers an interactive --yes confirm) and the actual cascade swap.
// We simulate "a server started after planning" by binding the configured
// port AFTER a valid plan was built, then assert applyRenameWithGuards
// refuses instead of applying.
func TestApplyRenameWithGuards_reChecksGuardsBeforeApply(t *testing.T) {
	c := renameTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "---\n# tp-aaaa\ntitle: A\nstatus: todo\ntype: task\n---\n",
	})

	plan, err := buildRenamePlan(c, nil, renameFlags{prefix: "op-", prefixSet: true})
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a server starting between plan and apply by occupying the
	// configured port now.
	port := c.Config().GetServerPort()
	ln, lerr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if lerr != nil {
		t.Skipf("cannot bind configured port %d: %v", port, lerr)
	}
	defer ln.Close()

	if err := applyRenameWithGuards(c, plan); err == nil {
		t.Fatal("expected refusal: guard must be re-checked immediately before apply (I01)")
	}

	// Bean must NOT have been rebranded on disk (guard closed the TOCTOU window).
	if _, err := c.Get("op-aaaa"); err == nil {
		t.Error("apply proceeded despite guard re-check — TOCTOU window not closed")
	}
	if _, err := c.Get("tp-aaaa"); err != nil {
		t.Error("original bean missing after refused apply — plan must not have mutated state")
	}
}

// TestApplyRenameWithGuards_nonPrefixModeSkipsGuards verifies that id/slug
// modes are not subject to the prefix-only D05 guards (they apply cleanly
// even while a listener occupies the configured port).
func TestApplyRenameWithGuards_nonPrefixModeSkipsGuards(t *testing.T) {
	c := renameTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "---\n# tp-aaaa\ntitle: A\nstatus: todo\ntype: task\n---\n",
	})
	plan, err := buildRenamePlan(c, []string{"tp-aaaa"}, renameFlags{slug: "new-slug", slugSet: true})
	if err != nil {
		t.Fatal(err)
	}

	port := c.Config().GetServerPort()
	ln, lerr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if lerr == nil {
		defer ln.Close()
	}

	if err := applyRenameWithGuards(c, plan); err != nil {
		t.Fatalf("slug rename should not be blocked by prefix-only guards: %v", err)
	}
}
