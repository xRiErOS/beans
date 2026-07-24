---
# beans-9yif
title: 'T07 rename: guards (worktrees + server)'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:21:52Z
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

## Prelude aus T06-Review (Supervisor, 2026-07-24)
- D02 (PO-Decision offen, siehe Epos beans-e040): .beans.yml-Config-Write bei Prefix-Rebrand ist nicht-atomar (config.Save = os.WriteFile). Falls dein Guard-Scope (server/worktree-Guards VOR Rename) sinnvoll erweiterbar ist um einen Post-Reload-Prefix-Konsistenz-Check oder atomareren Config-Write, prüfe das — aber primär ist es PO-Entscheidung, NICHT ungefragt implementieren. Kontext für dich, damit die Guards diesen Failure-Mode kennen.
- I01 (low): TestRebrand_samePrefixRejected pinnt den same-prefix-Fall nur indirekt (via len(idMap)==0-Guard), nicht den newPrefix==oldPrefix-Frühguard. Test-Präzision — kein Handlungszwang.

## Summary of Changes

TDD (RED→GREEN): guard tests written first against the existing (guardless) `PlanRebrand`, confirmed failing, then `checkServerNotRunning`/`checkNoActiveWorktrees` implemented and wired in.

- `pkg/beancore/rename.go`: added `checkNoActiveWorktrees()` (refuses if the resolved worktree dir contains any `*.meta.json`; project-name resolution mirrors `serve.go:115-117` — `GetProjectName()` first, `basename(ConfigDir())` fallback) and `checkServerNotRunning()` (refuses if a 200ms TCP dial to `127.0.0.1:<GetServerPort()>` succeeds). Both called at the very top of `PlanRebrand`, before `c.mu.RLock()` — fail-fast, no lock held across the network dial, no staging reached on refusal (SC-002).
- `pkg/beancore/rename_test.go`: `TestGuard_activeWorktreeRefusesRebrand`, `TestGuard_runningServerRefusesRebrand` (SC-001, both green) plus two false-positive guards (`TestGuard_activeWorktreeAllowsRebrand_whenDirEmpty`, `TestGuard_noServerRunningAllowsRebrand`) proving the guards don't block a clean rebrand. Refusal tests also assert the original bean file is untouched (SC-002 no-mutation proof).

Todos:
- [x] checkNoActiveWorktrees — refuse on `*.meta.json` present, project-name resolution matches serve.go
- [x] checkServerNotRunning — refuse on successful TCP dial to configured port
- [x] Both wired into PlanRebrand before any lock/staging (fail-fast)
- [x] SC-001: `go test ./pkg/beancore/ -run TestGuard` green
- [x] SC-002: refusal leaves original files untouched, verified in tests
- [x] Full `go test ./...` and `go vet ./...` green (no regressions)

Deviations from plan: none — implementation matches PLAN.md Task 7 verbatim (function bodies, wiring point, guard order). D02 (Config-Write atomicity) intentionally NOT touched per Prelude instruction — stayed in T07 scope.
