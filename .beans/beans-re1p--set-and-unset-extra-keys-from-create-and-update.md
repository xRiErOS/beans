---
# beans-re1p
title: Set and unset extra keys from create and update
status: completed
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T11:16:34Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

Add `--set key=value` and `--unset key`, both repeatable, to `internal/commands/create.go` and `internal/commands/update.go`.

The known names are reserved. `--set title=…` must fail and point at `--title` rather than quietly creating a shadow field that the renderer then writes next to the real one.

### Requirement 1: Extra keys are writable from the CLI

**Objective:** As an agent maintaining a plan, I want to set and remove extra front matter keys from the command line, so that planning data does not have to be edited by hand.

#### Acceptance Criteria

1. WHEN --set key=value is passed to create or update THE CLI SHALL store the pair as an extra front matter key of the bean
2. WHEN --set or --unset is repeated THE CLI SHALL apply every occurrence
3. WHEN --unset key is passed THE CLI SHALL remove that key from the bean front matter
4. IF the key named by --set or --unset is a field of the known schema THEN THE CLI SHALL exit non-zero with an error that names the native flag for that field
5. IF --unset names a key the bean does not carry THEN THE CLI SHALL leave the bean unchanged and exit zero
6. IF the argument to --set carries no equals sign THEN THE CLI SHALL exit non-zero with a usage error

#### Success Criteria

- SC-01: `beans create "x" -t task --set release=0-4-1 --set klasse=bugfix` writes a file whose front matter carries both keys, and `--set status=done` exits non-zero naming `--status`.

_Requirements: R-03, R-04_

## Recommended Skills

- `tdd`


## Completion

Implemented in `pkg/bean/bean.go`, `internal/commands/content.go`, `internal/commands/create.go`, `internal/commands/update.go`.

