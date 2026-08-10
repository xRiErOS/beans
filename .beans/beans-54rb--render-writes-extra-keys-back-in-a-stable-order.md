---
# beans-54rb
title: Render writes extra keys back in a stable order
status: completed
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:22:45Z
parent: beans-2ark
blocked_by:
    - beans-n3hl
---

`Render` (`pkg/bean/bean.go:227`) serialises the `renderFrontMatter` struct (`pkg/bean/bean.go:212`), and a struct cannot serialise dynamic keys. Move the write path to `yaml.Node` or an ordered map: the known fields first, in the order they have today, then the extra keys sorted by key.

The sort is not cosmetic. Without a fixed order every write produces a different key sequence, and every bean file churns in git for no reason.

### Requirement 1: Extra keys survive the write path in a fixed order

**Objective:** As an agent editing beans through the CLI, I want a write to preserve every extra key in a stable position, so that a round trip is a no-op and diffs stay readable.

#### Acceptance Criteria

1. WHEN Render writes a bean carrying extra keys THE renderer SHALL emit the known fields in their present order followed by the extra keys sorted by key
2. WHEN a bean file is parsed and rendered without modification THE output SHALL be byte-identical to the input except for updated_at
3. WHEN Render runs twice on the same bean THE two outputs SHALL carry the same key order
4. WHEN a command updates a known field of a bean carrying extra keys THE written file SHALL still carry every extra key

#### Success Criteria

- SC-01: A file with three extra keys parsed and rendered is byte-identical except for updated_at, and two consecutive renders produce identical bytes.
- SC-02: Removing the extra-key branch from Render turns the update test of AC-4 red — the reverse mutation against the guard is part of this task, not a later one.

_Requirements: R-01, R-02_

## Recommended Skills

- `tdd`


## Summary of Changes

`Render` (`pkg/bean/bean.go:288`) now builds the front matter via `yaml.Node` instead of marshaling `renderFrontMatter` directly: `node.Encode(&fm)` produces the known-field mapping in struct order, then extra keys (sorted via `sort.Strings`) are appended as key/value node pairs to `node.Content` before the whole node is marshaled. This gives a stable, deterministic key order without introducing a hand-rolled ordered-map type.

## Test-Output

RED (guard-branch removed, AC-4 test fails — SC-02 reverse mutation, verified inline then reverted, not committed):
```
=== RUN   TestRenderPreservesExtraKeysAfterKnownFieldUpdate
    bean_test.go:1930: len(Extra) = 0, want 2; Extra = map[string]interface {}(nil)
--- FAIL: TestRenderPreservesExtraKeysAfterKnownFieldUpdate (0.00s)
```

GREEN (full package):
```
=== RUN   TestRenderWritesExtraKeysSortedAfterKnownFields
--- PASS: TestRenderWritesExtraKeysSortedAfterKnownFields (0.00s)
=== RUN   TestRenderTwiceProducesIdenticalOutput
--- PASS: TestRenderTwiceProducesIdenticalOutput (0.00s)
=== RUN   TestParseRenderRoundtripWithExtraKeys
--- PASS: TestParseRenderRoundtripWithExtraKeys (0.00s)
=== RUN   TestRenderPreservesExtraKeysAfterKnownFieldUpdate
--- PASS: TestRenderPreservesExtraKeysAfterKnownFieldUpdate (0.00s)
PASS
ok  	github.com/hmans/beans/pkg/bean	0.302s
```

## Smoke

`go build ./...` clean. `go test ./...` — all packages `ok` (ran directly, not via `mise test`; see beans-n3hl for the pre-existing frontend/codegen environment issue unrelated to this task).

## Notes for T(n+1)

Epic 1's data format (`beans-n3hl` + `beans-54rb`) is now closed: `Bean.Extra` parses and round-trips through `Render`/`Parse` with a stable, JSON-safe, sorted key order. The fanned-out leaves (`re1p`, `7ohz`, `usk9`, `slvx`, `k9cc`) can build on this without touching `pkg/bean`.
