---
# beans-lok4
title: 'T09 rename: renameBean GraphQL mutation (optional)'
status: scrapped
type: task
priority: deferred
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-08-10T11:29:44Z
parent: beans-e040
blocked_by:
    - beans-z1we
---

UI-only, deferred. Plan Task 9.

## Objective
(OPTIONAL / deferred — PO-Entscheidung: CLI nutzt kein GraphQL.) Als Beans-UI will ich Slug-/Einzel-ID-Rename über eine `renameBean`-Mutation auslösen. NICHT auf dem kritischen Pfad — nur umsetzen, wenn die UI es braucht.

## EARS
- WHEN die `renameBean(input)`-Mutation mit `newID` aufgerufen wird, THE SYSTEM SHALL `PlanRenameID` + `ApplyRename` ausführen und die umbenannte bean zurückgeben.
- WHEN mit `newSlug`/`reslug` aufgerufen, THE SYSTEM SHALL den Slug-Pfad ausführen.
- Prefix-Rebrand SHALL NICHT über diese Mutation angeboten werden (offline-Batch, verweigert bei laufendem Server).

## Success Criteria
- SC-001: Resolver-Test analog `UpdateBean` GRÜN; `mise codegen` lässt den Baum clean.

## Betroffene Pfade
- `internal/graph/schema.graphqls`, `pkg/beangraph/mutations.go`, `mise codegen`. Details: PLAN.md Task 9. Deferred — separat priorisieren.
