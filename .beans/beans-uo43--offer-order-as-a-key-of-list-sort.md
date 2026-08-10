---
# beans-uo43
title: Offer order as a key of list --sort
status: todo
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-zb0r
blocked_by:
    - beans-nda7
---

`beans list --sort` accepts created, updated, status, priority and id. Add order, scoped per parent (R-12), and say so in the help text — a sort key whose scope is not written down gets guessed wrong once and then relied upon.

### Requirement 1: Beans can be listed in manual order

**Objective:** As someone reading a backlog, I want to list beans in the order I placed them, so that the sequence I decided is the sequence I see.

#### Acceptance Criteria

1. WHEN list --sort order runs THE CLI SHALL sort beans by their fractional index within their parent group
2. IF a bean carries no order value THEN THE CLI SHALL place it after every bean of the same parent that carries one
3. THE help text of --sort SHALL name order among the accepted keys and SHALL state that it is scoped per parent

#### Success Criteria

- SC-01: Four siblings whose order values were assigned out of creation sequence come back from `beans list --sort order` in the assigned sequence, and a fifth sibling without an order value comes last.

_Requirements: R-09, R-12_

## Recommended Skills

- `tdd`
