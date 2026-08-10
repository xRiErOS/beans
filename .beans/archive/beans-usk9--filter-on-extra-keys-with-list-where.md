---
# beans-usk9
title: Filter on extra keys with list --where
status: completed
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:54:11Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

Add `--where key=value` to `internal/commands/list.go`, filtering on extra front matter keys. Without it every custom field is data one can read but not search.

### Requirement 1: Extra keys are filterable

**Objective:** As an agent surveying a plan, I want to select beans by an extra front matter key, so that a release or a class can be listed without post-processing the whole store.

#### Acceptance Criteria

1. WHEN the list command receives a key-value filter on an extra key THE CLI SHALL return only beans whose extra key of that name equals that value
2. WHEN the list command receives more than one such filter THE CLI SHALL return only beans that satisfy every given pair
3. IF the filtered key is a field of the known schema THEN THE CLI SHALL exit non-zero with an error naming the native filter flag for that field
4. WHEN the filtered key is carried by no bean THE CLI SHALL return an empty result and exit zero

#### Success Criteria

- SC-01: In a store of five beans of which two carry `release: 0-4-1`, `beans list --where release=0-4-1` returns exactly those two.

_Requirements: R-06_

## Recommended Skills

- `tdd`

## Summary of Changes

Added `--where key=value` (`StringArrayVar`, repeatable, AND semantics) to `internal/commands/list.go`. Two new pure functions, both reusing existing generic helpers from `content.go` instead of duplicating them:

- `validateWhereKeys(wheres []string) error` — runs before the query, reuses `parseSetPair` (usage error on missing `=`) and `checkReservedKey`/`bean.ReservedKeyFlag` (AC3: reserved key -> error naming the native flag, e.g. `"status" is a reserved field; use --status instead`).
- `filterByWhere(beans []*bean.Bean, wheres []string) []*bean.Bean` — runs after `resolver.Beans(ctx, filter)` returns, filters the `[]*bean.Bean` in CLI code (AC1/AC2: AND-combines every pair against `Bean.Extra`); an unmatched key naturally yields an empty slice, no error (AC4).

`model.BeanFilter` (generated GraphQL type) was left untouched, per the task's constraint.

## Test Output

RED (functions did not exist yet):
```
internal/commands/list_test.go:152:9: undefined: validateWhereKeys
internal/commands/list_test.go:183:9: undefined: filterByWhere
internal/commands/list_test.go:284:9: undefined: listWhere
FAIL	github.com/hmans/beans/internal/commands [build failed]
```

GREEN (new tests):
```
=== RUN   TestValidateWhereKeysReservedKeyFails
--- PASS: TestValidateWhereKeysReservedKeyFails (0.00s)
=== RUN   TestValidateWhereKeysWithoutEqualsFails
--- PASS: TestValidateWhereKeysWithoutEqualsFails (0.00s)
=== RUN   TestValidateWhereKeysAcceptsExtraKey
--- PASS: TestValidateWhereKeysAcceptsExtraKey (0.00s)
=== RUN   TestFilterByWhereSingleMatch
--- PASS: TestFilterByWhereSingleMatch (0.00s)
=== RUN   TestFilterByWhereMultiplePairsAreANDed
--- PASS: TestFilterByWhereMultiplePairsAreANDed (0.00s)
=== RUN   TestFilterByWhereNoMatchIsEmpty
--- PASS: TestFilterByWhereNoMatchIsEmpty (0.00s)
=== RUN   TestFilterByWhereEmptyWheresReturnsAllBeans
--- PASS: TestFilterByWhereEmptyWheresReturnsAllBeans (0.00s)
=== RUN   TestListCmdWhereEndToEnd
--- PASS: TestListCmdWhereEndToEnd (0.00s)
PASS
```

Full suite (excluding `pkg/bean`, see Deviations): `go build ./...` clean; `go test $(go list ./... | grep -v '/pkg/bean$')` all `ok`.

SC-01 also verified end-to-end via the built binary: 5 beans, 2 carrying `release: 0-4-1`, `beans list --where release=0-4-1 --json` returned exactly those 2; `--where status=done` (reserved) exited non-zero naming `--status`; `--where release=nope` (no carrier) returned `[]` with exit 0.

## Deviations

- `pkg/bean` currently fails to build (`undefined: SortByOrder` in `pkg/bean/sort_order_test.go`) due to untracked, uncommitted work-in-progress from a parallel task (`beans-y2a2`, `pkg/bean/sort_order.go` + test) already present in the working tree before this task started. Per the task's explicit instruction not to touch `pkg/bean`, this was left as-is and excluded from the full-suite run; all other packages, including `internal/commands`, build and pass.
