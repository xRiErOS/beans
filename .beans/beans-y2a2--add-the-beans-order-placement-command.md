---
# beans-y2a2
title: Add the beans order placement command
status: todo
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-zb0r
blocked_by:
    - beans-nda7
---

`beans order <id> --after <id>`, `--before <id>`, `--first`, `--last`. Placement is relative to siblings under the same parent (R-12).

The property this command exists for is that a move writes exactly one file. A test that only checks the resulting sequence would pass just as happily against a full renumbering, and would then be an evergreen — assert the write count.

### Requirement 1: A bean can be placed relative to its siblings

**Objective:** As someone maintaining a backlog, I want to move one bean to a position among its siblings, so that reordering costs one write and one small diff.

#### Acceptance Criteria

1. WHEN beans order <id> --after <sibling> runs THE CLI SHALL set the order value of <id> strictly between that sibling and the sibling following it
2. WHEN beans order <id> --before <sibling> runs THE CLI SHALL set the order value of <id> strictly between that sibling and the sibling preceding it
3. WHEN beans order <id> --first runs THE CLI SHALL set an order value that sorts before every sibling
4. WHEN beans order <id> --last runs THE CLI SHALL set an order value that sorts after every sibling
5. WHEN a bean is placed THE CLI SHALL write exactly one bean file
6. IF the bean named by --after or --before has a different parent THEN THE CLI SHALL exit non-zero with an error stating that order is scoped per parent
7. IF more than one of --after, --before, --first and --last is passed THEN THE CLI SHALL exit non-zero with a usage error

#### Success Criteria

- SC-01: Moving the fifth of six siblings to second position leaves the six in the expected sequence and modifies exactly one file on disk, asserted by a write count rather than by the resulting order alone.

_Requirements: R-10, R-12_

## Recommended Skills

- `tdd`
