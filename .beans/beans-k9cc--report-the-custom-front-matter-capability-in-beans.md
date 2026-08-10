---
# beans-k9cc
title: Report the custom front matter capability in beans version
status: todo
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
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
