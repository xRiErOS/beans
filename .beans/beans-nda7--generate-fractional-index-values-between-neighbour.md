---
# beans-nda7
title: Generate fractional index values between neighbours
status: completed
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:39:03Z
parent: beans-zb0r
---

The core the rest of the epic rests on: given two neighbouring index values, produce a value that sorts lexicographically strictly between them. Not an integer rank — a fractional index exists so that a reorder writes **one** file instead of renumbering all of them.

### Requirement 1: A value can always be placed between two neighbours

**Objective:** As a maintainer of an ordered backlog, I want an index value that can always be inserted between two others, so that moving one bean never rewrites its siblings.

#### Acceptance Criteria

1. WHEN a predecessor and a successor value are given THE generator SHALL return a value that sorts lexicographically strictly after the predecessor and strictly before the successor
2. WHEN only a successor is given THE generator SHALL return a value that sorts strictly before it
3. WHEN only a predecessor is given THE generator SHALL return a value that sorts strictly after it
4. WHEN neither neighbour is given THE generator SHALL return a valid initial value
5. WHEN the generator is applied repeatedly between a fixed pair THE returned values SHALL stay strictly ordered and SHALL stay distinct

#### Success Criteria

- SC-01: One thousand successive insertions between the same two neighbours yield one thousand distinct values, each strictly between its own neighbours under plain lexicographic comparison.

_Requirements: R-10_

## Recommended Skills

- `tdd`


## Summary of Changes

`OrderBetween` (`pkg/bean/fractional.go:16`) already existed from the initial repo import (commit `99260bf`, months before this milestone) and already satisfied AC1-4 structurally. Writing the SC-01 test (1000 successive insertions into the same narrowing gap, asserting all 1000 values distinct via a `map[string]bool`, not just pairwise-ordered) surfaced a real bug in `midpoint`'s "go deeper" branch (`pkg/bean/fractional.go`, previously lines 58-69): when `a`'s next digit was already at the maximum (61, `'z'`), `mid := (nextA + 62) / 2` computed a value equal to `nextA` instead of greater than it, so `OrderBetween("yz", "z")` returned `"yz"` — equal to its own lower bound, violating strict betweenness (AC1) and, over repeated narrowing, breaking distinctness (AC5).

Fixed by reusing `incrementKey` (which already has correct max-digit/carry handling and length-extension) on the suffix past the divergence point, instead of hand-computing a single midpoint digit that didn't check for saturation.

## Test-Output

RED (real bug caught by the new SC-01 test, not a placeholder):
```
=== RUN   TestOrderBetween_ThousandInsertionsStayDistinctAndOrdered
    fractional_test.go:107: insertion 12: "yz" not strictly between "yz" and "z"
--- FAIL: TestOrderBetween_ThousandInsertionsStayDistinctAndOrdered (0.00s)
```

GREEN (full fractional suite, old + new tests):
```
=== RUN   TestOrderBetween_BothEmpty
--- PASS
=== RUN   TestOrderBetween_BeforeKey
--- PASS
=== RUN   TestOrderBetween_AfterKey
--- PASS
=== RUN   TestOrderBetween_Between
--- PASS
=== RUN   TestOrderBetween_AdjacentDigits
--- PASS
=== RUN   TestOrderBetween_DeepAdjacentPrefixAtMaxDigit
--- PASS
=== RUN   TestOrderBetween_ManyInsertions
--- PASS
=== RUN   TestOrderBetween_ManyInsertionsAtStart
--- PASS
=== RUN   TestOrderBetween_ThousandInsertionsStayDistinctAndOrdered
--- PASS
=== RUN   TestOrderBetween_ThousandInsertionsNarrowingTowardLo
--- PASS
=== RUN   TestOrderBetween_ManyInsertionsBetween
--- PASS
PASS
ok  	github.com/hmans/beans/pkg/bean	0.302s
```

## Smoke

`go build ./...` clean at the time of this commit (pkg/bean is shared with the concurrently in-flight beans-re1p task; this bean's diff is scoped to fractional.go/fractional_test.go only, no overlap).

## Notes for T(n+1)

The primitive (`OrderBetween`) is solid and covered; wiring it into create/update/reorder commands is the rest of `beans-zb0r`'s epic, not this bean. No CLI surface was touched here.
