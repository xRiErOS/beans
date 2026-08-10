---
# beans-9ftf
title: Accept an explicit index in beans create --order
status: todo
type: task
priority: low
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-zb0r
blocked_by:
    - beans-nda7
---

For importing an order that already exists elsewhere, so that a migration does not have to run a placement command per bean.

### Requirement 1: An order value can be set at creation

**Objective:** As someone importing an existing backlog, I want to set the order value while creating a bean, so that the imported sequence arrives in one pass.

#### Acceptance Criteria

1. WHEN beans create --order <value> runs THE CLI SHALL write that value to the order field of the new bean
2. WHEN beans create runs without --order THE CLI SHALL leave the order field empty
3. IF the given value is not a valid fractional index THEN THE CLI SHALL exit non-zero with an error

#### Success Criteria

- SC-01: Five beans created with ascending explicit order values come back from `beans list --sort order` in exactly that sequence.

_Requirements: R-11_

## Recommended Skills

- `tdd`
