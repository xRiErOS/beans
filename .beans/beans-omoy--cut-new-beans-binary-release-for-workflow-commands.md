---
# beans-omoy
title: cut new beans binary release for workflow commands + prime
status: todo
type: task
priority: normal
created_at: 2026-08-10T12:40:50Z
updated_at: 2026-08-10T12:41:06Z
parent: beans-mmyp
blocked_by:
    - beans-0ajg
    - beans-r780
    - beans-jvkq
    - beans-m364
    - beans-18db
    - beans-p17z
    - beans-9m5y
---

Cut a new beans binary release once the workflow commands and the
prime recipes update have landed, so the extensions in this epic actually
reach users.

## Scope

- Confirm all sibling tasks in this epic are completed
- Run `mise run release:minor` (workflow commands are new user-facing
  capability -> minor bump per semver, per mise.toml:95-103)
- Verify `beans version` reports the new tag (internal/version, ldflags
  wired in mise.toml:56-58)
- git push && git push --tags -- ONLY WITH ERIK'S EXPLICIT GO-AHEAD AT THAT
  TIME (repo rule: push only with PO approval, not implied by this task
  being ready)

## Not in scope

- Any code changes -- this is purely the release/tag step
- Deciding whether push happens automatically -- always ask first
