# beans-rename-command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `beans rename` CLI command that changes bean slugs, single bean IDs (with ref cascade), and project-wide ID prefixes, so overlong prefixes like `bew_BeWiki-Python-Download-` can be shortened to `bew-`.

**Architecture:** All three modes run as **direct-`beancore.Core` operations** inside a new Cobra command — the CLI builds its own `Core` per invocation exactly like every existing command (`internal/commands/root.go:56-59`); there is **no GraphQL/server round-trip** in the CLI path (verified: no GraphQL client exists in `internal/commands/`). Slug-rename is a single-file `os.Rename`. Single-ID rename and prefix-rebrand both cascade refs across all beans and therefore share **one atomic staging+swap primitive** (build a complete new `.beans/` tree in a temp sibling dir, then atomic directory swap with backup). A `renameBean` GraphQL mutation is **optional/deferred** (UI-only, Task 9) and not on the critical path.

**Tech Stack:** Go 1.x, Cobra (CLI), `pkg/bean` (pure ID/slug/filename transforms, no I/O), `pkg/beancore` (disk I/O + in-memory `beans` map), `pkg/config` (`.beans.yml` read/write), `internal/worktree` (worktree detection). Table-driven Go tests (`mise test`).

## Global Constraints

- **Direct-Core only for the CLI** — no CLI→GraphQL client layer (PO decision, revises SPEC D04). Each command builds its own `Core` via `beancore.New(root, cfg)` + `core.Load()`.
- **No etag/staleness check for rename** — D06's "etag on the live-mutation path" is **N/A** under direct-Core: the CLI is a single-shot process (`Load()` → plan → apply → exit), so there is no in-process staleness window. Cross-process concurrency is covered structurally: prefix-rebrand **refuses** while a server runs (Task 7 port-probe), and slug/single-ID renames write to disk where a running server's file watcher merges them via the existing dirty-merge flow (Worktree State Architecture, CLAUDE.md). **PO-confirm item at the freigabe gate** (reviewer Q01): accept dropping etag for single-ID rename on this rationale.
- **Crash cleanup (deferred, I03)** — an interrupted rebrand between the atomic swap and `os.RemoveAll(backup)` can leave an orphan `.beans.bak-*` sibling. Out of scope for this plan; a future `Load()`-time sweep of stale `.beans.bak-*` dirs can reclaim it. Not a correctness risk (the live `.beans/` is always the swapped-in tree).
- **beans never commits/`git mv`** (D10) — plain filesystem operations; the user stages bean files. git detects renames by content.
- **External ID references** (commit messages, docs, SSTD) are **not** rewritten (D11) — out of scope, document only.
- **`--dry-run` on every mode** (D07); prefix-rebrand additionally requires `--yes` (or interactive confirm) before applying (D07).
- **Prefix-rebrand refuses** if `beans serve` is running (port-probe) OR active worktrees exist (D05).
- Cross-refs store the **full ID** in `parent`/`blocking`/`blocked_by` (`pkg/bean/bean.go:159-166`); the `# <id>` line in the body is a comment written by `Render()` (`bean.go:249`), ignored by `Parse()`.
- Filenames: `BuildFilename` always emits the modern `id--slug.md` (or `id.md`) form (`pkg/bean/id.go:92`). Renames **normalize** legacy `id.slug`/`id-slug` filenames to `id--slug` as a side effect (they set a fresh path).
- Slug is a separate field `Bean.Slug` (`yaml:"-"`), **not** in frontmatter and **never** in cross-refs — a slug change touches only the filename.
- Always write/update tests for every change (`.claude/rules` + CLAUDE.md testing rule). Use table-driven tests. E2E: none (no UI touched).

---

## File Structure

- **`pkg/bean/id.go`** (modify) — add pure, I/O-free transform helpers (`RebrandID`, `IDSuffix`). Table-driven unit tests in `pkg/bean/id_test.go`.
- **`pkg/beancore/rename.go`** (create) — the whole rename engine: `RenamePlan` type, `PlanRenameSlug`/`PlanRenameID`/`PlanRebrand` (compute, no writes → dry-run), `ApplyRename` (execute), the atomic staging+swap primitive, ref-rewrite helper, and the two guards. Same package as `Core`, so it accesses private fields (`root`, `config`, `beans`, `mu`).
- **`pkg/beancore/rename_test.go`** (create) — cascade integrity, atomicity (simulated pre-swap failure), guard refusal, `.beans.yml` consistency.
- **`internal/commands/rename.go`** (create) — Cobra command, flag parsing, dry-run rendering, `--yes` confirm, `--json`. Registered via `RegisterRenameCmd` added to `RegisterCoreCommands` in `internal/commands/register.go`.
- **`internal/commands/rename_test.go`** (create) — flag→mode dispatch, dry-run prints-no-mutation, collision error surfacing.
- **`internal/graph/schema.graphqls` + `pkg/beangraph/mutations.go`** (modify, **optional** Task 9) — `renameBean` mutation for the Beans UI (slug + single-ID only).

Signatures consumed from existing code (verified):
- `bean.ParseFilename(name string) (id, slug string)` — `pkg/bean/id.go:67`
- `bean.BuildFilename(id, slug string) string` — `pkg/bean/id.go:92`
- `bean.Slugify(title string) string` — `pkg/bean/id.go:100`
- `(*Core).All() []*bean.Bean` — `pkg/beancore/core.go:296`
- `(*Core).Root() string` — returns `c.root` (used `serve.go:182`)
- `(*Core).Get(id string) (*bean.Bean, error)` — `pkg/beancore/core.go` (used by resolvers)
- `beancore.New(root string, cfg *config.Config) *Core` — `core.go:76`
- `(*bean.Bean).Render() ([]byte, error)` — writes `# <id>` comment
- `Bean.ID`, `Bean.Slug`, `Bean.Path`, `Bean.Parent string`, `Bean.Blocking []string`, `Bean.BlockedBy []string` — `pkg/bean/bean.go:137-166`
- `(*config.Config).Save(dir string) error` — `pkg/config/config.go:343`
- `(*config.Config).ConfigDir() string` — repo root holding `.beans.yml` (used `serve.go:134`)
- `(*config.Config).GetServerPort() int` — `config.go:807`
- `(*config.Config).ResolveWorktreePath(projectName string) (string, error)` — `config.go:680`
- `config.Config.Beans.Prefix string` (yaml `prefix`) — `config.go:177`

---

## Task 1: Pure ID transform helpers (`pkg/bean`)

**Files:**
- Modify: `pkg/bean/id.go` (append two functions)
- Test: `pkg/bean/id_test.go`

**Interfaces:**
- Produces:
  - `func IDSuffix(id, prefix string) string` — returns `id` with `prefix` stripped (returns `id` unchanged if it lacks the prefix).
  - `func RebrandID(id, oldPrefix, newPrefix string) string` — returns `newPrefix + IDSuffix(id, oldPrefix)`.

