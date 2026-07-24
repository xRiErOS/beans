---
# beans-9yif
title: 'T07 rename: guards (worktrees + server)'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:50Z
parent: beans-e040
blocked_by:
    - beans-z1we
---

checkNoActiveWorktrees + checkServerNotRunning. Plan Task 7.

## Objective
Als Nutzer will ich, dass ein Prefix-Rebrand verweigert wird, wenn ein Server läuft oder aktive Worktrees existieren (D05) — damit kein divergenter/halb-migrierter Zustand entsteht.

## EARS
- WHEN `PlanRebrand` startet UND ein TCP-Dial auf `127.0.0.1:<GetServerPort()>` gelingt, THE SYSTEM SHALL den Rebrand mit Server-läuft-Fehler verweigern.
- WHEN `PlanRebrand` startet UND das aufgelöste Worktree-Verzeichnis eine `*.meta.json` enthält, THE SYSTEM SHALL mit aktive-Worktrees-Fehler verweigern.
- Der projectName SHALL wie `serve.go:115-117` aufgelöst werden: `GetProjectName()` zuerst, `basename(ConfigDir())` nur als Fallback.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestGuard` GRÜN — `TestGuard_activeWorktreeRefusesRebrand` UND `TestGuard_runningServerRefusesRebrand`.
- SC-002: Guards feuern VOR jeder Mutation (fail-fast in PlanRebrand, vor RLock/Staging).

## Betroffene Pfade
- `pkg/beancore/rename.go` (checkNoActiveWorktrees + checkServerNotRunning + Wiring in PlanRebrand; Imports `net`,`time`,`strings`), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 7.
