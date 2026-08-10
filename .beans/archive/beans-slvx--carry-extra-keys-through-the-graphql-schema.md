---
# beans-slvx
title: Carry extra keys through the GraphQL schema
status: completed
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:50:06Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

Pull `gqlgen.yml` and `pkg/beangraph` along, then regenerate with `mise codegen`. Without it the CLI and the web UI show different beans for the same file, which is worse than the feature being absent.

### Requirement 1: GraphQL and CLI agree on a bean

**Objective:** As a user of the beans web UI, I want extra front matter keys to appear there too, so that the CLI and the UI do not describe the same bean differently.

#### Acceptance Criteria

1. WHEN a bean carrying extra keys is queried over GraphQL THE API SHALL return those keys
2. WHEN a bean is mutated over GraphQL THE API SHALL preserve every extra key the bean carried before the mutation
3. WHEN the schema changes THE generated code SHALL be regenerated with mise codegen and committed in the same change

#### Success Criteria

- SC-01: A GraphQL query for a bean carrying `release: 0-4-1` returns that pair, and a mutation of its priority over the API leaves the pair in the file.

_Requirements: R-07_

## Recommended Skills

- `tdd`


## Summary of Changes

AC1 (query returns extra keys) and AC2 (mutation preserves extra keys) were already structurally satisfied before any schema change: `gqlgen.yml` autobinds the GraphQL `Bean` type directly to `pkg/bean.Bean` (`model: github.com/hmans/beans/pkg/bean.Bean`), and `CoreResolver.UpdateBean` (`pkg/beangraph/mutations.go:106`) fetches the existing bean via `Core.Get`, mutates only the fields named in the input, and never touches `Extra` — so any mutation of an unrelated field naturally carries `Extra` through untouched. Added `TestQueryBeanExtra` and `TestMutationUpdateBeanPreservesExtra` (`internal/graph/schema.resolvers_test.go`, mirroring the file's existing `resolver.Query()`/`resolver.Mutation()` direct-call test convention — this codebase has no wire-level GraphQL string execution tests anywhere, so this matches established practice) — both passed on first run, confirming the backend logic needed no changes.

The actual gap was AC3: the field didn't exist on the wire. Added:
- `scalar Map` + `extra: Map` on `type Bean` (`internal/graph/schema.graphqls`) — nullable, since most beans carry no extra keys and a non-null scalar would error on a nil Go map.
- `gqlgen.yml`: bound the `Map` scalar to gqlgen's built-in `github.com/99designs/gqlgen/graphql.Map` (`= map[string]interface{}`) — no custom marshaler needed. `Bean.Extra`'s type `map[string]any` is assignable to the named `graphql.Map` (identical underlying type, one side unnamed — Go spec assignability), so gqlgen's autobind generated a direct field accessor (`internal/graph/generated.go:3671`, `return obj.Extra, nil`) with no manual resolver method required.
- Ran `mise codegen` (both `go generate ./...` for the backend and `pnpm codegen` for the frontend TS types) and committed the regenerated `internal/graph/generated.go` and `frontend/src/lib/graphql/generated.ts` in this change (AC3).

## Test-Output

```
=== RUN   TestQueryBeanExtra
--- PASS: TestQueryBeanExtra (0.00s)
=== RUN   TestMutationUpdateBeanPreservesExtra
--- PASS: TestMutationUpdateBeanPreservesExtra (0.00s)
PASS
ok  	github.com/hmans/beans/internal/graph	0.943s
```

## Smoke

`go build ./...` clean. `go test ./...` — all packages `ok`. `npx tsc --noEmit` in `frontend/` — clean (confirms the generated TS `extra?: Maybe<Scalars['Map']['output']>` field type-checks against the rest of the frontend).

## Notes for T(n+1)

No frontend UI component reads `extra` yet — this bean only wires the data through the schema/codegen layer, matching its stated scope ("so the CLI and the web UI do not describe the same bean differently" — describing, not yet displaying). Wiring a UI affordance for it is not part of this milestone.