- [ ] **Step 1: Write the failing test**

```go
// pkg/bean/id_test.go
package bean

import "testing"

func TestIDSuffix(t *testing.T) {
	tests := []struct {
		name, id, prefix, want string
	}{
		{"strips prefix", "bew_BeWiki-Python-Download-ljs5", "bew_BeWiki-Python-Download-", "ljs5"},
		{"no prefix match returns id", "abc-ljs5", "xyz-", "abc-ljs5"},
		{"empty prefix returns id", "abc-ljs5", "", "abc-ljs5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IDSuffix(tt.id, tt.prefix); got != tt.want {
				t.Errorf("IDSuffix(%q,%q) = %q, want %q", tt.id, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestRebrandID(t *testing.T) {
	tests := []struct {
		name, id, oldP, newP, want string
	}{
		{"long to short", "bew_BeWiki-Python-Download-ljs5", "bew_BeWiki-Python-Download-", "bew-", "bew-ljs5"},
		{"idempotent when already short", "bew-ljs5", "bew-", "bew-", "bew-ljs5"},
		{"id without old prefix keeps suffix", "ljs5", "bew_", "bew-", "bew-ljs5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RebrandID(tt.id, tt.oldP, tt.newP); got != tt.want {
				t.Errorf("RebrandID(%q,%q,%q) = %q, want %q", tt.id, tt.oldP, tt.newP, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/bean/ -run 'TestIDSuffix|TestRebrandID' -v`
Expected: FAIL — `undefined: IDSuffix`, `undefined: RebrandID`.

- [ ] **Step 3: Write minimal implementation**

```go
// append to pkg/bean/id.go (imports already include "strings")

// IDSuffix returns id with prefix removed. If id does not start with
// prefix (or prefix is empty), id is returned unchanged.
func IDSuffix(id, prefix string) string {
	if prefix == "" {
		return id
	}
	return strings.TrimPrefix(id, prefix)
}

// RebrandID rewrites id from oldPrefix to newPrefix, preserving the suffix.
func RebrandID(id, oldPrefix, newPrefix string) string {
	return newPrefix + IDSuffix(id, oldPrefix)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/bean/ -run 'TestIDSuffix|TestRebrandID' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/bean/id.go pkg/bean/id_test.go
git commit -m "feat(rename): pure ID transform helpers

- IDSuffix strips a prefix from a bean ID
- RebrandID rewrites prefix keeping the suffix

Refs: <epic-bean-id>"
```

---

## Task 2: RenamePlan type + slug-rename (single-file)

**Files:**
- Create: `pkg/beancore/rename.go`
- Test: `pkg/beancore/rename_test.go`

**Interfaces:**
- Produces:
  - `type RenameChange struct { OldID, NewID, OldPath, NewPath string }`
  - `type RenamePlan struct { Mode string; Changes []RenameChange; RefUpdates map[string]int; NewPrefix string; ConfigWrite bool }` — `Mode` is `"slug"|"id"|"prefix"`; `OldPath`/`NewPath` are `.beans/`-root-relative (relative to `c.root`, NOT repo-root — Subdir bleibt erhalten).
  - `func (c *Core) PlanRenameSlug(id string, newSlug *string, reslug bool) (*RenamePlan, error)` — `newSlug != nil` sets an explicit slug (empty string clears → `id.md`); `reslug` regenerates from title via `Slugify`. Exactly one of `newSlug`/`reslug` is used; the caller enforces mutual exclusivity.
  - `func (c *Core) ApplyRename(plan *RenamePlan) error` — for `Mode=="slug"` performs the single-file rename; other modes handled in Tasks 5/6 (ApplyRename dispatches on `plan.Mode`).

- [ ] **Step 1: Write the failing test**

```go
// pkg/beancore/rename_test.go
package beancore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/bean" // NOTE: verify module path via `head -1 go.mod`; adjust import if different
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
	cfg.SetConfigDir(repo) // ensure Save/ConfigDir target repo root; see note below
	c := New(beansDir, cfg)
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	return c
}

const minimalBean = `# %s
---
title: Test Bean
status: todo
type: task
---
Body.
`

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

func strPtr(s string) *string { return &s }
```

> **Note (test harness, verified):** module path is `github.com/hmans/beans` (`go.mod`). `config.DefaultWithPrefix(prefix) *Config` (`config.go:209`) and `(*Config).SetConfigDir(dir)` (`config.go:337`) both exist — no new API needed. This is test scaffolding.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/beancore/ -run TestApplyRenameSlug -v`
Expected: FAIL — `undefined: (*Core).PlanRenameSlug` / `undefined: (*Core).ApplyRename`.

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/beancore/rename.go
package beancore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hmans/beans/pkg/bean" // adjust to module path
)

type RenameChange struct {
	OldID, NewID     string
	OldPath, NewPath string // relative to the .beans dir, matching Bean.Path (e.g. "tp-aaaa--slug.md" or "epic-auth/tp-aaaa--slug.md")
}

type RenamePlan struct {
	Mode        string // "slug" | "id" | "prefix"
	Changes     []RenameChange
	RefUpdates  map[string]int // beanID -> number of ref fields rewritten
	NewPrefix   string
	ConfigWrite bool
}

// repoRoot returns the directory containing the .beans dir (used for the
// staging sibling dir and worktree/config lookups, not for bean paths).
func (c *Core) repoRoot() string { return filepath.Dir(c.root) }

// newBeanPath returns the .beans-relative path a bean should have after a
// rename, preserving any subdirectory it currently lives in (Bean.Path may be
// nested, e.g. "epic-auth/id--slug.md").
func newBeanPath(oldPath, newID, newSlug string) string {
	dir := filepath.Dir(oldPath) // "." for a flat file
	name := bean.BuildFilename(newID, newSlug)
	if dir == "." {
		return name
	}
	return filepath.Join(dir, name)
}

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
			OldPath: b.Path, NewPath: newBeanPath(b.Path, b.ID, slug), // .beans-relative
		}},
		RefUpdates: map[string]int{},
	}, nil
}

func (c *Core) ApplyRename(plan *RenamePlan) error {
	switch plan.Mode {
	case "slug":
		return c.applyRenameSlug(plan)
	case "id", "prefix":
		return c.applyRenameCascade(plan) // implemented in Task 5/6
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
```

> Add a stub so the package compiles before Task 5:
> ```go
> func (c *Core) applyRenameCascade(plan *RenamePlan) error {
> 	return fmt.Errorf("cascade rename not yet implemented")
> }
> ```
> Remove the stub when Task 5 provides the real body.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/beancore/ -run TestApplyRenameSlug -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/beancore/rename.go pkg/beancore/rename_test.go
git commit -m "feat(rename): RenamePlan type and slug-rename path

- RenamePlan/RenameChange dry-run model
- PlanRenameSlug + ApplyRename slug branch (single-file os.Rename)
- cascade branch stubbed for follow-up tasks

