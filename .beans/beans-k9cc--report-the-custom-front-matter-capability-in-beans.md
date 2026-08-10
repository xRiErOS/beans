---
# beans-k9cc
title: Report the custom front matter capability in beans version
status: completed
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:42:41Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

The data format is shared by at least five stores: lean-stack, sproutling, okf-tools, plug-in_VC-Search and the worktrees. An older binary writing a file that carries extra keys deletes them exactly as measured in the brief. The format cannot defend itself against that; what it can do is make the version tell.

### Requirement 1: The capability is visible from the binary

**Objective:** As someone sharing one bean store across machines and versions, I want to see from the binary whether it preserves extra front matter keys, so that a silent data loss becomes a visible version mismatch.

#### Acceptance Criteria

1. WHEN beans version runs THE CLI SHALL report whether the binary preserves custom front matter keys
2. WHEN beans version --json runs THE CLI SHALL report the same fact in a machine-readable field

#### Success Criteria

- SC-01: `beans version --json` on the built binary carries a field stating the custom front matter capability, and the release of this epic is cut as a version bump rather than a patch.

_Requirements: R-08_

## Recommended Skills

- `tdd`

## Implementation Report

Added `internal/version.CustomFrontMatter` (const, always `true`) as the single source of truth, surfaced two ways:

- Text: `beans version` now prints a second line `custom front matter: preserved` (AC1).
- JSON: `beans version --json` (new flag, did not exist before) encodes `internal/version.Info` with field `custom_front_matter: true` (AC2), plus `version`/`commit`/`date`.

An older binary predating this bean simply lacks the `--json` flag and the second text line entirely -- that absence is the version signal SC-01 asks for.

No capability-registry abstraction added (single bool field, YAGNI per task brief).

### Verification

- TDD: `internal/version/version_test.go` and `internal/commands/version_test.go` written first, confirmed RED (undefined: JSON / undefined: versionJSON), then GREEN after implementation.
- `/opt/homebrew/bin/go build ./...` clean.
- `/opt/homebrew/bin/go test ./...` all packages pass, no regressions.
- Manual smoke on built binary: `beans version` -> `beans dev (unknown) built unknown
custom front matter: preserved`; `beans version --json` -> `{"version":"dev","commit":"unknown","date":"unknown","custom_front_matter":true}`.

### Deviations

None from the task brief.
