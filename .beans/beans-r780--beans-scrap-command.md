---
# beans-r780
title: beans scrap command
status: completed
type: task
priority: normal
created_at: 2025-12-27T21:44:04Z
updated_at: 2026-08-10T14:30:28Z
order: VVv
parent: beans-mmyp
---

Add `beans scrap <id> --reason <text>` command.

## Behavior

- Sets status to `scrapped`
- **Required** `--reason` flag to document why the bean was scrapped
- Adds a `## Reason for Scrapping` section to the bean body (preserves project memory)
- Shows confirmation message

## Example

```bash
beans scrap beans-abc --reason "Superseded by beans-xyz approach"
```

## Summary of Changes

Implemented beans scrap <id> --reason <text> (required, dual-enforced) in internal/commands/scrap.go. Reviewed and approved, no fix round needed.