Refs: <epic-bean-id>"
```

---

## Task 3: Ref-rewrite helper + rename map (cascade computation)

**Files:**
- Modify: `pkg/beancore/rename.go`
- Test: `pkg/beancore/rename_test.go`

**Interfaces:**
- Produces:
  - `func rewriteRefs(b *bean.Bean, m map[string]string) int` — replaces every occurrence of a mapped old-ID in `b.Parent`/`b.Blocking`/`b.BlockedBy` with its new-ID; returns the number of ref fields changed. Pure over the passed bean (mutates it in place); no I/O.
  - `func (c *Core) buildRenameMap(mode, single-args…)` is folded into Tasks 5/6; this task delivers `rewriteRefs` and its test only.

- [ ] **Step 1: Write the failing test**

```go
func TestRewriteRefs(t *testing.T) {
	b := &bean.Bean{
		ID:        "x-2",
		Parent:    "old-1",
		Blocking:  []string{"old-1", "keep-9"},
		BlockedBy: []string{"other-5"},
	}
	m := map[string]string{"old-1": "new-1"}
	n := rewriteRefs(b, m)
	if n != 2 { // Parent + one Blocking entry
		t.Errorf("changed count = %d, want 2", n)
	}
	if b.Parent != "new-1" {
		t.Errorf("Parent = %q", b.Parent)
	}
	if b.Blocking[0] != "new-1" || b.Blocking[1] != "keep-9" {
		t.Errorf("Blocking = %v", b.Blocking)
	}
	if b.BlockedBy[0] != "other-5" {
		t.Errorf("BlockedBy = %v", b.BlockedBy)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/beancore/ -run TestRewriteRefs -v`
Expected: FAIL — `undefined: rewriteRefs`.

- [ ] **Step 3: Write minimal implementation**

```go
// append to pkg/beancore/rename.go

// rewriteRefs replaces mapped old IDs with their new IDs in b's relationship
// fields. Returns the number of individual ref fields changed.
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
```

> **Verify field types** against `pkg/bean/bean.go:159-166`: this assumes `Parent string`, `Blocking []string`, `BlockedBy []string`. If `Parent` is `*string` or a slice, adjust the first branch accordingly (the grounding shows scalar `parent` and list `blocking`/`blocked_by`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/beancore/ -run TestRewriteRefs -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/beancore/rename.go pkg/beancore/rename_test.go
git commit -m "feat(rename): ref-rewrite helper for ID cascade

Refs: <epic-bean-id>"
```

---

## Task 4: Atomic staging+swap primitive

**Files:**
- Modify: `pkg/beancore/rename.go`
- Test: `pkg/beancore/rename_test.go`

**Interfaces:**
- Produces:
  - `func (c *Core) stageAndSwap(writes map[string][]byte, removes []string) error` — builds a complete new `.beans/` tree in a temp sibling dir by cloning the current tree, then applies `removes` (`.beans`-relative paths to omit) and `writes` (`.beans`-relative path → new file content), then atomically swaps the temp dir in for `.beans/` (renaming the old tree to a `.bak-*` sibling first, removing it on success). Any error **before** the swap leaves `.beans/` untouched and the staging dir removed.
  - Internal helper `copyTree(src, dst string, skip map[string]bool) error`.

- [ ] **Step 1: Write the failing test**

```go
func TestStageAndSwap_atomicOnPreSwapFailure(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "content-a",
		"tp-bbbb--b.md": "content-b",
	})
	// A write whose parent dir is impossible to create would fail staging.
	// Force failure with a NUL byte in a path element (MkdirAll errors).
	badWrites := map[string][]byte{
		filepath.Join("\x00bad", "x.md"): []byte("nope"), // .beans-relative; NUL → MkdirAll error
	}
	err := c.stageAndSwap(badWrites, nil)
	if err == nil {
		t.Fatal("expected staging failure, got nil")
	}
	// original tree intact
	got, _ := os.ReadFile(filepath.Join(c.Root(), "tp-aaaa--a.md"))
	if string(got) != "content-a" {
		t.Errorf("original file corrupted: %q", got)
	}
	// no staging/backup siblings left behind
	entries, _ := os.ReadDir(c.repoRoot())
	for _, e := range entries {
		name := e.Name()
		if name != ".beans" && (len(name) > 6 && name[:6] == ".beans") {
			t.Errorf("leftover sibling: %s", name)
		}
	}
}

func TestStageAndSwap_appliesWritesAndRemoves(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "content-a",
		"tp-bbbb--b.md": "content-b",
	})
	writes := map[string][]byte{
		"tp-cccc--c.md": []byte("content-c"), // .beans-relative
	}
	removes := []string{"tp-bbbb--b.md"}
	if err := c.stageAndSwap(writes, removes); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(c.Root(), "tp-bbbb--b.md")); !os.IsNotExist(err) {
		t.Error("removed file still present")
	}
	if got, _ := os.ReadFile(filepath.Join(c.Root(), "tp-cccc--c.md")); string(got) != "content-c" {
		t.Errorf("write not applied: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(c.Root(), "tp-aaaa--a.md")); string(got) != "content-a" {
		t.Errorf("untouched file changed: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/beancore/ -run TestStageAndSwap -v`
Expected: FAIL — `undefined: (*Core).stageAndSwap`.

- [ ] **Step 3: Write minimal implementation**

```go
// append to pkg/beancore/rename.go
// add imports: "io", "time"

// copyTree recursively copies src into dst, skipping any src-relative path in skip.
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

// stageAndSwap builds a new .beans tree in a temp sibling dir, applies removes
// and writes, then atomically swaps it in. writes/removes keys are
// .beans-relative (matching Bean.Path, e.g. "x.md" or "epic/x.md"). On any
// pre-swap error the original .beans tree is untouched and the staging dir is
// removed.
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/beancore/ -run TestStageAndSwap -v`
Expected: PASS (both subtests).

- [ ] **Step 5: Commit**

```bash
git add pkg/beancore/rename.go pkg/beancore/rename_test.go
git commit -m "feat(rename): atomic staging+swap primitive

- copyTree clones the .beans tree with a skip set
- stageAndSwap writes new tree to temp sibling, atomic dir swap with backup
- pre-swap failure leaves original untouched

Refs: <epic-bean-id>"
```

---

## Task 5: Single-ID rename (cascade via staging+swap)

**Files:**
- Modify: `pkg/beancore/rename.go` (replace the `applyRenameCascade` stub; add `PlanRenameID`)
- Test: `pkg/beancore/rename_test.go`

**Interfaces:**
- Produces:
  - `func (c *Core) PlanRenameID(oldID, newID string) (*RenamePlan, error)` — collision-checks `newID` against existing beans, computes the own-file rename plus ref updates across all referencing beans. `Mode="id"`.
  - `func (c *Core) applyRenameCascade(plan *RenamePlan) error` — shared apply path for `id` and `prefix` modes: re-renders every affected bean and drives `stageAndSwap`.

- [x] **Step 1: Write the failing test**

```go
func TestRenameID_cascadesRefs(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--parent.md": "# tp-aaaa\n---\ntitle: Parent\nstatus: todo\ntype: epic\n---\n",
		"tp-bbbb--child.md":  "# tp-bbbb\n---\ntitle: Child\nstatus: todo\ntype: task\nparent: tp-aaaa\nblocked_by:\n  - tp-aaaa\n---\n",
	})
	plan, err := c.PlanRenameID("tp-aaaa", "tp-zzzz")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != "id" {
		t.Fatalf("mode = %q", plan.Mode)
	}
	if plan.RefUpdates["tp-bbbb"] != 2 { // parent + blocked_by
		t.Errorf("RefUpdates[tp-bbbb] = %d, want 2", plan.RefUpdates["tp-bbbb"])
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	// I01: the applied disk state matches the plan exactly.
	for _, ch := range plan.Changes {
		if _, err := os.Stat(filepath.Join(c.Root(), ch.NewPath)); err != nil {
			t.Errorf("planned NewPath %q missing after apply: %v", ch.NewPath, err)
		}
		if ch.OldPath != ch.NewPath {
			if _, err := os.Stat(filepath.Join(c.Root(), ch.OldPath)); !os.IsNotExist(err) {
				t.Errorf("planned OldPath %q still present after apply", ch.OldPath)
			}
		}
	}
	// B01: the SAME core must reflect the rename in-memory after apply.
	if _, err := c.Get("tp-zzzz"); err != nil {
		t.Errorf("same-core Get(new id) failed — in-memory state not refreshed: %v", err)
	}
	if _, err := c.Get("tp-aaaa"); err == nil {
		t.Error("same-core still resolves the old ID after rename")
	}
	// also verify what landed on disk via a fresh core
	c2 := New(c.Root(), c.config)
	if err := c2.Load(); err != nil {
		t.Fatal(err)
	}
	child, err := c2.Get("tp-bbbb")
	if err != nil {
		t.Fatal(err)
	}
	if child.Parent != "tp-zzzz" {
		t.Errorf("child.Parent = %q, want tp-zzzz", child.Parent)
	}
	if len(child.BlockedBy) != 1 || child.BlockedBy[0] != "tp-zzzz" {
		t.Errorf("child.BlockedBy = %v", child.BlockedBy)
	}
	if _, err := c2.Get("tp-zzzz"); err != nil {
		t.Errorf("renamed bean not found under new ID: %v", err)
	}
	if _, err := c2.Get("tp-aaaa"); err == nil {
		t.Error("old ID still resolvable")
	}
}

func TestRenameID_collisionRejected(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "# tp-aaaa\n---\ntitle: A\nstatus: todo\ntype: task\n---\n",
		"tp-bbbb--b.md": "# tp-bbbb\n---\ntitle: B\nstatus: todo\ntype: task\n---\n",
	})
	if _, err := c.PlanRenameID("tp-aaaa", "tp-bbbb"); err == nil {
		t.Fatal("expected collision error")
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/beancore/ -run TestRenameID -v`
Expected: FAIL — `undefined: (*Core).PlanRenameID` (and cascade stub error if reached).

- [x] **Step 3: Write minimal implementation**

```go
// in pkg/beancore/rename.go — remove the Task 2 stub and add:

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

// planCascade builds a RenamePlan for an ID map. renamed holds the beans whose
// own ID changes (for filename changes); all beans are scanned for ref updates.
func (c *Core) planCascade(mode string, idMap map[string]string, renamed map[string]*bean.Bean) (*RenamePlan, error) {
	plan := &RenamePlan{Mode: mode, RefUpdates: map[string]int{}}
	// own-file renames (.beans-relative paths, subdir preserved)
	for oldID, b := range renamed {
		newID := idMap[oldID]
		plan.Changes = append(plan.Changes, RenameChange{
			OldID: oldID, NewID: newID,
			OldPath: b.Path,
			NewPath: newBeanPath(b.Path, newID, b.Slug),
		})
	}
	// ref updates across all beans (dry-run count via a scratch copy)
	for _, b := range c.beans {
		n := countRefHits(b, idMap)
		if n > 0 {
			plan.RefUpdates[b.ID] = n
		}
	}
	return plan, nil
}

// countRefHits counts how many ref fields of b reference a mapped old ID,
// without mutating b.
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

// applyRenameCascade re-renders every affected bean and swaps the tree in.
func (c *Core) applyRenameCascade(plan *RenamePlan) error {
	// Build the ID map from the plan's own-file changes.
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
		// work on a shallow clone so we never mutate live state pre-swap
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
	// Refresh in-memory state: stageAndSwap only rewrote disk, but Core is the
	// authoritative in-memory view (CLAUDE.md). Re-read so c.beans reflects the
	// new IDs/refs — otherwise a caller (e.g. the Task 9 resolver) Get()ing the
	// new ID immediately after ApplyRename would miss it.
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loadFromDisk() // core.go:128 — clears and rebuilds c.beans from disk
}
```

> `clone.Render()` writes the `# <id>` comment from `clone.ID`, so the comment line is corrected automatically for renamed beans. `Bean` is a plain struct (`bean.go:137`, no embedded mutex), so the shallow copy plus fresh `Blocking`/`BlockedBy` slices is safe.

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/beancore/ -run 'TestRenameID|TestApplyRenameSlug|TestStageAndSwap|TestRewriteRefs' -v`
Expected: PASS (all).

- [x] **Step 5: Commit**

```bash
git add pkg/beancore/rename.go pkg/beancore/rename_test.go
git commit -m "feat(rename): single-ID rename with ref cascade

- PlanRenameID collision-checks and builds a cascade plan
- applyRenameCascade re-renders affected beans and swaps atomically
- # id comment corrected via Render on the renamed bean

Refs: <epic-bean-id>"
```

---

## Task 6: Prefix-rebrand (map all beans + `.beans.yml`)

**Files:**
- Modify: `pkg/beancore/rename.go` (add `PlanRebrand`; extend apply to write config)
- Test: `pkg/beancore/rename_test.go`

**Interfaces:**
- Produces:
  - `func (c *Core) PlanRebrand(newPrefix string) (*RenamePlan, error)` — maps every bean ID from the current prefix to `newPrefix` (suffix preserved via `bean.RebrandID`), builds the cascade plan, sets `NewPrefix` and `ConfigWrite=true`. `Mode="prefix"`.
  - `ApplyRename` for `Mode="prefix"` performs the cascade swap **then** writes `.beans.yml` with the new prefix.

- [ ] **Step 1: Write the failing test**

```go
func TestRebrand_remapsAllAndWritesConfig(t *testing.T) {
	c := newTestCore(t, "old_Long-Prefix-", map[string]string{
		"old_Long-Prefix-aaaa--a.md": "# old_Long-Prefix-aaaa\n---\ntitle: A\nstatus: todo\ntype: epic\n---\n",
		"old_Long-Prefix-bbbb--b.md": "# old_Long-Prefix-bbbb\n---\ntitle: B\nstatus: todo\ntype: task\nparent: old_Long-Prefix-aaaa\n---\n",
	})
	plan, err := c.PlanRebrand("op-")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode != "prefix" || !plan.ConfigWrite || plan.NewPrefix != "op-" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	if len(plan.Changes) != 2 {
		t.Errorf("changes = %d, want 2", len(plan.Changes))
	}
	if err := c.ApplyRename(plan); err != nil {
		t.Fatal(err)
	}
	// config prefix written
	cfg2, err := config.LoadFromDirectory(c.repoRoot()) // config.go:296 (dir loader)
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Beans.Prefix != "op-" {
		t.Errorf("config prefix = %q, want op-", cfg2.Beans.Prefix)
	}
	// beans reloadable under new IDs with intact refs
	c2 := New(c.Root(), cfg2)
	if err := c2.Load(); err != nil {
		t.Fatal(err)
	}
	child, err := c2.Get("op-bbbb")
	if err != nil {
		t.Fatal(err)
	}
	if child.Parent != "op-aaaa" {
		t.Errorf("child.Parent = %q, want op-aaaa", child.Parent)
	}
}

func TestRebrand_mixedPrefixRefused(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md":    "# tp-aaaa\n---\ntitle: A\nstatus: todo\ntype: task\n---\n",
		"other-bbbb--b.md": "# other-bbbb\n---\ntitle: B\nstatus: todo\ntype: task\n---\n",
	})
	if _, err := c.PlanRebrand("op-"); err == nil {
		t.Fatal("expected refusal on a mixed-prefix repo (B04 guard)")
	}
}
```

> **I02/collision note:** the `clash := c.beans[nid]` branch in `PlanRebrand` is defensive — with a uniform prefix (enforced by the B04 guard) suffixes are unique, so new IDs cannot collide. No separate rebrand collision test is added; `TestRenameID_collisionRejected` covers the single-ID collision path.

> **Note (verified):** `config.LoadFromDirectory(startDir) (*Config, error)` (`config.go:296`) walks up to find `.beans.yml`; `config.Load(configPath)` (`config.go:260`) takes a file path. Use the directory loader here.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/beancore/ -run TestRebrand -v`
Expected: FAIL — `undefined: (*Core).PlanRebrand`.

