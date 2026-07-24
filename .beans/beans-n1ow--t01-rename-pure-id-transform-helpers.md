---
# beans-n1ow
title: 'T01 rename: pure ID transform helpers'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:16Z
parent: beans-e040
---

IDSuffix + RebrandID in pkg/bean/id.go. Plan Task 1.

## Objective
Als beans-Maintainer will ich reine, I/O-freie ID-Transform-Helper, damit ID-Rebranding isoliert unit-testbar ist (Vorbedingung für T06 Prefix-Rebrand).

## EARS
- WHEN `IDSuffix(id, prefix)` mit einem `id` aufgerufen wird, der mit `prefix` beginnt, THE SYSTEM SHALL den `id` ohne diesen Prefix zurückgeben.
- IF `id` nicht mit `prefix` beginnt ODER `prefix` leer ist, THE SYSTEM SHALL `id` unverändert zurückgeben.
- WHEN `RebrandID(id, oldPrefix, newPrefix)` aufgerufen wird, THE SYSTEM SHALL `newPrefix + IDSuffix(id, oldPrefix)` zurückgeben.

## Success Criteria
- SC-001: `go test ./pkg/bean/ -run 'TestIDSuffix|TestRebrandID'` GRÜN (table-driven: long→short, no-prefix, empty-prefix, idempotent).
- SC-002: `pkg/bean/id.go` bleibt I/O-frei (kein neuer Import außer `strings`).

## Betroffene Pfade
- `pkg/bean/id.go` (append), `pkg/bean/id_test.go` (create). Details + Code: `docs/beans-rename-command/PLAN.md` Task 1.
