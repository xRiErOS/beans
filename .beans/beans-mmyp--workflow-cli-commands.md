---
# beans-mmyp
title: Workflow CLI commands
status: completed
type: epic
priority: normal
created_at: 2025-12-27T21:43:38Z
updated_at: 2026-08-10T16:18:15Z
order: w
parent: beans-xej5
---

Add explicit workflow-style CLI commands that provide intuitive shortcuts for common operations. These commands wrap existing functionality with cleaner, more memorable interfaces.

## Rationale

Currently, users need to use `beans update <id> -s completed` to complete a bean. Workflow commands like `beans complete <id>` are more intuitive and can enforce best practices (like requiring a reason when scrapping).

## Proposed Commands

- `beans complete` - Complete a bean with optional summary
- `beans scrap` - Scrap a bean with required reason  
- `beans start` - Start working on a bean
- `beans ready` - Find beans ready to work on
- `beans next` - Show the next bean to work on
- `beans milestones` - List planned milestones
- `beans blocked` - Show blocked beans
- `beans progress` - Show work progress summary


## Scope Update (2026-08-10)

Two tasks added to close the loop from command to documentation to shipped
binary: `beans-9m5y` (rewrite `beans prime` to reference the new commands
authoritatively instead of derived list/update incantations) and
`beans-omoy` (cut the release). Both are blocked by all six command tasks;
the prime task additionally blocks the release task, so the sequence is
commands -> prime docs -> release.

## Summary of Changes

Epic complete: all 6 workflow CLI commands (complete/scrap/start/next/milestones/progress) implemented, reviewed, and merged; prime docs rewritten to document them authoritatively; v0.5.0 tagged and pushed to fork.
