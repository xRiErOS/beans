---
# beans-bafb
title: 'T03 rename: ref-rewrite helper'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T11:47:09Z
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

## Summary of Changes
- [x] `rewriteRefs(b *bean.Bean, m map[string]string) int` in `pkg/beancore/rename.go` — ersetzt gemappte alte IDs in `Parent`/`Blocking`/`BlockedBy`, mutiert in-place, liefert Anzahl geänderter Felder. Leerer `Parent` matched nie (auch nicht bei `m[""]`-Eintrag).
- [x] `TestRewriteRefs` in `pkg/beancore/rename_test.go` — table-driven, 3 Subtests: (1) gemischte Map Parent+eine Blocking-Entry geändert/Rest unberührt (SC-001, Count==2), (2) Ref nicht in Map bleibt unverändert, (3) leerer Parent nie gematcht auch wenn `""` als Map-Key vorhanden.
- Deviation vom bean-Wortlaut: SC-001 verlangt genau einen Testfall; umgesetzt als table-driven Test mit 3 Fällen (SC-001-Fall + 2 Edge-Cases aus den EARS-IF/WHEN-Klauseln), da EARS explizit auch "IF ein Ref-Wert nicht in der Map ist" und implizit den Empty-Parent-Fall abdeckt. Kein Scope-Creep — nur `rewriteRefs`, keine `buildRenameMap`/Cascade-Logik (bleibt T05/T06 laut Plan).

**Validation:** `go test ./pkg/beancore/ -run TestRewriteRefs -v` GRÜN (alle 3 Subtests). Voll-Gate `command go build ./...`, `command go vet ./...`, `command go test ./...` — alle GRÜN, keine Regression.
