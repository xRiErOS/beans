---
# beans-bafb
title: 'T03 rename: ref-rewrite helper'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:16Z
parent: beans-e040
blocked_by:
    - beans-eeqn
---

rewriteRefs (parent/blocking/blocked_by). Plan Task 3.

## Objective
Als Entwickler will ich einen reinen Ref-Rewrite-Helper, der bei ID-Änderung alle referenzierenden Felder (parent/blocking/blocked_by) aktualisiert — Baustein für die Kaskade (T05/T06).

## EARS
- WHEN `rewriteRefs(b, m)` mit einer old→new-ID-Map aufgerufen wird, THE SYSTEM SHALL jedes Vorkommen einer gemappten alten ID in `b.Parent`/`b.Blocking`/`b.BlockedBy` durch die neue ID ersetzen.
- WHEN `rewriteRefs` zurückkehrt, THE SYSTEM SHALL die Anzahl der geänderten Ref-Felder liefern.
- IF ein Ref-Wert nicht in der Map ist, THE SYSTEM SHALL ihn unverändert lassen.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestRewriteRefs` GRÜN — gemischte Map (Parent + eine Blocking-Entry geändert, andere unberührt), Count == 2.

## Betroffene Pfade
- `pkg/beancore/rename.go` (append rewriteRefs), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 3. Feldtypen verifiziert: `Parent string`, `Blocking`/`BlockedBy []string` (bean.go:159-166).
