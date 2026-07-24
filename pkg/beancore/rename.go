package beancore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/hmans/beans/pkg/bean"
)

// RenameChange describes a single bean's rename (own file move and/or ID
// change). OldPath/NewPath are relative to the .beans dir, matching
// Bean.Path (e.g. "tp-aaaa--slug.md" or "epic-auth/tp-aaaa--slug.md").
type RenameChange struct {
	OldID, NewID     string
	OldPath, NewPath string
}

// RenamePlan is the dry-run result of a rename computation. Mode is one of
// "slug", "id", "prefix".
type RenamePlan struct {
	Mode        string
	Changes     []RenameChange
	RefUpdates  map[string]int // beanID -> number of ref fields rewritten
	NewPrefix   string
	ConfigWrite bool
}

// repoRoot returns the directory containing the .beans dir (used for the
// staging sibling dir and worktree/config lookups, not for bean paths).
func (c *Core) repoRoot() string { return filepath.Dir(c.root) }

// newBeanPath returns the .beans-relative path a bean should have after a
// rename, preserving any subdirectory it currently lives in (Bean.Path may
// be nested, e.g. "epic-auth/id--slug.md").
func newBeanPath(oldPath, newID, newSlug string) string {
	dir := filepath.Dir(oldPath) // "." for a flat file
	name := bean.BuildFilename(newID, newSlug)
	if dir == "." {
		return name
	}
	return filepath.Join(dir, name)
}

// PlanRenameSlug computes a dry-run plan to change a bean's slug. Exactly
// one of newSlug (explicit, "" clears it) or reslug (regenerate from Title)
// must be used — the caller enforces mutual exclusivity.
func (c *Core) PlanRenameSlug(id string, newSlug *string, reslug bool) (*RenamePlan, error) {
	c.mu.RLock()
	b, ok := c.beans[id]
	c.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("bean %q not found", id)
	}
	var slug string
	switch {
	case reslug:
		slug = bean.Slugify(b.Title)
	case newSlug != nil:
		slug = *newSlug
	default:
		return nil, fmt.Errorf("no slug source given")
	}
	return &RenamePlan{
		Mode: "slug",
		Changes: []RenameChange{{
			OldID: b.ID, NewID: b.ID,
			OldPath: b.Path, NewPath: newBeanPath(b.Path, b.ID, slug),
		}},
		RefUpdates: map[string]int{},
	}, nil
}

// ApplyRename executes a previously computed RenamePlan.
func (c *Core) ApplyRename(plan *RenamePlan) error {
	switch plan.Mode {
	case "slug":
		return c.applyRenameSlug(plan)
	case "id", "prefix":
		return c.applyRenameCascade(plan)
	default:
		return fmt.Errorf("unknown rename mode %q", plan.Mode)
	}
}

func (c *Core) applyRenameSlug(plan *RenamePlan) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := plan.Changes[0]
	oldAbs := filepath.Join(c.root, ch.OldPath) // OldPath/NewPath are .beans-relative
	newAbs := filepath.Join(c.root, ch.NewPath)
	if oldAbs == newAbs {
		return nil
	}
	if err := os.Rename(oldAbs, newAbs); err != nil {
		return fmt.Errorf("renaming slug file: %w", err)
	}
	if b, ok := c.beans[ch.OldID]; ok {
		b.Path = ch.NewPath
		_, b.Slug = bean.ParseFilename(filepath.Base(newAbs))
	}
	return nil
}

// rewriteRefs replaces mapped old IDs with their new IDs in b's relationship
// fields (Parent/Blocking/BlockedBy). It mutates b in place and returns the
// number of individual ref fields changed. A ref value not present in m is
// left unchanged; an empty Parent is never considered a match.
func rewriteRefs(b *bean.Bean, m map[string]string) int {
	changed := 0
	if nv, ok := m[b.Parent]; ok && b.Parent != "" {
		b.Parent = nv
		changed++
	}
	for i, id := range b.Blocking {
		if nv, ok := m[id]; ok {
			b.Blocking[i] = nv
			changed++
		}
	}
	for i, id := range b.BlockedBy {
		if nv, ok := m[id]; ok {
			b.BlockedBy[i] = nv
			changed++
		}
	}
	return changed
}

// applyRenameCascade handles Mode "id" and "prefix". Implemented in a
// follow-up task (single-ID rename / prefix rebrand); stubbed here so the
// package compiles for the slug-rename path.
func (c *Core) applyRenameCascade(plan *RenamePlan) error {
	return fmt.Errorf("cascade rename not yet implemented")
}

// copyTree recursively copies src into dst, skipping any src-relative path
// present in skip (a directory match skips the whole subtree).
func copyTree(src, dst string, skip map[string]bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relToSrc, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if skip[relToSrc] {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, relToSrc)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}

// stageAndSwap builds a new .beans tree in a temp sibling dir (a full clone
// of the current tree with removes/writes applied), then atomically swaps it
// in for c.root. writes/removes keys are .beans-relative (matching
// Bean.Path, e.g. "x.md" or "epic/x.md"). Any error before the swap leaves
// the original .beans tree untouched and removes the staging dir.
func (c *Core) stageAndSwap(writes map[string][]byte, removes []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	repo := c.repoRoot()
	staging, err := os.MkdirTemp(repo, ".beans-staging-*")
	if err != nil {
		return fmt.Errorf("creating staging dir: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(staging)
		}
	}()

	// skip set: removed files, keyed .beans-relative (== copyTree's walk keys)
	skip := map[string]bool{}
	for _, r := range removes {
		skip[filepath.Clean(r)] = true
	}
	if err := copyTree(c.root, staging, skip); err != nil {
		return fmt.Errorf("cloning tree: %w", err)
	}
	for relPath, content := range writes {
		target := filepath.Join(staging, relPath)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("staging write dir: %w", err)
		}
		if err := os.WriteFile(target, content, 0644); err != nil {
			return fmt.Errorf("staging write: %w", err)
		}
	}

	// atomic swap
	backup := filepath.Join(repo, fmt.Sprintf(".beans.bak-%d", time.Now().UnixNano()))
	if err := os.Rename(c.root, backup); err != nil {
		return fmt.Errorf("backing up .beans: %w", err)
	}
	if err := os.Rename(staging, c.root); err != nil {
		os.Rename(backup, c.root) // best-effort rollback
		return fmt.Errorf("swapping in new tree: %w", err)
	}
	cleanup = false
	os.RemoveAll(backup)
	return nil
}
