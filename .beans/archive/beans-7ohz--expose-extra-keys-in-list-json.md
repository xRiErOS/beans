---
# beans-7ohz
title: Expose extra keys in list --json
status: completed
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:45:55Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

`beans list --json` emits a fixed schema today: id, slug, path, title, status, type, priority, tags, parent, created_at, updated_at, etag. An extra key would stay invisible even once it survives the round trip.

### Requirement 1: JSON output carries the extra keys

**Objective:** As a tool reading beans as data, I want the extra front matter keys in the JSON output, so that a generated plan can be built without reading every bean file.

#### Acceptance Criteria

1. WHEN list --json runs over a bean carrying extra keys THE CLI SHALL emit those keys as an object under the field extra
2. WHEN list --json runs over a bean carrying no extra keys THE CLI SHALL omit the field extra
3. WHEN show --json runs over a bean carrying extra keys THE CLI SHALL emit them under the same field name as list --json

#### Success Criteria

- SC-01: `beans list --json` over a store with one bean carrying `release: 0-4-1` yields an entry whose `extra.release` equals `0-4-1`, and beans without extra keys carry no `extra` field.

_Requirements: R-05_

## Recommended Skills

- `tdd`


## Summary of Changes

No production code was needed: `internal/commands/list.go` and `show.go` both already serialize beans via `internal/output.SuccessMultiple`/`SuccessSingle`, which `json.Encode` the raw `*bean.Bean` — and `Bean.MarshalJSON` (from `beans-n3hl`) already includes `Extra` under the tag `extra,omitempty`. AC1-3 were already satisfied structurally; this task adds the regression-locking coverage that was missing.

Added:
- `pkg/bean/bean_test.go`: two new subtests on `TestBeanJSONSerialization` (`extra omitted when empty`, `extra included when non-empty`) — locks AC1/AC2 at the actual mechanism (`Bean.MarshalJSON`).
- `internal/output/output_test.go` (new): `TestSuccessSingleAndMultipleAgreeOnExtraField` — captures stdout from both `SuccessSingle` (the function `show --json` calls) and `SuccessMultiple` (the function `list --json` calls) with the same bean and asserts both expose `extra.release` identically, and that a bean without extra keys omits the field in the array case too. Locks AC3 at the shared code path both commands go through.

## Test-Output

All new tests passed on first run (expected — no new production behavior, only coverage for already-correct code; confirmed via a manual smoke test on the built binary first: `beans create ... --set release=0-4-1` then `beans list --json` showed `extra.release`, a bean without extras omitted the field, matching SC-01 exactly):

```
=== RUN   TestBeanJSONSerialization/extra_omitted_when_empty
--- PASS
=== RUN   TestBeanJSONSerialization/extra_included_when_non-empty
--- PASS
=== RUN   TestSuccessSingleAndMultipleAgreeOnExtraField
--- PASS
```

## Smoke

`go build ./...` clean. `go test ./...` — all packages `ok`.

## Notes for T(n+1)

`beans-usk9` (--where filter) and `beans-uo43` (--sort key) both touch `internal/commands/list.go` — sequenced after this one, not run in parallel with it, to avoid overlapping edits in the same file.
