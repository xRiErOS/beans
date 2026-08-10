---
# beans-usk9
title: Filter on extra keys with list --where
status: todo
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:06:19Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

Add `--where key=value` to `internal/commands/list.go`, filtering on extra front matter keys. Without it every custom field is data one can read but not search.

### Requirement 1: Extra keys are filterable

**Objective:** As an agent surveying a plan, I want to select beans by an extra front matter key, so that a release or a class can be listed without post-processing the whole store.

#### Acceptance Criteria

1. WHEN the list command receives a key-value filter on an extra key THE CLI SHALL return only beans whose extra key of that name equals that value
2. WHEN the list command receives more than one such filter THE CLI SHALL return only beans that satisfy every given pair
3. IF the filtered key is a field of the known schema THEN THE CLI SHALL exit non-zero with an error naming the native filter flag for that field
4. WHEN the filtered key is carried by no bean THE CLI SHALL return an empty result and exit zero

#### Success Criteria

- SC-01: In a store of five beans of which two carry `release: 0-4-1`, `beans list --where release=0-4-1` returns exactly those two.

_Requirements: R-06_

## Recommended Skills

- `tdd`
