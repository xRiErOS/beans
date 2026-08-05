package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/config"
)

func TestResolveBeansPath(t *testing.T) {
	// Create a valid beans directory for tests that need one
	tmpDir := t.TempDir()
	validBeansDir := filepath.Join(tmpDir, ".beans")
	if err := os.MkdirAll(validBeansDir, 0755); err != nil {
		t.Fatalf("failed to create test .beans dir: %v", err)
	}

	altBeansDir := filepath.Join(tmpDir, "alt-beans")
	if err := os.MkdirAll(altBeansDir, 0755); err != nil {
		t.Fatalf("failed to create alt beans dir: %v", err)
	}

	// Config that points to the valid beans dir
	cfg := config.Default()
	cfg.SetConfigDir(tmpDir)

	t.Run("flag takes highest precedence", func(t *testing.T) {
		t.Setenv("BEANS_PATH", altBeansDir)

		got, err := resolveBeansPath(validBeansDir, cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		if got != validBeansDir {
			t.Errorf("expected flag path %q, got %q", validBeansDir, got)
		}
	})

	t.Run("flag overrides env var", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "/nonexistent/should/not/be/used")

		got, err := resolveBeansPath(validBeansDir, cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		if got != validBeansDir {
			t.Errorf("expected flag path %q, got %q", validBeansDir, got)
		}
	})

	t.Run("env var used when flag is empty", func(t *testing.T) {
		t.Setenv("BEANS_PATH", altBeansDir)

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		if got != altBeansDir {
			t.Errorf("expected env var path %q, got %q", altBeansDir, got)
		}
	})

	t.Run("config used when flag and env var are empty", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		expected := cfg.ResolveBeansPath()
		if got != expected {
			t.Errorf("expected config path %q, got %q", expected, got)
		}
	})

	t.Run("invalid flag path returns error", func(t *testing.T) {
		_, err := resolveBeansPath("/nonexistent/path", cfg)
		if err == nil {
			t.Fatal("expected error for invalid flag path, got nil")
		}
		if !strings.Contains(err.Error(), "does not exist or is not a directory") {
			t.Errorf("expected 'does not exist' error, got %q", err.Error())
		}
	})

	t.Run("invalid env var path returns error", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "/nonexistent/env/path")

		_, err := resolveBeansPath("", cfg)
		if err == nil {
			t.Fatal("expected error for invalid env var path, got nil")
		}
		if !strings.Contains(err.Error(), "does not exist or is not a directory") {
			t.Errorf("expected 'does not exist' error, got %q", err.Error())
		}
	})

	t.Run("invalid config path returns init suggestion", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		// Config pointing to a nonexistent directory
		badCfg := config.Default()
		badCfg.SetConfigDir("/nonexistent/config/dir")

		_, err := resolveBeansPath("", badCfg)
		if err == nil {
			t.Fatal("expected error for invalid config path, got nil")
		}
		if !strings.Contains(err.Error(), "beans init") {
			t.Errorf("expected error to suggest 'beans init', got %q", err.Error())
		}
	})

	t.Run("file path rejected as not a directory", func(t *testing.T) {
		// Create a regular file (not a directory)
		filePath := filepath.Join(tmpDir, "not-a-dir")
		if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := resolveBeansPath(filePath, cfg)
		if err == nil {
			t.Fatal("expected error for file path (not directory), got nil")
		}
		if !strings.Contains(err.Error(), "does not exist or is not a directory") {
			t.Errorf("expected 'does not exist' error, got %q", err.Error())
		}
	})
}

// evalSymlinks resolves a path so comparisons survive macOS's /var -> /private/var
// symlink: git always reports the resolved path, t.TempDir() does not.
func evalSymlinks(t *testing.T, p string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", p, err)
	}
	return resolved
}

