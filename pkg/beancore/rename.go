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

// PlanRenameID computes a dry-run plan to rename a single bean's ID,
// cascading the change into every referencing bean's Parent/Blocking/
// BlockedBy fields. Refuses (no mutation) if newID collides with an existing
// bean.
func (c *Core) PlanRenameID(oldID, newID string) (*RenamePlan, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if oldID == newID {
		return nil, fmt.Errorf("new ID equals old ID")
	}
	b, ok := c.beans[oldID]
	if !ok {
		return nil, fmt.Errorf("bean %q not found", oldID)
	}
	if _, exists := c.beans[newID]; exists {
		return nil, fmt.Errorf("ID collision: %q already exists", newID)
	}
	return c.planCascade("id", map[string]string{oldID: newID}, map[string]*bean.Bean{oldID: b})
}

// planCascade builds a RenamePlan for an ID map (oldID -> newID). renamed
// holds only the beans whose own ID changes (their filename is recomputed
// via newBeanPath, preserving any subdir); every bean in c.beans is scanned
// for ref-field hits against idMap. Must be called with at least a read lock
// held.
func (c *Core) planCascade(mode string, idMap map[string]string, renamed map[string]*bean.Bean) (*RenamePlan, error) {
	plan := &RenamePlan{Mode: mode, RefUpdates: map[string]int{}}
	for oldID, b := range renamed {
		newID := idMap[oldID]
		plan.Changes = append(plan.Changes, RenameChange{
			OldID: oldID, NewID: newID,
			OldPath: b.Path,
			NewPath: newBeanPath(b.Path, newID, b.Slug),
		})
	}
	for _, b := range c.beans {
		if n := countRefHits(b, idMap); n > 0 {
			plan.RefUpdates[b.ID] = n
		}
	}
	return plan, nil
}

// countRefHits counts how many of b's ref fields (Parent/Blocking/BlockedBy)
// reference an old ID present in m, without mutating b.
func countRefHits(b *bean.Bean, m map[string]string) int {
	n := 0
	if _, ok := m[b.Parent]; ok && b.Parent != "" {
		n++
	}
	for _, id := range b.Blocking {
		if _, ok := m[id]; ok {
			n++
		}
	}
	for _, id := range b.BlockedBy {
		if _, ok := m[id]; ok {
			n++
		}
	}
	return n
}

// applyRenameCascade is the shared apply path for Mode "id" and "prefix": it
// re-renders every bean touched by the plan's ID map — the renamed bean(s)
// themselves (new filename, corrected "# id" comment via Render) plus every
// referencing bean (rewritten ref fields) — and drives the whole change
// through stageAndSwap for atomicity. After a successful swap it refreshes
// in-memory state from disk so the same Core resolves the new ID(s)
// immediately (Get(newID) works without a separate Load()).
func (c *Core) applyRenameCascade(plan *RenamePlan) error {
	idMap := map[string]string{}
	renamedIDs := map[string]bool{}
	for _, ch := range plan.Changes {
		idMap[ch.OldID] = ch.NewID
		renamedIDs[ch.OldID] = true
	}

	c.mu.RLock()
	writes := map[string][]byte{}
	removes := []string{}
	for _, b := range c.beans {
		// Work on a shallow clone so live state is never mutated pre-swap —
		// if rendering or staging fails partway through, c.beans is untouched.
		clone := *b
		clone.Blocking = append([]string(nil), b.Blocking...)
		clone.BlockedBy = append([]string(nil), b.BlockedBy...)

		idChanged := renamedIDs[b.ID]
		refChanged := rewriteRefs(&clone, idMap) > 0
		if !idChanged && !refChanged {
			continue
		}
		if idChanged {
			clone.ID = idMap[b.ID]
		}
		content, err := clone.Render()
		if err != nil {
			c.mu.RUnlock()
			return fmt.Errorf("rendering %s: %w", b.ID, err)
		}
		if idChanged {
			// remove old-named file, write new-named file (subdir preserved)
			removes = append(removes, b.Path)
			writes[newBeanPath(b.Path, clone.ID, clone.Slug)] = content
		} else {
			// same filename, new content (ref-only change)
			writes[b.Path] = content
		}
	}
	c.mu.RUnlock()

	if err := c.stageAndSwap(writes, removes); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loadFromDisk()
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

// swapRename performs the second (staging-into-c.root) rename in
// stageAndSwap. It is a package-level variable — rather than a bare
// os.Rename call — purely so tests can inject a deterministic failure here
// and assert the rollback branch, without depending on an OS-level race
// between the two renames. Production code always uses the real os.Rename.
var swapRename = os.Rename

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
	if err := swapRename(staging, c.root); err != nil {
		os.Rename(backup, c.root) // best-effort rollback
		return fmt.Errorf("swapping in new tree: %w", err)
	}
	cleanup = false
	os.RemoveAll(backup)
	return nil
}
