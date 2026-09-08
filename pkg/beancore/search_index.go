package beancore

import (
	"encoding/hex"
	"hash/fnv"
	"os"
	"path/filepath"
)

// indexDir returns the directory used to persist this store's search index.
// It is always outside the store root (AC-04): writeGitignore only excludes
// .conversations/, so an index file inside c.root would be git-tracked and
// end up committed to the repo or the hull. The directory name is a hash of
// the store's absolute root path rather than the configured project name,
// so two different stores never collide even if they share (or lack) a
// project name -- ~/.beans/worktrees/<project>/ (the config-driven sibling
// convention for worktrees) already accepts that collision risk, but the
// index has no such existing contract to match and a hash is exact instead
// of merely conventional.
func (c *Core) indexDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(c.root)
	if err != nil {
		root = c.root
	}
	h := fnv.New64a()
	h.Write([]byte(root))
	id := hex.EncodeToString(h.Sum(nil))
	return filepath.Join(home, ".beans", "index", id), nil
}
