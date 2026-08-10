---
# beans-nda7
title: Generate fractional index values between neighbours
status: todo
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-zb0r
---

The core the rest of the epic rests on: given two neighbouring index values, produce a value that sorts lexicographically strictly between them. Not an integer rank — a fractional index exists so that a reorder writes **one** file instead of renumbering all of them.

### Requirement 1: A value can always be placed between two neighbours

**Objective:** As a maintainer of an ordered backlog, I want an index value that can always be inserted between two others, so that moving one bean never rewrites its siblings.

#### Acceptance Criteria

1. WHEN a predecessor and a successor value are given THE generator SHALL return a value that sorts lexicographically strictly after the predecessor and strictly before the successor
2. WHEN only a successor is given THE generator SHALL return a value that sorts strictly before it
3. WHEN only a predecessor is given THE generator SHALL return a value that sorts strictly after it
4. WHEN neither neighbour is given THE generator SHALL return a valid initial value
5. WHEN the generator is applied repeatedly between a fixed pair THE returned values SHALL stay strictly ordered and SHALL stay distinct

#### Success Criteria

- SC-01: One thousand successive insertions between the same two neighbours yield one thousand distinct values, each strictly between its own neighbours under plain lexicographic comparison.

_Requirements: R-10_

## Recommended Skills

- `tdd`
