---
# beans-9p12
title: 'T05 rename: single-ID rename (cascade)'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T11:45:29Z
parent: beans-e040
blocked_by:
    - beans-bafb
    - beans-is3j
---

PlanRenameID + applyRenameCascade. Plan Task 5.

## Objective
Als Nutzer will ich eine einzelne bean-ID ändern (`beans rename <id> <neue-id>`), wobei alle referenzierenden beans automatisch nachgezogen werden, ohne tote Refs.

## EARS
- WHEN `PlanRenameID(oldID, newID)` aufgerufen wird UND `newID` bereits existiert, THE SYSTEM SHALL einen Kollisions-Fehler zurückgeben (keine Mutation).
- WHEN ein Einzel-ID-Plan angewendet wird, THE SYSTEM SHALL die eigene Datei umbenennen, die `# id`-Kommentarzeile via `Render()` korrigieren und alle Refs (parent/blocking/blocked_by) referenzierender beans kaskadieren — atomar via stageAndSwap.
- WHEN `ApplyRename` (id-Modus) erfolgreich war, THE SYSTEM SHALL `c.beans` in-memory nachführen (`loadFromDisk`), sodass `Get(newID)` am selben Core auflöst.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestRenameID` GRÜN — Kaskade (parent+blocked_by des Kindes → neue ID), Kollisions-Refusal.
- SC-002: Same-Core `Get(newID)` löst nach Apply auf, `Get(oldID)` nicht (B01).
- SC-003: Angewendete Disk-Pfade == `plan.Changes` (I01-Assertion).

## Betroffene Pfade
- `pkg/beancore/rename.go` (PlanRenameID + planCascade + countRefHits + applyRenameCascade, Stub aus T02 ersetzen), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 5.

## Prelude aus T02-Review (Supervisor, 2026-07-24)

- I01 (medium): `newBeanPath` (pkg/beancore/rename.go) hat KEINEN Test für verschachtelten Subdir-Fall (z.B. `epic-auth/tp-aaaa--slug.md`). Logik via filepath.Dir/Join sieht korrekt aus, ist aber unbewiesen. Cascade-Rename (dieser Task) baut direkt darauf. VOR/BEI der Cascade-Implementierung einen Test mit verschachteltem Bean.Path ergänzen, der Subdir-Erhalt beweist.
- I02 (low): T02-Tests sind nicht table-driven (Plan-vorgegeben). Bei neuen Rename-Tests (id/cascade) table-driven bevorzugen (repo-CLAUDE.md-Konvention).
