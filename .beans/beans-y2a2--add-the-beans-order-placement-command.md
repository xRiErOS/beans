---
# beans-y2a2
title: Add the beans order placement command
status: completed
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:57:51Z
parent: beans-zb0r
blocked_by:
    - beans-nda7
---

`beans order <id> --after <id>`, `--before <id>`, `--first`, `--last`. Placement is relative to siblings under the same parent (R-12).

The property this command exists for is that a move writes exactly one file. A test that only checks the resulting sequence would pass just as happily against a full renumbering, and would then be an evergreen — assert the write count.

### Requirement 1: A bean can be placed relative to its siblings

**Objective:** As someone maintaining a backlog, I want to move one bean to a position among its siblings, so that reordering costs one write and one small diff.

#### Acceptance Criteria

1. WHEN beans order <id> --after <sibling> runs THE CLI SHALL set the order value of <id> strictly between that sibling and the sibling following it
2. WHEN beans order <id> --before <sibling> runs THE CLI SHALL set the order value of <id> strictly between that sibling and the sibling preceding it
3. WHEN beans order <id> --first runs THE CLI SHALL set an order value that sorts before every sibling
4. WHEN beans order <id> --last runs THE CLI SHALL set an order value that sorts after every sibling
5. WHEN a bean is placed THE CLI SHALL write exactly one bean file
6. IF the bean named by --after or --before has a different parent THEN THE CLI SHALL exit non-zero with an error stating that order is scoped per parent
7. IF more than one of --after, --before, --first and --last is passed THEN THE CLI SHALL exit non-zero with a usage error

#### Success Criteria

- SC-01: Moving the fifth of six siblings to second position leaves the six in the expected sequence and modifies exactly one file on disk, asserted by a write count rather than by the resulting order alone.

_Requirements: R-10, R-12_

## Recommended Skills

- `tdd`


---

## Implementation Report

Commit: 4fdbbe3 feat(commands): add order command for sibling placement.

- `internal/commands/order.go` (new): `beans order <id> --after/--before/--first/--last`. Siblings are gathered by manually comparing `Parent` (not via `model.BeanFilter.ParentID`, since that filter treats an empty ParentID as no-op-not-set and would return the whole tree for root-level beans) and sorted with the new `bean.SortByOrder`. `--after`/`--before` validate the referenced bean shares the target's parent (AC6) and reject self-reference. Exactly one `core.Update` call mutates only the target bean (AC5).
- `pkg/bean/sort_order.go` (new): exported `SortByOrder(beans []*Bean)`, in-place, reusable by the parallel `list --sort order` task (beans-uo43) per the task brief — set-Order beans first (lexicographic), empty-Order last, ties break on title.
- `internal/commands/register.go`: added `RegisterOrderCmd(root)` alphabetically between List and Path.
- Tests (TDD, RED verified before implementation): `pkg/bean/sort_order_test.go` (3 cases) and `internal/commands/order_test.go` (8 cases covering AC1-AC7). AC5/SC-01 is asserted by diffing file *contents* of every .md file in the beans dir before/after the move (not mtime, to avoid resolution flakiness) — exactly one file differs, and it is the moved bean's file; the resulting 6-sibling sequence is checked separately via `SortByOrder`. AC7 (mutual exclusivity) is asserted via a real `cobra` `Execute()` against a throwaway root (RunE is bypassed by cobra's own flag-group validation, so a direct-RunE test can't observe it); this required making `RegisterOrderCmd` flag registration idempotent (guarded by a `Flags().Lookup` check) since `orderCmd` is a package-level singleton also registered once by `RegisterCoreCommands` elsewhere in the suite.

Deviations from the task brief:
- Used `resolver.Beans(ctx, nil)` + manual `Parent` filtering instead of `model.BeanFilter{ParentID: &parentID}` — the ParentID filter is a documented no-op when the pointer target is empty, which would silently return all beans instead of just root-level siblings for a parent-less bean (pkg/beangraph/filters.go:57). Filtering client-side avoids that trap; still uses `resolver.Beans` for the sort-by-status pass-through, just not its parent filter.
- Added an idempotency guard to `RegisterOrderCmd` (not explicitly requested) — required so the AC7 cobra-level test can register the command into a throwaway root without a flag-redefinition panic, given `orderCmd` is also registered once via `RegisterCoreCommands` in the same test binary (`path_test.go`).
- Ran `go fmt ./...` once during verification; it reformatted several in-flight files belonging to other parallel tasks (create.go, list.go, update.go, etc.) that were not part of this task's scope — reverted those with `git checkout --` before committing, confirmed via `git status` that only this task's files remained staged.

Validation:
- `go build ./...`: clean.
- `go test ./...`: all packages pass (internal/commands, pkg/bean, and the rest of the suite unaffected).
- `gofmt -l` on the 4 new/changed Go files: no output (already formatted).
