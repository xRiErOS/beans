---
# beans-m364
title: beans progress command
status: completed
type: task
priority: normal
created_at: 2025-12-27T21:44:05Z
updated_at: 2026-08-10T14:30:29Z
order: VVd
parent: beans-mmyp
---

Add `beans progress` command to show a summary of work status.

## Behavior

- Shows counts by status (e.g., "5 in-progress, 12 todo, 8 completed")
- Could show a simple progress bar
- Optional: filter by milestone/epic to see progress on specific initiatives

## Example

```bash
beans progress
# Output:
# In Progress: 2
# Todo: 15  
# Completed: 23
# Scrapped: 3
# ━━━━━━━━━━━━━━━━ 57% complete
```

## Summary of Changes

Implemented beans progress [--parent <id>] in internal/commands/progress.go, reusing childindex.go's descendant helpers. Final whole-branch review found and fixed a short-ID normalization bug in --parent scoping (progress.go now uses parent.ID); regression test added with a non-empty ID prefix.
