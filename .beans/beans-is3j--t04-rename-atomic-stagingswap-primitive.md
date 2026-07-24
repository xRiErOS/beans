---
# beans-is3j
title: 'T04 rename: atomic staging+swap primitive'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:16Z
parent: beans-e040
blocked_by:
    - beans-eeqn
---

stageAndSwap + copyTree. Plan Task 4.

## Objective
Als Nutzer will ich, dass ein kaskadierender Rename (viele Dateien) atomar ist — entweder alle Änderungen greifen oder keine — damit ein Abbruch nie einen halb-migrierten `.beans/`-Zustand hinterlässt. Kein Atomic-Write-Helfer existiert heute.

## EARS
- WHEN `stageAndSwap(writes, removes)` aufgerufen wird, THE SYSTEM SHALL einen vollständigen neuen `.beans/`-Baum in einem Temp-Sibling-Dir bauen (Klon + removes + writes, Keys `.beans`-relativ), dann atomar per Dir-Swap einschwenken (alt → `.bak-*`, neu → `.beans`, Backup löschen).
- IF vor dem Swap ein Fehler auftritt, THE SYSTEM SHALL den Original-`.beans/`-Baum unberührt lassen und das Staging-Dir entfernen.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestStageAndSwap` GRÜN — beide Subtests (Pre-Swap-Failure lässt Original intakt + räumt Siblings; writes/removes werden korrekt angewendet, unberührte Dateien bleiben).
- SC-002: Nicht-bean-Dateien (z.B. `archive/`) im `.beans/` überleben den Swap.

## Betroffene Pfade
- `pkg/beancore/rename.go` (append copyTree + stageAndSwap), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 4.
