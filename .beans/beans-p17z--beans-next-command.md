---
# beans-p17z
title: beans next command
status: completed
type: task
priority: normal
created_at: 2025-12-27T21:44:04Z
updated_at: 2026-08-10T14:30:28Z
order: VVg
parent: beans-mmyp
---

Add `beans next` command to show the single most important bean to work on.

## Behavior

- Returns the highest-priority `todo` bean that is not blocked
- Shows full bean details (like `beans show`)
- If nothing is ready, suggests checking `beans blocked` or `beans list`

## Example

```bash
beans next
# Shows the single most important bean to tackle
```

## Summary of Changes

Implemented beans next in internal/commands/next.go, reusing list.go's --ready filter via an extracted applyReadyFilter(filter) helper (behavior-preserving refactor, verified via a new list_test.go regression test). Reviewed and approved.
