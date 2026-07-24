---
# beans-z1we
title: 'T06 rename: prefix-rebrand + .beans.yml'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:50Z
parent: beans-e040
blocked_by:
    - beans-9p12
    - beans-n1ow
---

PlanRebrand + config write. Plan Task 6.

## Objective
Als Nutzer will ich projekt-weit den ID-Prefix tauschen (`beans rename --prefix "bew-"`), sodass alle beans (Suffix erhalten), alle Refs und `.beans.yml` konsistent migriert werden — die Kern-Motivation (überlange Prefixe kürzen).

## EARS
- WHEN `PlanRebrand(newPrefix)` aufgerufen wird, THE SYSTEM SHALL jede bean-ID via `RebrandID` auf `newPrefix` mappen (Suffix erhalten) und `Mode:"prefix"`, `NewPrefix`, `ConfigWrite:true` setzen.
- IF eine bean-ID nicht mit dem aktuellen Prefix beginnt, THE SYSTEM SHALL den Rebrand verweigern (kein Doppel-Prefix, B04-Guard).
- WHEN ein prefix-Plan angewendet wird, THE SYSTEM SHALL die Kaskade atomar ausführen UND danach den neuen `prefix:` in `.beans.yml` via `Config.Save(ConfigDir())` schreiben.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestRebrand` GRÜN — alle IDs remapped, Refs intakt nach Reload, `.beans.yml` prefix geschrieben.
- SC-002: `TestRebrand_mixedPrefixRefused` GRÜN — Mixed-Prefix-Repo wird verweigert.

## Betroffene Pfade
- `pkg/beancore/rename.go` (PlanRebrand + ApplyRename prefix-Zweig + writeRebrandConfig; Import `strings`), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 6.