// initWorktreePair creates a git repo with a secondary worktree, gives both a
// .beans directory, and returns the two resolved roots.
func initWorktreePair(t *testing.T) (mainRoot, worktreeRoot string) {
	t.Helper()

	dir := t.TempDir()
	setup := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "initial"},
	}
	for _, args := range setup {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v failed: %s: %v", args, out, err)
		}
	}

	wt := filepath.Join(t.TempDir(), "secondary")
	cmd := exec.Command("git", "worktree", "add", wt, "-b", "feature")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git worktree add failed: %s: %v", out, err)
	}

	for _, d := range []string{dir, wt} {
		if err := os.MkdirAll(filepath.Join(d, ".beans"), 0755); err != nil {
			t.Fatalf("failed to create .beans in %q: %v", d, err)
		}
	}

	return evalSymlinks(t, dir), evalSymlinks(t, wt)
}

// TestResolveBeansPathAnchor covers the repo-root anchor: the store follows the
// repository the config belongs to, not the working tree the caller happens to
// stand in. This is the capability that BEANS_PATH used to stand in for.
func TestResolveBeansPathAnchor(t *testing.T) {
	mainRoot, worktreeRoot := initWorktreePair(t)

	t.Run("repo-root anchor resolves to the main worktree store", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		cfg := config.Default()
		cfg.Beans.Anchor = config.AnchorRepoRoot
		cfg.SetConfigDir(worktreeRoot)

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		want := filepath.Join(mainRoot, ".beans")
		if got != want {
			t.Errorf("expected main worktree store %q, got %q", want, got)
		}
	})

	t.Run("no anchor keeps the worktree-local store", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		cfg := config.Default()
		cfg.SetConfigDir(worktreeRoot)

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		want := filepath.Join(worktreeRoot, ".beans")
		if got != want {
			t.Errorf("expected worktree-local store %q, got %q", want, got)
		}
	})

	t.Run("repo-root anchor is a no-op in the main worktree", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		cfg := config.Default()
		cfg.Beans.Anchor = config.AnchorRepoRoot
		cfg.SetConfigDir(mainRoot)

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		want := filepath.Join(mainRoot, ".beans")
		if got != want {
			t.Errorf("expected main store %q, got %q", want, got)
		}
	})

	t.Run("repo-root anchor outside a git repo falls back to the config dir", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		plain := t.TempDir()
		if err := os.MkdirAll(filepath.Join(plain, ".beans"), 0755); err != nil {
			t.Fatalf("failed to create .beans: %v", err)
		}

		cfg := config.Default()
		cfg.Beans.Anchor = config.AnchorRepoRoot
		cfg.SetConfigDir(plain)

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		want := filepath.Join(plain, ".beans")
		if got != want {
			t.Errorf("expected config-dir store %q, got %q", want, got)
		}
	})

	t.Run("env var still outranks the anchor", func(t *testing.T) {
		t.Setenv("BEANS_PATH", filepath.Join(worktreeRoot, ".beans"))

		cfg := config.Default()
		cfg.Beans.Anchor = config.AnchorRepoRoot
		cfg.SetConfigDir(worktreeRoot)

		got, err := resolveBeansPath("", cfg)
		if err != nil {
			t.Fatalf("resolveBeansPath() error = %v", err)
		}
		want := filepath.Join(worktreeRoot, ".beans")
		if got != want {
			t.Errorf("expected env var store %q, got %q", want, got)
		}
	})

	t.Run("unknown anchor value is rejected", func(t *testing.T) {
		t.Setenv("BEANS_PATH", "")

		cfg := config.Default()
		cfg.Beans.Anchor = "repo-rooot"
		cfg.SetConfigDir(worktreeRoot)

		_, err := resolveBeansPath("", cfg)
		if err == nil {
			t.Fatal("expected error for unknown anchor value, got nil")
		}
		if !strings.Contains(err.Error(), "anchor") {
			t.Errorf("expected error to name the anchor setting, got %q", err.Error())
		}
	})
}
