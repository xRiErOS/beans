---
# beans-re1p
title: Set and unset extra keys from create and update
status: todo
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
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