- [ ] **Step 3: Write minimal implementation**

```go
// add to pkg/beancore/rename.go

func (c *Core) PlanRebrand(newPrefix string) (*RenamePlan, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	oldPrefix := ""
	if c.config != nil {
		oldPrefix = c.config.Beans.Prefix
	}
	if newPrefix == oldPrefix {
		return nil, fmt.Errorf("new prefix equals current prefix %q", oldPrefix)
	}
	// add "strings" to the rename.go imports for the HasPrefix guard below
	idMap := map[string]string{}
	renamed := map[string]*bean.Bean{}
	for id, b := range c.beans {
		// B04 guard: never remap an ID that lacks the current prefix — RebrandID
		// would otherwise emit newPrefix+fullOldID (double-prefixed, corrupt).
		if oldPrefix != "" && !strings.HasPrefix(id, oldPrefix) {
			return nil, fmt.Errorf("bean %q does not start with the current prefix %q; refusing rebrand to avoid ID corruption", id, oldPrefix)
		}
		nid := bean.RebrandID(id, oldPrefix, newPrefix)
		if nid == id {
			continue
		}
		if _, clash := c.beans[nid]; clash {
			return nil, fmt.Errorf("rebrand collision: %q already exists", nid)
		}
		idMap[id] = nid
		renamed[id] = b
	}
	if len(idMap) == 0 {
		return nil, fmt.Errorf("no beans matched prefix %q", oldPrefix)
	}
	plan, err := c.planCascade("prefix", idMap, renamed)
	if err != nil {
		return nil, err
	}
	plan.NewPrefix = newPrefix
	plan.ConfigWrite = true
	return plan, nil
}
```

