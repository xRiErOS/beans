---
# beans-uo43
title: Offer order as a key of list --sort
status: completed
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T11:02:52Z
parent: beans-zb0r
blocked_by:
    - beans-nda7
---

`beans list --sort` accepts created, updated, status, priority and id. Add order, scoped per parent (R-12), and say so in the help text — a sort key whose scope is not written down gets guessed wrong once and then relied upon.

### Requirement 1: Beans can be listed in manual order

**Objective:** As someone reading a backlog, I want to list beans in the order I placed them, so that the sequence I decided is the sequence I see.

#### Acceptance Criteria

1. WHEN list --sort order runs THE CLI SHALL sort beans by their fractional index within their parent group
2. IF a bean carries no order value THEN THE CLI SHALL place it after every bean of the same parent that carries one
3. THE help text of --sort SHALL name order among the accepted keys and SHALL state that it is scoped per parent

#### Success Criteria

- SC-01: Four siblings whose order values were assigned out of creation sequence come back from `beans list --sort order` in the assigned sequence, and a fifth sibling without an order value comes last.

_Requirements: R-09, R-12_

## Recommended Skills

- `tdd`



## Implementation Report

Commit: 652f5de "feat(list): add order as a list --sort key, scoped per parent"

- Added case "order" in sortBeans (internal/commands/list.go): groups beans by Parent (stable, first-appearance order), sorts each group with bean.SortByOrder (pkg/bean/sort_order.go, from beans-y2a2), then concatenates groups back into the input slice.
- Updated --sort flag help text to list order and note it is scoped per parent (AC3).
- Deviation/decision (spec left the grouping strategy open): grouped by Parent field, group order = order of first appearance of that parent in the input list (stable, deterministic, no extra sorting of parent IDs themselves). Chosen because it requires no assumption about a canonical parent ordering and keeps the existing input order (already filtered/queried) intact across parent boundaries.

### Verification (TDD, RED then GREEN)

Test added: internal/commands/list_test.go, TestSortBeans/"sort by order, scoped per parent" — 4 siblings with out-of-sequence Order values + 1 sibling with no Order (must sort last) under parent p1, plus a 6th bean under a different parent p2 sharing an Order value with a p1 sibling (must not interleave), matching SC-01.

Build in the shared checkout was blocked mid-task by an unrelated parallel edit to internal/commands/create_test.go (undefined: beansPkg, from a different in-flight bean). Verified RED/GREEN and go vet in an isolated git worktree (detached at HEAD, only list.go/list_test.go copied in) instead of the shared checkout:

RED (case "order" temporarily removed):
  --- FAIL: TestSortBeans/sort_by_order,_scoped_per_parent
  list_test.go:125: sort by order[1]: got [p1-a p2-x p1-b p1-c p1-d p1-e], want [p1-a p1-b p1-c p1-d p1-e p2-x]

GREEN (implementation restored):
  --- PASS: TestSortBeans (0.00s)
      --- PASS: TestSortBeans/sort_by_id
      --- PASS: TestSortBeans/sort_by_created
      --- PASS: TestSortBeans/sort_by_created_with_nil
      --- PASS: TestSortBeans/sort_by_updated
      --- PASS: TestSortBeans/sort_by_status
      --- PASS: TestSortBeans/sort_by_order,_scoped_per_parent
      --- PASS: TestSortBeans/default_sort_(archive_status_then_type)
  PASS
  ok  	github.com/hmans/beans/internal/commands	0.553s

go vet ./internal/commands/...: clean (no output)

The worktree was removed after verification (git worktree remove --force); no files besides list.go/list_test.go/beans-uo43's bean file were touched.
