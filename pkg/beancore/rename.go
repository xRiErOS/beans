package beancore

import (
	"fmt"
	"os"
	"path/filepath"

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

// applyRenameCascade handles Mode "id" and "prefix". Implemented in a
// follow-up task (single-ID rename / prefix rebrand); stubbed here so the
// package compiles for the slug-rename path.
func (c *Core) applyRenameCascade(plan *RenamePlan) error {
	return fmt.Errorf("cascade rename not yet implemented")
}
