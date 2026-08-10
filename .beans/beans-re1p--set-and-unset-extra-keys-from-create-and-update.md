---
# beans-re1p
title: Set and unset extra keys from create and update
status: completed
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:41:24Z
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