Extend `ApplyRename` (or `applyRenameCascade`) to persist config for prefix mode. Modify the dispatch:

```go
func (c *Core) ApplyRename(plan *RenamePlan) error {
	switch plan.Mode {
	case "slug":
		return c.applyRenameSlug(plan)
	case "id":
		return c.applyRenameCascade(plan)
	case "prefix":
		if err := c.applyRenameCascade(plan); err != nil {
			return err
		}
		return c.writeRebrandConfig(plan.NewPrefix)
	default:
		return fmt.Errorf("unknown rename mode %q", plan.Mode)
	}
}

func (c *Core) writeRebrandConfig(newPrefix string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.config == nil {
		return fmt.Errorf("no config to update")
	}
	c.config.Beans.Prefix = newPrefix
	if err := c.config.Save(c.config.ConfigDir()); err != nil {
		return fmt.Errorf("writing .beans.yml: %w", err)
	}
	return nil
}
```

> `ConfigDir()` returns the repo root that holds `.beans.yml` (set during load; `serve.go:134` uses `cfg.ConfigDir()`). Confirm it is populated after `config.Load`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/beancore/ -run 'TestRebrand|TestRenameID|TestApplyRenameSlug|TestStageAndSwap' -v`
Expected: PASS (all).

- [ ] **Step 5: Commit**

```bash
git add pkg/beancore/rename.go pkg/beancore/rename_test.go
git commit -m "feat(rename): project-wide prefix rebrand

- PlanRebrand maps all bean IDs to a new prefix (suffix preserved)
- ApplyRename writes new prefix to .beans.yml after the atomic swap

Refs: <epic-bean-id>"
```

---

## Task 7: Guards — active worktrees + running server

**Files:**
- Modify: `pkg/beancore/rename.go` (add guard functions; call them in `PlanRebrand`/apply)
- Test: `pkg/beancore/rename_test.go`

**Interfaces:**
- Produces:
  - `func (c *Core) checkNoActiveWorktrees() error` — errors if `~/.beans/worktrees/<project>/` contains any `*.meta.json` (an active worktree marker; see Worktree State Architecture in CLAUDE.md).
  - `func (c *Core) checkServerNotRunning() error` — errors if a TCP dial to `127.0.0.1:<GetServerPort()>` succeeds (heuristic; documented limitation).
  - Both are invoked at the top of `PlanRebrand` (fail fast, before any staging).

- [ ] **Step 1: Write the failing test**

```go
func TestGuard_activeWorktreeRefusesRebrand(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "# tp-aaaa\n---\ntitle: A\nstatus: todo\ntype: task\n---\n",
	})
	// Simulate an active worktree marker at the resolved worktree path.
	projectName := c.config.GetProjectName()
	if projectName == "" {
		projectName = filepath.Base(c.config.ConfigDir())
	}
	wtPath, err := c.config.ResolveWorktreePath(projectName)
	if err != nil {
		t.Skipf("cannot resolve worktree path: %v", err)
	}
	if err := os.MkdirAll(wtPath, 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(wtPath)
	if err := os.WriteFile(filepath.Join(wtPath, "tp-aaaa.meta.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := c.PlanRebrand("op-"); err == nil {
		t.Fatal("expected refusal due to active worktree")
	}
}

// requires imports "net" and "fmt" in the test file
func TestGuard_runningServerRefusesRebrand(t *testing.T) {
	c := newTestCore(t, "tp-", map[string]string{
		"tp-aaaa--a.md": "# tp-aaaa\n---\ntitle: A\nstatus: todo\ntype: task\n---\n",
	})
	port := c.config.GetServerPort()
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Skipf("cannot bind configured port %d: %v", port, err)
	}
	defer ln.Close()
	if _, err := c.PlanRebrand("op-"); err == nil {
		t.Fatal("expected refusal while a listener occupies the configured port")
	}
}
```

> This test writes into the real resolved worktree path (typically `~/.beans/worktrees/<tmpname>`). The project name is the temp repo's basename, so collisions with real projects are effectively impossible; the `defer os.RemoveAll` cleans it up. If the environment forbids writing under `$HOME`, the test skips.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/beancore/ -run TestGuard_activeWorktree -v`
Expected: FAIL — either `undefined: checkNoActiveWorktrees` or no error returned.