- `bean.ReservedKeyFlag(key)` (pkg/bean/bean.go) maps the 11 known front-matter keys to their native CLI flag; `created_at`/`updated_at`/`order` map to `flag=""` (managed field, no direct write flag) since create/update expose no flag for them.
- `content.go` adds `parseSetPair`, `checkReservedKey`, `validateExtraKeys`, `applyExtraOps` — shared by create and update, reused as-is (not `--set`-specific) so `beans-usk9` can call the same helpers for `--where`.
- `create.go`/`update.go` add repeatable `--set key=value` / `--unset key` (StringArrayVar). Validated up front via `validateExtraKeys` (AC4/AC6) before any bean is created/mutated. Applied via a second `core.Update(b, nil)` write after the primary create/update call, per bean guidance — the generated `CreateBeanInput`/`UpdateBeanInput` types were not touched.
- Conflict rule (same key in both `--set` and `--unset` in one invocation): `--unset` wins — `applyExtraOps` applies all `--set` first, then all `--unset`. Chosen because unset is the more destructive, deliberately-final operation; predictable regardless of flag order on the command line (cobra doesn't preserve cross-flag interleaving order for two separate `StringArrayVar`s anyway).
- AC5 (`--unset` of an absent key): no-op on the map, exits zero, no error — but still performs the second write (bumps `updated_at`), matching the existing `--remove-tag`-on-absent-tag precedent elsewhere in `update.go` rather than adding a new "was there an actual change" branch.

## Verification

- TDD throughout: RED confirmed via `go vet`/`go test` build failures before each implementation step (`ReservedKeyFlag`, `content.go` helpers, CLI-level create/update tests), then GREEN.
- `/opt/homebrew/bin/go build ./...` clean.
- `/opt/homebrew/bin/go test ./...` — all packages pass (pkg/bean, internal/commands, and the rest of the repo untouched/green).
- SC-01 manually verified against the built binary: `beans create "x" -t task --set release=0-4-1 --set klasse=bugfix` writes both keys into the file's front matter; `beans create "y" -t task --set status=done` exits 1 with `Error: "status" is a reserved field; use --status instead`.

No deviations from the bean spec beyond the documented conflict-order decision (not specified by AC, decided and tested above).


## Fix Round 1 (Opus spec review, B01)

Independent Opus review of the whole `beans-2ark` epic found a real, evidenced bug: the second write that persists `--set`/`--unset` (and, since `beans-9ftf`, `--order`) passed `ifMatch=nil` to `core.Update`. Two failure modes, both reproduced by the reviewer and independently re-verified here:

1. **Silent bypass in the default config:** `beans update <id> --set k=v --if-match <stale-etag>` succeeded and wrote the key, ignoring the stale `--if-match` entirely — the optimistic-concurrency contract `--priority`/`--status` already honor was a no-op for `--set`/`--unset`.
2. **Hard failure + half-written state under `require_if_match: true`:** `beans create "x" -t task --set k=v` created the bean on disk WITHOUT the extra key, then failed with `if-match etag is required` on the second write. `beans update <id> --set k=v --if-match <correct-etag>` failed identically, since the correct `--if-match` was captured but never passed through to the second write either.

### Fix

`internal/commands/create.go`: capture `etag := b.ETag()` on the freshly-created bean **before** `applyExtraOps`/order-assignment mutate it, pass `&etag` (not `nil`) to `core.Update`. Create has no `--if-match` flag of its own, so using the bean's own just-created etag is the correct assertion ("nothing external touched this since I made it") and satisfies `require_if_match`.

`internal/commands/update.go`: the extra-key write now computes `extraIfMatch`:
- If a field-update write already ran this invocation (`hasFieldUpdates(input)`), that write already validated the caller's `--if-match`; `b` now reflects that freshly-persisted state, so `extraIfMatch` becomes `b.ETag()` captured before `applyExtraOps` runs — asserting "nothing changed between the two writes" without demanding a second `--if-match` from the caller.
- Otherwise (extras are the only change), `extraIfMatch` is the caller's own `ifMatch` (from `--if-match`), passed straight through — a stale or wrong etag is now rejected exactly like it would be for `--priority`/`--status`, and `require_if_match:true` with no `--if-match` correctly demands one (consistent with every other update path, not a regression).

Error handling on this second write also switched from `cmdError(..., ErrFileError, ...)` to the existing `mutationError` helper (already used for the first write), so an etag mismatch here surfaces as `CONFLICT`, not `FILE_ERROR`.

### Test-Output

RED (guard removed, reverted to `core.Update(b, nil)`, both proven cases fail exactly as the review reported):
```
=== RUN   TestCreateCmdSetSucceedsUnderRequireIfMatch
    create_test.go:270: createCmd.RunE() error = if-match etag is required (set require_if_match: false in config to disable) ...
--- FAIL: TestCreateCmdSetSucceedsUnderRequireIfMatch

=== RUN   TestUpdateCmdSetAloneRejectsStaleIfMatch
    update_test.go:231: expected an etag-mismatch error for a stale --if-match, got nil
--- FAIL: TestUpdateCmdSetAloneRejectsStaleIfMatch

=== RUN   TestUpdateCmdSetSucceedsUnderRequireIfMatch
    update_test.go:312: updateCmd.RunE() error = if-match etag is required ...
--- FAIL: TestUpdateCmdSetSucceedsUnderRequireIfMatch
```

GREEN (fix restored):
```
=== RUN   TestCreateCmdSetSucceedsUnderRequireIfMatch
--- PASS
=== RUN   TestUpdateCmdSetAloneRejectsStaleIfMatch
--- PASS
=== RUN   TestUpdateCmdSetAloneAcceptsCorrectIfMatch
--- PASS
=== RUN   TestUpdateCmdSetSucceedsUnderRequireIfMatch
--- PASS
```

Note on `TestUpdateCmdSetAloneRejectsStaleIfMatch`: asserts against the file re-read from disk (`readBeanFromDisk`), not `core.Get()`. `Core.Update` takes the bean by pointer and `applyExtraOps` mutates that same pointer before the etag check runs, so `core.Get()` after a *rejected* write still shows the attempted mutation (`c.beans[id]` is literally the same object) — this is documented in `Core.Update`'s own source comment and is exactly why its etag validation reads the on-disk file rather than trusting the in-memory bean. Pre-existing characteristic of `Core.Update`, not something this fix introduced or is in scope to change; the property that actually matters here (the file was not overwritten) can only be shown by reading the file.

## Smoke

`go build ./...` clean. `go test ./...` — all packages `ok`.

## Deferred (not blocking, per review verdict)

B02 (`--where`'s malformed-pair error names `--set` instead of `--where`, via shared `parseSetPair`), B03 (empty `--set` key `=value` is silently accepted) — both low severity, left for a follow-up bean rather than expanding this one's scope.
