---
# beans-n1ow
title: 'T01 rename: pure ID transform helpers'
status: in-progress
type: task
priority: high
tags:
    - to-review
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T11:37:37Z
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

## Success Criteria Status
- [x] SC-001: `command go test ./pkg/bean/ -run 'TestIDSuffix|TestRebrandID' -v` GRÜN.
- [x] SC-002: `pkg/bean/id.go` bleibt I/O-frei — kein neuer Import (nur `strings`, bereits vorhanden).

## Summary of Changes
- `pkg/bean/id.go`: `IDSuffix(id, prefix string) string` und `RebrandID(id, oldPrefix, newPrefix string) string` angehängt, exakt wie im Plan Task 1 spezifiziert. Keine neuen Imports.
- `pkg/bean/id_test.go`: `TestIDSuffix` und `TestRebrandID` (table-driven) an bestehende Testdatei angehängt.
- TDD: RED verifiziert (`undefined: IDSuffix`, `undefined: RebrandID`) vor Implementierung, dann GREEN.
- Volles `pkg/bean`-Paket + `go vet` + `go build ./...` grün, keine Regressionen.
- Deviation: keine — 1:1 nach Plan Task 1 umgesetzt.
- Commit: `f5f04f4` `feat(rename): pure ID transform helpers`.