- [ ] **Step 3: Write minimal implementation**

```go
// add to pkg/beancore/rename.go
// add imports: "net", "strings", "time" (time already added in Task 4)

func (c *Core) checkNoActiveWorktrees() error {
	if c.config == nil {
		return nil
	}
	// Project name resolution mirrors serve.go:115-117 exactly: configured
	// project.name first, basename of the .beans.yml dir only as fallback.
	// (Using only the basename would check the wrong worktree dir when
	// project.name is set, silently letting a rebrand through — undermining D05.)
	projectName := c.config.GetProjectName() // config.go:802
	if projectName == "" {
		projectName = filepath.Base(c.config.ConfigDir())
	}
	wtPath, err := c.config.ResolveWorktreePath(projectName)
	if err != nil {
		return nil // cannot resolve → assume none
	}
	entries, err := os.ReadDir(wtPath)
	if err != nil {
		return nil // dir absent → none
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".meta.json") {
			return fmt.Errorf("active worktrees present under %s; merge or remove them before a prefix rebrand", wtPath)
		}
	}
	return nil
}

func (c *Core) checkServerNotRunning() error {
	if c.config == nil {
		return nil
	}
	port := c.config.GetServerPort()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return fmt.Errorf("beans serve appears to be running on port %d; stop it before a prefix rebrand", port)
	}
	return nil
}
```

Wire both into `PlanRebrand` (top, after acquiring/releasing the read lock — call them before `c.mu.RLock()` to avoid holding the lock across a network dial):

```go
func (c *Core) PlanRebrand(newPrefix string) (*RenamePlan, error) {
	if err := c.checkServerNotRunning(); err != nil {
		return nil, err
	}
	if err := c.checkNoActiveWorktrees(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	// ... existing body ...
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/beancore/ -run 'TestGuard|TestRebrand|TestRenameID' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/beancore/rename.go pkg/beancore/rename_test.go
git commit -m "feat(rename): rebrand guards for worktrees and server

- refuse prefix rebrand when active worktree meta files exist
- refuse when a server responds on the configured port

Refs: <epic-bean-id>"
```

---

## Task 8: CLI command `beans rename`

**Files:**
- Create: `internal/commands/rename.go`
- Modify: `internal/commands/register.go` (add `RegisterRenameCmd(root)` to `RegisterCoreCommands`)
- Test: `internal/commands/rename_test.go`

> **Registration convention (verified):** commands are **not** added in `root.go`. `internal/commands/register.go` → `RegisterCoreCommands(root)` calls one `Register<X>Cmd(root)` per command (e.g. `RegisterArchiveCmd`). Each command function does `root.AddCommand(cmd)` internally. The `core *beancore.Core` and `cfg *config.Config` are **package-level vars** (`root.go:12-13`) initialized in `PersistentPreRunE` (`root.go:26-56`) — commands access them directly (see `archive.go:22` `core.All()`, `archive.go:28` `cfg.IsArchiveStatus(...)`). There is **no** `loadCore(cmd)` helper.

**Interfaces:**
- Consumes: `PlanRenameSlug`, `PlanRenameID`, `PlanRebrand`, `ApplyRename` (Tasks 2/5/6), `RenamePlan`, and package-level `core`.
- Produces: `type renameFlags struct{...}`, `func buildRenamePlan(c *beancore.Core, args []string, f renameFlags) (*beancore.RenamePlan, error)` (the testable core of the command — maps flags+args to exactly one mode).
- CLI surface (SPEC §8):
  ```
  beans rename <id> --slug "neuer-slug"   # set slug
  beans rename <id> --no-slug             # clear slug → id.md
  beans rename <id> --reslug              # regenerate from title
  beans rename <id> <neue-id>             # full new ID
  beans rename <id> --suffix k7x2         # new suffix, keep prefix
  beans rename --prefix "bew-"            # project-wide prefix rebrand
  # cross-cutting: --dry-run, --yes, --json, --beans-path
  ```

- [ ] **Step 1: Write the failing test**

We test `buildRenamePlan` directly — it takes a `*beancore.Core` plus parsed flags/args, so it needs no cobra-execution harness (none exists in `internal/commands/*_test.go`). This covers mode dispatch (B02), mutual exclusivity (I05), and the `--suffix` prefix guard (I04).

```go
// internal/commands/rename_test.go
package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hmans/beans/pkg/beancore"
	"github.com/hmans/beans/pkg/config"
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
		"tp-aaaa--a.md": "# tp-aaaa\n---\ntitle: A\nstatus: todo\ntype: task\n---\n",
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
		"other-aaaa--a.md": "# other-aaaa\n---\ntitle: A\nstatus: todo\ntype: task\n---\n",
	})
	if _, err := buildRenamePlan(c, []string{"other-aaaa"}, renameFlags{suffix: "k7x2", suffixSet: true}); err == nil {
		t.Fatal("expected error: --suffix on an id lacking the configured prefix")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/commands/ -run TestBuildRenamePlan -v`
Expected: FAIL — `undefined: renameFlags` / `undefined: buildRenamePlan`.

- [ ] **Step 3: Write minimal implementation**

