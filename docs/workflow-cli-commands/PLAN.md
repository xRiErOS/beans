---
type: Plan
title: "PLAN — workflow-cli-commands"
description: "Implementation plan for beans-mmyp: six workflow CLI commands, prime docs update, release cut"
tags:
  - tpic
  - workflow-cli-commands
timestamp: 2026-08-10T12:49:21Z
---

# workflow-cli-commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add six intuitive `beans` CLI wrapper commands (`complete`, `scrap`, `start`, `next`, `milestones`, `progress`) on top of existing `update`/`list`/`show` machinery, then rewrite `internal/commands/prompt.tmpl` (`beans prime`) to document them authoritatively, then cut a minor release.

**Source of truth:** beans-mmyp (epic) and its children beans-0ajg, beans-r780, beans-jvkq, beans-p17z, beans-18db, beans-m364, beans-9m5y, beans-omoy. This PLAN.md is a working artifact for the subagent-driven-development tooling (`task-brief`, `review-package`) — the beans remain the authoritative work-state; update each bean's checklist/body as its task completes.

**Architecture:** Every new command is a standalone Cobra command in `internal/commands/`, following the exact structure of the existing `update`/`list`/`show` commands — package-level flag vars, a `var xCmd = &cobra.Command{...}`, a `RegisterXCmd(root *cobra.Command)` function called from `RegisterCoreCommands` (`internal/commands/register.go:11-27`). All status mutations go through `beangraph.CoreResolver.UpdateBean` (`pkg/beangraph/mutations.go:106`) — never touch `bean.Bean.Status` directly. `beans next` and `beans milestones`/`beans progress` reuse (not duplicate) existing filter/sort logic (`list.go`'s `--ready` filter, `bean.SortByStatusPriorityAndType`).

**Tech Stack:** Go 1.x, Cobra (CLI), `pkg/beangraph` (GraphQL-shaped resolver used directly by the CLI, no server round-trip — same pattern as `update.go:51`), `pkg/beancore`/`pkg/bean` (data model), `internal/output` (`--json`), `internal/ui` (styled plain-text). Table-driven-optional Go tests per `mise test`, mirroring `internal/commands/update_test.go`'s setup/reset-flags pattern.

## Global Constraints

- **Command skeleton:** package-level `var <name>Cmd = &cobra.Command{Use: "...", Short: "...", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {...}}` plus `func Register<Name>Cmd(root *cobra.Command) { <name>Cmd.Flags()...; root.AddCommand(<name>Cmd) }`, registered from `internal/commands/register.go:11-27` (alphabetically ordered call list). Full resulting order after all six insertions: `Archive, Check, Complete, Create, Delete, Graphql, Init, List, Milestones, Next, Order, Path, Prime, Progress, Rename, Roadmap, Scrap, Show, Start, Update, Version`.
- **Mutations only via `beangraph.CoreResolver`:** `resolver := &beangraph.CoreResolver{Core: core}` (pattern at `update.go:51`), then `resolver.UpdateBean(ctx, b.ID, model.UpdateBeanInput{...})` (`pkg/beangraph/mutations.go:106`, sets `b.Status = *input.Status` at mutations.go:126-128). Never assign `bean.Bean.Status` directly from a command.
- **Status literals:** `"in-progress"`, `"todo"`, `"draft"`, `"completed"`, `"scrapped"` (`pkg/config/config.go:32-38`, `DefaultStatuses`). Validate with `cfg.IsValidStatus(status)` (config.go:528) before building the mutation input, exactly like `update.go`'s guard; on failure return `cmdError(jsonMode, output.ErrValidation, "%s", err)` (`internal/commands/content.go:60`).
- **Body-append for `--summary`/`--reason`:** build `model.BodyModification{Append: &text}` (`pkg/beangraph/model/models_gen.go:183-190`) and set it on `input.BodyMod`, exactly like `update.go`'s `--body-append` path (update.go:208-231). Resolve the flag value through `resolveAppendContent(value string) (string, error)` (`content.go:167-176`) so `-` (stdin) and `bean.UnescapeBody` semantics come for free — do not hand-roll a second content resolver.
- **Output:** `--json` flag (pattern `update.go:40,337`) renders via `output.Success(b, msg)` (`internal/output/output.go:42`) / `output.SuccessMultiple` (output.go:70) / `output.Error` (output.go:94); plain text renders via `ui.Success.Render(...) + ui.ID.Render(b.ID) + " " + ui.Muted.Render(b.Path)` (`update.go:157-159`). No command in this plan introduces a third rendering path.
- **`beans ready` / `beans blocked` are NOT new commands** — they already exist as `list --ready` (`list.go:111-119`) and `list --is-blocked` (`list.go:107-109`). `beans next` (Task 4) reuses this filter by calling an extracted `applyReadyFilter(filter *model.BeanFilter)` helper (see Task 4) instead of re-typing the `ExcludeStatus`/`ExcludeImplicitTerminal` literals — verbatim duplication of that block is a review defect, not a style choice. Note the real shape: `list.go`'s `--ready` block (list.go:111-119) **mutates an already partially-built `*model.BeanFilter`** (constructed earlier at list.go:69-77 from `--status`/`--type`/etc. flags, then further mutated for `--has-parent`/`--is-blocked` at list.go:86-109) — it is not a standalone factory. The extracted helper must mutate a passed-in filter, not construct+return a fresh one.
- **Sorting:** any command that orders beans to pick "the most important one" or a stable list uses `bean.SortByStatusPriorityAndType(beans, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())` (call site `list.go:317`) — same helper, same argument order, everywhere, per `.claude/rules/beans.md` ("Sorting Consistency": all list-producing code paths must use this one sort).
- **Percent-complete math (Tasks 5 & 6):** `scrapped` beans are excluded from both numerator and denominator. `percent = completed / (total - scrapped)`. This is grounded in beans-mmyp's own `beans progress` example (`2 in-progress + 15 todo + 23 completed + 3 scrapped` → shown as `57%`; `23 / (2+15+23) = 57.5% ≈ 57%`, not `23/43`). Apply the same formula to `beans milestones`' per-milestone "N/M children completed" figure, where N/M = completed-descendants / (total-descendants − scrapped-descendants).
- **Children/descendant counting is new logic** — no reusable helper exists today (`roadmap.go:137-140` inlines a `parent → children` map local to roadmap rendering; it is a different feature and stays untouched by this plan). Tasks 5 and 6 share one new helper file (`internal/commands/childindex.go`) instead of each rolling their own traversal.
- **Test convention:** mirror `setupUpdateTest`/`resetUpdateFlags` (`internal/commands/update_test.go:20-61`) — a `setup<Name>Test(t *testing.T) *bean.Bean` that builds a `t.TempDir()` + `.beans/` + `beancore.New(...)` + `core.Load()`, swaps the package-level `core`/`cfg` globals with `t.Cleanup` restore, seeds beans via `core.Create(b)`; a `reset<Name>Flags(t *testing.T)` that saves/clears/restores the command's package-level flag vars; tests call `<name>Cmd.RunE(<name>Cmd, []string{...})` directly and assert via `core.Get(id)`.
- Always write/update tests for every change (`.claude/rules` + CLAUDE.md testing rule; TDD per `superpowers:test-driven-development` — write the failing test first for every new RunE branch).
- `mise test` (`internal/commands/...` at minimum) must pass after every task.

---

## File Structure

New files:
- `internal/commands/complete.go`, `internal/commands/complete_test.go` (Task 1)
- `internal/commands/scrap.go`, `internal/commands/scrap_test.go` (Task 2)
- `internal/commands/start.go`, `internal/commands/start_test.go` (Task 3)
- `internal/commands/next.go`, `internal/commands/next_test.go` (Task 4)
- `internal/commands/childindex.go`, `internal/commands/childindex_test.go` (Task 5, shared by Tasks 5 & 6)
- `internal/commands/milestones.go`, `internal/commands/milestones_test.go` (Task 5)
- `internal/commands/progress.go`, `internal/commands/progress_test.go` (Task 6)

Modified files:
- `internal/commands/register.go` — add six `Register<Name>Cmd(root)` calls (Tasks 1-6)
- `internal/commands/list.go` — extract the `--ready` filter block (list.go:111-119) into `func applyReadyFilter(filter *model.BeanFilter)` (Task 4), used by both `listCmd.RunE` (mutating its already-built `filter`) and `nextCmd.RunE` (mutating a fresh `&model.BeanFilter{}`)
- `internal/commands/prime.go`, `internal/commands/prompt.tmpl`, `internal/commands/prime_test.go` — recipes rewrite (Task 7)

Signatures consumed from existing code (verified):
- `beangraph.CoreResolver{Core: core}` — instantiation pattern, `update.go:51`
- `(*CoreResolver).Bean(ctx, id string) (*bean.Bean, error)` — `pkg/beangraph/queries.go:13`
- `(*CoreResolver).Beans(ctx, filter *model.BeanFilter) ([]*bean.Bean, error)` — `pkg/beangraph/queries.go:22`
- `(*CoreResolver).UpdateBean(ctx, id string, input model.UpdateBeanInput, opts ...beancore.UpdateOption) (*bean.Bean, error)` — `pkg/beangraph/mutations.go:106`
- `model.UpdateBeanInput{Status *string, BodyMod *BodyModification, ...}` — `pkg/beangraph/model/models_gen.go:318-345`
- `model.BodyModification{Replace []*ReplaceOperation, Append *string}` — `models_gen.go:183-190`
- `(*config.Config).IsValidStatus(status string) bool` — `pkg/config/config.go:528`
- `(*config.Config).StatusNames() []string` / `PriorityNames()` / `TypeNames()` — used `list.go:317`
- `(*config.Config).IsArchiveStatus(name string) bool` — `config.go:583` (true for `completed` and `scrapped`)
- `bean.SortByStatusPriorityAndType(beans []*Bean, statusNames, priorityNames, typeNames []string)` — `pkg/bean/sort.go:12`
- `resolveAppendContent(value string) (string, error)` — `internal/commands/content.go:167-176`
- `cmdError(jsonMode bool, code string, format string, args ...any) error` — `content.go:60`
- `output.Success(b *bean.Bean, message string) error` / `SuccessMultiple` / `SuccessMessage` / `Error` — `internal/output/output.go:42,70,77,94`
- `ui.Success`, `ui.ID`, `ui.Muted` render helpers — used `update.go:157-159`
- `Bean.ID, Slug, Path, Title, Status, Type, Priority, Parent string` — `pkg/bean/bean.go:137-171`
- `RegisterCoreCommands(root *cobra.Command)` call list — `internal/commands/register.go:11-27`

---

## Task 1: `beans complete <id>` command

**Files:**
- Create: `internal/commands/complete.go`
- Test: `internal/commands/complete_test.go`
- Modify: `internal/commands/register.go` (add `RegisterCompleteCmd(root)`)

**Behavior (from beans-0ajg):**
- `beans complete <id> [--summary <text>] [--json]`
- Sets status to `completed` via `resolver.UpdateBean` with `input.Status = &"completed"`.
- Optional `--summary`: appends `"## Summary of Changes\n\n" + resolveAppendContent(summary)` via `input.BodyMod = &model.BodyModification{Append: &text}` — same call shape as `update.go --body-append`, heading text matches the convention already documented in `prompt.tmpl:24` ("update it with a `## Summary of Changes` section").
- Prints a confirmation with the bean title (plain: `ui.Success.Render("Completed ") + ui.ID.Render(b.ID) + " " + b.Title`; `--json`: `output.Success(b, "Bean completed")`).

**Steps:**
1. Write `complete_test.go` first (TDD): `TestCompleteSetsStatus`, `TestCompleteWithSummaryAppendsSection`, `TestCompleteRejectsUnknownID`, `TestCompleteJSONOutput`.
2. Implement `complete.go` to make tests pass, following the Global Constraints skeleton exactly.
3. Wire `RegisterCompleteCmd(root)` into `register.go` (alpha position: before `RegisterCreateCmd`).
4. `mise test ./internal/commands/...`.

**Acceptance:** `beans complete beans-abc --summary "Implemented via PR #42"` sets status and appends the summary section; bare `beans complete beans-abc` sets status with no body change; unknown ID errors via `cmdError`.

---

## Task 2: `beans scrap <id>` command

**Files:**
- Create: `internal/commands/scrap.go`
- Test: `internal/commands/scrap_test.go`
- Modify: `internal/commands/register.go` (add `RegisterScrapCmd(root)`)

**Behavior (from beans-r780):**
- `beans scrap <id> --reason <text> [--json]`
- `--reason` is **required** — enforce with `scrapCmd.MarkFlagRequired("reason")` at registration time, plus (since `MarkFlagRequired` only fires cobra's own arg-parsing error path) an explicit empty-string check inside `RunE` returning `cmdError(jsonMode, output.ErrValidation, "reason is required")` so the `--json` error path is also covered by a JSON-shaped error rather than cobra's raw usage error.
- Sets status to `scrapped`.
- Appends `"## Reason for Scrapping\n\n" + resolveAppendContent(reason)` the same way Task 1 appends the summary (matches `prompt.tmpl`'s existing "When SCRAPPING a bean, update it with a `## Reasons for Scrapping` section" convention — note the plan/epic body says `## Reason for Scrapping` (singular); use the epic body's exact heading verbatim since Task 7 will reconcile the template wording against whatever this task ships).
- Confirmation message.

**Steps:**
1. Write `scrap_test.go` first: `TestScrapRequiresReason` (both flag-missing and empty-string paths), `TestScrapSetsStatusAndAppendsReason`, `TestScrapRejectsUnknownID`, `TestScrapJSONOutput`.
2. Implement `scrap.go`.
3. Wire `RegisterScrapCmd(root)` into `register.go` (alpha position: before `RegisterShowCmd`).
4. `mise test ./internal/commands/...`.

**Acceptance:** `beans scrap beans-abc --reason "Superseded by beans-xyz"` sets status and appends the reason section; omitting `--reason` errors (both plain and `--json` modes) without mutating the bean.

---

## Task 3: `beans start <id>` command

**Files:**
- Create: `internal/commands/start.go`
- Test: `internal/commands/start_test.go`
- Modify: `internal/commands/register.go` (add `RegisterStartCmd(root)`)

**Behavior (from beans-jvkq):**
- `beans start <id> [--json]`
- Sets status to `in-progress`.
- Then displays the bean like `beans show` — **reuse, don't reimplement**: after the mutation succeeds, set the package-level `showJSON = startJSON` (both commands live in package `commands`; `showJSON` is `show.go`'s package-level `--json` flag var, declared in the `var (...)` block near the top of the file — confirm the exact line when editing) and `return showCmd.RunE(showCmd, []string{b.ID})`. `showCmd.RunE` is a plain field on the `*cobra.Command` value, directly callable from another file in the same package.
- Do not implement the "warn if another bean is already in-progress" possibility mentioned as optional in beans-jvkq's body — out of scope per "Could optionally" wording; do not add it speculatively (YAGNI).

**Steps:**
1. Write `start_test.go` first: `TestStartSetsStatusInProgress`, `TestStartDisplaysBeanDetails` (assert on captured stdout containing the title/status), `TestStartRejectsUnknownID`.
2. Implement `start.go`.
3. Wire `RegisterStartCmd(root)` into `register.go` (alpha position: before `RegisterUpdateCmd`).
4. `mise test ./internal/commands/...`.

**Acceptance:** `beans start beans-abc` sets status to `in-progress` and prints the same output `beans show beans-abc` would.

---

## Task 4: `beans next` command (+ `list.go` filter extraction)

**Files:**
- Create: `internal/commands/next.go`
- Test: `internal/commands/next_test.go`
- Modify: `internal/commands/list.go` (extract `readyFilter()`)
- Modify: `internal/commands/register.go` (add `RegisterNextCmd(root)`)

**Behavior (from beans-p17z):**
- `beans next [--json]`
- Returns the highest-priority `todo` bean that is not blocked, and displays it like `beans show` (same delegation pattern as Task 3: set `showJSON = nextJSON`, then `showCmd.RunE(showCmd, []string{b.ID})`).
- If nothing is ready, print a plain-text suggestion to check `beans list --is-blocked` / `beans list` (no such bean exists yet to `show`); `--json` mode returns `output.SuccessMessage("no ready beans found", nil)` or equivalent empty-result JSON shape (match existing `output` helpers, do not invent a new response shape).

**Steps:**
1. In `list.go`, extract lines 111-119 into:
   ```go
   // applyReadyFilter mutates filter in place to express "--ready": not
   // blocked, excludes in-progress/completed/scrapped/draft, and excludes
   // beans with implicit terminal status from a scrapped/completed ancestor.
   func applyReadyFilter(filter *model.BeanFilter) {
       isBlocked := false
       excludeImplicitTerminal := true
       filter.IsBlocked = &isBlocked
       filter.ExcludeStatus = append(filter.ExcludeStatus, "in-progress", "completed", "scrapped", "draft")
       filter.ExcludeImplicitTerminal = &excludeImplicitTerminal
   }
   ```
   and replace `list.go`'s inline `if listReady { ... }` body (list.go:111-119) with `if listReady { applyReadyFilter(filter) }`, keeping the existing `filter` variable (built at list.go:69-77 and further mutated at list.go:86-109) — do not construct a second filter or drop the flags already applied to it.
2. Write `next_test.go` first: `TestNextReturnsHighestPriorityReady`, `TestNextRespectsBlockedExclusion`, `TestNextReportsNoneReady` (plain + `--json`).
3. Implement `next.go`: build `filter := &model.BeanFilter{}`, call `applyReadyFilter(filter)`, `resolver.Beans(ctx, filter)` → `bean.SortByStatusPriorityAndType(beans, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())` → take `beans[0]` → delegate to `showCmd.RunE`.
4. Wire `RegisterNextCmd(root)` into `register.go` (alpha position: before `RegisterOrderCmd`).
5. `mise test ./internal/commands/...` — run the full `list_test.go` suite too, since `list.go` was modified.

**Acceptance:** `beans next` shows the top ready bean by the project's standard sort order; with no ready beans, it suggests `beans blocked`/`beans list` instead of erroring.

---

## Task 5: `beans milestones` command (+ shared child-index helper)

**Files:**
- Create: `internal/commands/childindex.go`, `internal/commands/childindex_test.go`
- Create: `internal/commands/milestones.go`, `internal/commands/milestones_test.go`
- Modify: `internal/commands/register.go` (add `RegisterMilestonesCmd(root)`)

**Behavior (from beans-18db):**
- `beans milestones [--all] [--json]`
- Lists all beans with `Type == "milestone"`.
- Shows progress per milestone as `"N/M children completed"` using the Global Constraints percent formula (descendants, not just direct children — a milestone's tasks are typically two levels down via epics, per this very epic's own shape: beans-mmyp epic → task beans-0ajg etc., with beans-mmyp itself parented under a milestone).
- Sorted by priority (reuse `bean.SortByStatusPriorityAndType`).
- Excludes completed/scrapped milestones by default; `--all` includes them.

**Steps:**
1. Write `childindex_test.go` first, then implement `childindex.go`:
   ```go
   // buildChildrenIndex maps each bean ID to its direct children.
   func buildChildrenIndex(all []*bean.Bean) map[string][]*bean.Bean {
       children := make(map[string][]*bean.Bean)
       for _, b := range all {
           if b.Parent != "" {
               children[b.Parent] = append(children[b.Parent], b)
           }
       }
       return children
   }

   // descendants returns every bean transitively parented under id (not including id itself).
   func descendants(id string, idx map[string][]*bean.Bean) []*bean.Bean {
       var out []*bean.Bean
       queue := idx[id]
       for len(queue) > 0 {
           b := queue[0]
           queue = queue[1:]
           out = append(out, b)
           queue = append(queue, idx[b.ID]...)
       }
       return out
   }

   // descendantProgress returns (completed, total) descendants per the
   // project's percent-complete convention: scrapped beans are excluded
   // from both completed and total.
   func descendantProgress(id string, idx map[string][]*bean.Bean, cfg *config.Config) (completed, total int) {
       for _, d := range descendants(id, idx) {
           if d.Status == "scrapped" {
               continue
           }
           total++
           if d.Status == "completed" {
               completed++
           }
       }
       return completed, total
   }
   ```
   Table-driven tests: empty tree, single level, multi-level, cycle-safety is not a concern here (beans already prevent cycles via `Core.DetectCycle`, per `.claude/rules/beans.md`), all-scrapped subtree (total=0 — decide and test the 0/0 display, e.g. render as `"0/0"` not `"NaN%"`).
2. Write `milestones_test.go`: `TestMilestonesListsByType`, `TestMilestonesShowsProgress`, `TestMilestonesExcludesCompletedScrappedByDefault`, `TestMilestonesAllFlagIncludesThem`, `TestMilestonesJSONOutput`.
3. Implement `milestones.go`: `resolver.Beans(ctx, nil)` → filter `Type == "milestone"` (+ status filter unless `--all`) → `bean.SortByStatusPriorityAndType` → for each, `descendantProgress` via the shared `childindex.go` helper → render `"<title> (<N>/<M> completed)"` plain, or JSON array of `{bean, completed, total}`.
4. Wire `RegisterMilestonesCmd(root)` into `register.go` (alpha position: `..., List, Milestones, Next, Order, ...` — `Milestones` sits immediately before `Next`, both before `Order`).
5. `mise test ./internal/commands/...`.

**Acceptance:** `beans milestones` lists non-terminal milestones sorted by the standard order, each annotated with completed/total descendant counts computed per the documented formula; `--all` includes completed/scrapped milestones.

---

## Task 6: `beans progress` command

**Files:**
- Create: `internal/commands/progress.go`, `internal/commands/progress_test.go`
- Modify: `internal/commands/register.go` (add `RegisterProgressCmd(root)`)

**Behavior (from beans-m364):**
- `beans progress [--parent <id>] [--json]`
- Shows counts by status across all statuses configured (`cfg.StatusNames()`/`cfg.StatusList()` — do not hardcode the four statuses shown in the epic's example; a project can define custom statuses, per `pkg/config`).
- Shows a percent-complete figure using the Global Constraints formula (`completed / (total - scrapped)`).
- Optional `--parent <id>`: scope the counts to `descendants(id, buildChildrenIndex(all))` (Task 5's shared helper) instead of every bean in the workspace — this satisfies beans-m364's "Optional: filter by milestone/epic to see progress on specific initiatives" without inventing a second traversal.
- Plain-text output includes a simple ASCII/unicode progress bar (e.g. `━` characters), matching the epic's example rendering; `--json` returns the raw counts/percent without any bar rendering (bars are a presentation concern, not data).

**Steps:**
1. Write `progress_test.go` first: `TestProgressCountsByStatus`, `TestProgressComputesPercent` (verify the `23/40=57%` example from the epic body as a literal test case), `TestProgressParentFlagScopesToDescendants`, `TestProgressJSONOutputHasNoBarString`.
2. Implement `progress.go`.
3. Wire `RegisterProgressCmd(root)` into `register.go` (alpha position: `..., Prime, Progress, Rename, ...` — `Progress` sits immediately after `RegisterPrimeCmd`, before `RegisterRenameCmd`; NOT next to `Order`/`Milestones`/`Next` despite the shared "reporting" theme — cobra registration order is alphabetical by command name, not by feature grouping).
4. `mise test ./internal/commands/...`.

**Acceptance:** `beans progress` matches the epic's worked example given the same input counts; `beans progress --parent beans-mmyp` scopes to that bean's descendants only.

---

## Task 7: `beans prime` recipes rewrite (beans-9m5y)

**Depends on:** Tasks 1-6 (all six commands must exist and be registered before this task starts — do not begin Task 7 until `mise test` is green with all six commands merged).

**Files:**
- Modify: `internal/commands/prompt.tmpl`
- Modify: `internal/commands/prime.go` (resolve the dead `GraphQLSchema` field)
- Modify: `internal/commands/prime_test.go`

**Scope (verbatim from beans-9m5y):**
- Consolidate the scattered recipe prose (the `<EXTREMELY_IMPORTANT>` task-lifecycle block at `prompt.tmpl:1-39`, plus the "Finding Work" section at `prompt.tmpl:28-38`) into a single `## Recipes` section: use-case → command, one line each, using `beans start <id>`, `beans complete <id>`, `beans scrap <id> --reason <text>`, `beans next` instead of `beans update <id> -s in-progress`/`-s completed` etc.
- Verify every use-case mentioned in the template maps to a command that actually exists (by this point, all six do).
- Document `beans ready` / `beans blocked` as the existing `list --ready` / `list --is-blocked` flags (they are not first-class commands — confirmed by this plan's scope: no sibling task promotes them).
- Resolve the dead `GraphQLSchema` field (`prime.go:16-21`, populated `prime.go:49`, never referenced in `prompt.tmpl` per the zero-match grep) — either render it in the template (if there is a genuine use for agents to see the schema inline) or remove the field from `promptData` and its population call. Prefer removal unless a concrete need surfaces during this task — do not keep dead plumbing on the theory it might be useful (YAGNI).
- Add substring assertions to `prime_test.go` per the established pattern (`prime_test.go:34-44`: render the template, `strings.Contains(out, want)` per expected string) — one assertion per new recipe command (`beans start`, `beans complete`, `beans scrap`, `beans next`, `beans milestones`, `beans progress`, `list --ready`, `list --is-blocked`).

**Not in scope:** building the six commands (Tasks 1-6, already done by the time this starts); cutting the release (Task 8).

**Acceptance (verbatim from beans-9m5y):**
- [ ] `prompt.tmpl` has one `## Recipes` section covering: starting work, finding work, completing work, scrapping work, checking progress, listing milestones, handling blocked work
- [ ] No recipe references a command that does not exist in this binary
- [ ] `GraphQLSchema` field is either rendered in the template or removed from `promptData`/`prime.go`
- [ ] `prime_test.go` has a substring assertion per new recipe command

**Steps:**
1. Write the new/updated assertions in `prime_test.go` first (TDD on the documentation contract itself).
2. Rewrite `prompt.tmpl`'s `<EXTREMELY_IMPORTANT>` block into the single `## Recipes` section.
3. Resolve `GraphQLSchema` in `prime.go` (render or remove).
4. `mise test ./internal/commands/...`.

---

## Task 8: Cut the release (beans-omoy)

**Depends on:** Tasks 1-7 all complete and merged.

**Scope (verbatim from beans-omoy):**
- Confirm all sibling tasks in this epic are completed (check beans-mmyp's children via `beans list --json --parent beans-mmyp` or `beans show`).
- Run `mise run release:minor` (workflow commands are new user-facing capability → minor bump per semver, per `mise.toml:95-103`).
- Verify `beans version` reports the new tag (`internal/version`, ldflags wired in `mise.toml:56-58`).
- `git push && git push --tags` — **ONLY WITH ERIK'S EXPLICIT GO-AHEAD AT THAT TIME.** Do not push as an implied consequence of this task being reached in the plan; stop and ask, presenting the new tag/version for confirmation first.

**Not in scope:** any code changes — this is purely the release/tag step; deciding whether push happens automatically (it never does implicitly).

**Steps:**
1. Confirm Tasks 1-7 are merged to the branch this release is cut from.
2. `mise run release:minor`.
3. Verify `beans version`.
4. STOP. Report the new version/tag to Erik and ask explicitly before `git push`/`git push --tags`.