```go
// internal/commands/rename.go
package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/hmans/beans/pkg/beancore"
)

// RegisterRenameCmd registers the rename command (called from register.go).
func RegisterRenameCmd(root *cobra.Command) {
	var (
		slug    string
		noSlug  bool
		reslug  bool
		suffix  string
		prefix  string
		dryRun  bool
		yes     bool
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "rename [id] [new-id]",
		Short: "Rename a bean slug, a single bean ID (cascading refs), or the project-wide ID prefix",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// core is a package-level var, initialized in root.go PersistentPreRunE
			plan, err := buildRenamePlan(core, args, renameFlags{
				slug: slug, slugSet: cmd.Flags().Changed("slug"),
				noSlug: noSlug, reslug: reslug,
				suffix: suffix, suffixSet: cmd.Flags().Changed("suffix"),
				prefix: prefix, prefixSet: cmd.Flags().Changed("prefix"),
			})
			if err != nil {
				return err
			}
			if err := renderPlan(cmd, plan, jsonOut); err != nil {
				return err
			}
			if dryRun {
				return nil
			}
			if plan.Mode == "prefix" && !yes {
				if !confirm(cmd, fmt.Sprintf("Rebrand %d beans to prefix %q?", len(plan.Changes), plan.NewPrefix)) {
					fmt.Fprintln(cmd.OutOrStdout(), "aborted")
					return nil
				}
			}
			return core.ApplyRename(plan)
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "set the slug part of the filename")
	cmd.Flags().BoolVar(&noSlug, "no-slug", false, "remove the slug (filename becomes id.md)")
	cmd.Flags().BoolVar(&reslug, "reslug", false, "regenerate the slug from the bean title")
	cmd.Flags().StringVar(&suffix, "suffix", "", "replace only the ID suffix, keeping the prefix")
	cmd.Flags().StringVar(&prefix, "prefix", "", "project-wide: rebrand all bean IDs to this prefix")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned changes without applying them")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt (prefix rebrand)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit the plan as JSON")
	root.AddCommand(cmd)
}

type renameFlags struct {
	slug      string
	slugSet   bool
	noSlug    bool
	reslug    bool
	suffix    string
	suffixSet bool
	prefix    string
	prefixSet bool
}

// modeCount reports how many distinct rename modes the flags/args request.
func (f renameFlags) modeCount(argCount int) int {
	n := 0
	if f.prefixSet {
		n++
	}
	if f.slugSet || f.noSlug || f.reslug {
		n++
	}
	if f.suffixSet {
		n++
	}
	if argCount == 2 { // <id> <new-id>
		n++
	}
	return n
}

// buildRenamePlan maps flags+args to exactly one plan mode.
func buildRenamePlan(c *beancore.Core, args []string, f renameFlags) (*beancore.RenamePlan, error) {
	if f.modeCount(len(args)) > 1 {
		return nil, fmt.Errorf("conflicting options: choose exactly one of --slug/--no-slug/--reslug, <new-id>, --suffix, or --prefix")
	}

	if f.prefixSet {
		if len(args) > 0 {
			return nil, fmt.Errorf("--prefix takes no positional arguments")
		}
		return c.PlanRebrand(f.prefix)
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("provide a bean id, or use --prefix")
	}
	id := args[0]

	switch {
	case f.noSlug:
		empty := ""
		return c.PlanRenameSlug(id, &empty, false)
	case f.reslug:
		return c.PlanRenameSlug(id, nil, true)
	case f.slugSet:
		s := f.slug
		return c.PlanRenameSlug(id, &s, false)
	}

	if f.suffixSet {
		// I04: keep the bean's own prefix. It must be the configured prefix;
		// refuse otherwise rather than fabricate a corrupt ID.
		prefix := c.Config().Beans.Prefix
		if prefix != "" && !strings.HasPrefix(id, prefix) {
			return nil, fmt.Errorf("bean %q does not start with the configured prefix %q; cannot apply --suffix", id, prefix)
		}
		return c.PlanRenameID(id, prefix+f.suffix)
	}
	if len(args) == 2 {
		return c.PlanRenameID(id, args[1])
	}
	return nil, fmt.Errorf("nothing to rename: pass a new id, --suffix, or a slug flag")
}

// renderPlan prints the plan as a table (or JSON with --json).
func renderPlan(cmd *cobra.Command, plan *beancore.RenamePlan, jsonOut bool) error {
	w := cmd.OutOrStdout()
	if jsonOut {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(plan)
	}
	fmt.Fprintf(w, "mode: %s\n", plan.Mode)
	for _, ch := range plan.Changes {
		if ch.OldID != ch.NewID {
			fmt.Fprintf(w, "  %s -> %s  (%s -> %s)\n", ch.OldID, ch.NewID, ch.OldPath, ch.NewPath)
		} else {
			fmt.Fprintf(w, "  %s  (%s -> %s)\n", ch.OldID, ch.OldPath, ch.NewPath)
		}
	}
	if len(plan.RefUpdates) > 0 {
		total := 0
		for _, n := range plan.RefUpdates {
			total += n
		}
		fmt.Fprintf(w, "ref updates: %d across %d beans\n", total, len(plan.RefUpdates))
	}
	if plan.ConfigWrite {
		fmt.Fprintf(w, "config: .beans.yml prefix -> %q\n", plan.NewPrefix)
	}
	return nil
}

// confirm reads a y/N answer from the command's input stream.
func confirm(cmd *cobra.Command, prompt string) bool {
	fmt.Fprintf(cmd.OutOrStdout(), "%s [y/N] ", prompt)
	sc := bufio.NewScanner(cmd.InOrStdin())
	if !sc.Scan() {
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(sc.Text()))
	return ans == "y" || ans == "yes"
}
```

Register in `internal/commands/register.go` — add one line to `RegisterCoreCommands`:

```go
	RegisterRenameCmd(root) // add alongside the other Register<X>Cmd calls
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/commands/ -run TestBuildRenamePlan -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/commands/rename.go internal/commands/rename_test.go internal/commands/register.go
git commit -m "feat(rename): beans rename CLI command

- slug (--slug/--no-slug/--reslug), single-ID (<new-id>/--suffix), prefix (--prefix)
- --dry-run on all modes, --yes confirm + prompt for rebrand, --json plan output
- mutual-exclusivity check; --suffix guarded against wrong-prefix IDs

Refs: <epic-bean-id>"
```

---

## Task 9 (OPTIONAL / deferred): `renameBean` GraphQL mutation for the UI

> **PO decision:** the CLI path does **not** use GraphQL. This mutation exists solely so the Beans UI can trigger slug / single-ID renames. It is **not on the critical path** — implement only if the UI needs it. Operationalize as a separate low-priority bean.

**Files:**
- Modify: `internal/graph/schema.graphqls` (add `renameBean(input: RenameBeanInput!): Bean!` + `input RenameBeanInput { id: ID!, newSlug: String, reslug: Boolean, newID: String }`)
- Modify: `pkg/beangraph/mutations.go` (resolver: dispatch to `PlanRenameSlug`/`PlanRenameID` + `ApplyRename`)
- Run: `mise codegen` (regenerates `internal/graph/generated.go` + `frontend/src/lib/graphql/generated.ts`)
- Test: `pkg/beangraph/mutations_test.go`

Scope excludes prefix-rebrand (offline batch, refuses when the server runs — inherently not a live mutation).

- [ ] **Step 1:** Write failing resolver test mirroring `UpdateBean` tests in `pkg/beangraph/mutations_test.go`.
- [ ] **Step 2:** Run it — fails (`renameBean` undefined).
- [ ] **Step 3:** Add schema + input type; run `mise codegen`; implement the resolver:
  ```go
  func (r *CoreResolver) RenameBean(ctx context.Context, input model.RenameBeanInput) (*bean.Bean, error) {
      var plan *beancore.RenamePlan
      var err error
      switch {
      case input.NewID != nil:
          plan, err = r.Core.PlanRenameID(input.ID, *input.NewID)
      case input.Reslug != nil && *input.Reslug:
          plan, err = r.Core.PlanRenameSlug(input.ID, nil, true)
      case input.NewSlug != nil:
          plan, err = r.Core.PlanRenameSlug(input.ID, input.NewSlug, false)
      default:
          return nil, fmt.Errorf("renameBean: provide newSlug, reslug, or newID")
      }
      if err != nil {
          return nil, err
      }
      if err := r.Core.ApplyRename(plan); err != nil {
          return nil, err
      }
      return r.Core.Get(plan.Changes[0].NewID)
  }
  ```
- [ ] **Step 4:** Run `mise test` — resolver test passes; `mise codegen` leaves the tree clean.
- [ ] **Step 5:** Commit `feat(rename): renameBean GraphQL mutation for UI (Refs: <bean-id>)`.

---

## Task 10: Documentation

**Files:**
- Modify: `beans-src/CLAUDE.md` (or `README`) — document the `beans rename` convention and the non-goals.

- [ ] **Step 1:** Add a short "Renaming beans" section: the three modes, that renames are plain FS operations (the user stages the changed bean files — beans never commits), that external ID references (commit messages, docs) are **not** rewritten (D11), and that prefix-rebrand refuses while `beans serve` runs or active worktrees exist (D05).
- [ ] **Step 2:** `git add beans-src/CLAUDE.md && git commit -m "docs(rename): document beans rename modes and non-goals (Refs: <bean-id>)"`

---

## Full-suite gate (after all tasks)

- [ ] Run `mise test` — all Go tests pass.
- [ ] Run `mise build` — binary builds (frontend embed intact).
- [ ] Manual smoke on a **copy** of a real project (never the live one): `beans rename --prefix "bew-" --dry-run` shows the full ID/file/ref plan; without `--dry-run` and with `--yes` it rewrites all files + `.beans.yml`; a fresh `beans list` resolves every bean under the new prefix with intact parent/blocking relations.

---

## Self-Review (completed by planner)

**Spec coverage (SPEC §2–§9):**
- §2 Slug-rename → Task 2 · Einzel-ID → Task 5 · Prefix-rebrand → Task 6. ✅
- §5 Cascade algorithm (filename, `# id` comment via Render, parent/blocking/blocked_by) → Tasks 3+5. ✅
- §6 Atomicity (staging+swap offline; slug = single os.Rename) → Task 4 (+5/6). Live-mutation/etag path is **N/A** under the PO's direct-Core decision — documented in Global Constraints. ✅
- §7 Guards (server, worktrees), `--dry-run`, `--yes` → Tasks 7+8. ✅
- §8 CLI surface → Task 8 (all flags present). ✅
- §9 Tests (table-driven transform, cascade integrity, dry-run, atomicity, guards; no E2E) → Tasks 1–8 each ship tests. ✅
- §10 open finenesses: cascade-core location (pkg/bean transforms + pkg/beancore orchestration) ✅; server detection (port-probe, no lock exists) ✅; GraphQL form (deferred, Task 9) ✅; legacy-filename normalization (BuildFilename emits `id--slug`; renames set fresh path) ✅.

**Decision coverage:** D01–D03 (both levels + cascade) ✅ · D04 **revised** (direct-Core, not GraphQL-over-server — PO-approved) ✅ · D05 (worktree refuse) Task 7 ✅ · D06 (staging+swap) Task 4 ✅ · D07 (dry-run + confirm) Task 8 ✅ · D08 (slug sources) Task 2/8 ✅ · D09 (full-id + --suffix + collision) Task 5/8 ✅ · D10 (no git) Global Constraints ✅ · D11 (external refs out of scope) Task 10 ✅ · D12 (.beans.yml) Task 6 ✅.

**Verified during planning** (no longer open): module path `github.com/hmans/beans`; `config.DefaultWithPrefix`/`SetConfigDir`/`LoadFromDirectory`/`ConfigDir`/`Save`; `Core.Config()`/`Root()`; `Bean.Parent string` (scalar), `Blocking`/`BlockedBy []string`, `Path` is `.beans`-relative and may be nested (path model corrected accordingly); `GetServerPort`/`ResolveWorktreePath`.

**Open verification points flagged for the implementer** (each marked inline with `>` notes — resolve by reading the cited code, not by guessing): `Bean` shallow-copy safety in `applyRenameCascade` (plain struct at `bean.go:137`; noted resolved but confirm no lock is added upstream). Everything else is grounded to file:line.

**Review round 1 (ce-plan-reviewer) — findings addressed:**
- **B01** (Core in-memory not refreshed after cascade swap) → `applyRenameCascade` now calls `c.loadFromDisk()` after a successful swap; Task 5 test asserts same-core `Get(newID)` resolves.
- **B02** (Task 8 phantom helpers / no `loadCore`) → Task 8 rewritten to use package-level `core` + real `renderPlan`/`confirm`/`buildRenamePlan`; test drives `buildRenamePlan` directly (no nonexistent cobra harness).
- **B03** (wrong registration target) → registration moved to `register.go` `RegisterRenameCmd`, not `root.go` `AddCommand`.
- **B04** (mixed-prefix → double-prefix ID corruption) → `PlanRebrand` refuses any ID lacking the current prefix; `TestRebrand_mixedPrefixRefused` added.
- **B05** (worktree guard project name) → `checkNoActiveWorktrees` mirrors `serve.go:115-117` exactly: `cfg.GetProjectName()` first, `cfg.ConfigDir()` basename only as fallback (round-2 correction — the ConfigDir-only version would have missed worktrees when `project.name` is set, undermining D05).
- **B06** (no server-guard test) → `TestGuard_runningServerRefusesRebrand` added.
- **I01** (dry-run plan==result) → Task 5 test asserts applied disk paths equal `plan.Changes`.
- **I02** → rebrand collision branch documented as defensively unreachable under the B04 guard (no artificial test).
- **I04** (--suffix prefix source) → `buildRenamePlan` refuses `--suffix` on an ID lacking the configured prefix instead of fabricating one.
- **I05** (flag mutual exclusivity) → `renameFlags.modeCount` rejects >1 mode.
- **I03** (orphan `.beans.bak-*` after crash) → documented as deferred, non-correctness (Global Constraints).
- **Q01** (etag dropped for single-ID) → documented rationale in Global Constraints; **flagged as a PO-confirm item** at the freigabe gate.
- **Q02** (refresh after cascade) → resolved by the B01 fix.
