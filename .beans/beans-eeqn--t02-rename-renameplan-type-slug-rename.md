---
# beans-eeqn
title: 'T02 rename: RenamePlan type + slug-rename'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:16Z
parent: beans-e040
---

RenamePlan/RenameChange + PlanRenameSlug + ApplyRename slug branch. Plan Task 2.

## Objective
Als Nutzer will ich den Slug einer bean nachträglich setzen/leeren/regenerieren (`beans rename <id> --slug/--no-slug/--reslug`), ohne die ID oder Refs zu berühren — der Slug ist reiner Dateiname-Teil.

## EARS
- WHEN `PlanRenameSlug(id, newSlug, reslug)` aufgerufen wird, THE SYSTEM SHALL einen `RenamePlan{Mode:"slug"}` mit genau einem `RenameChange` (OldPath/NewPath `.beans`-relativ, Subdir erhalten) zurückgeben.
- WHEN `ApplyRename` einen slug-Plan erhält, THE SYSTEM SHALL die Datei per `os.Rename` umbenennen und `Bean.Slug`/`Bean.Path` in-memory nachführen.
- IF alte und neue Pfade identisch sind, THE SYSTEM SHALL keine Änderung vornehmen.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestApplyRenameSlug` GRÜN — alte Datei weg, neue da, NewPath `.beans`-relativ.
- SC-002: ID, `# id`-Kommentar und alle Refs bleiben unverändert (Slug nicht in Frontmatter/Cross-Refs).

## Betroffene Pfade
- `pkg/beancore/rename.go` (create: RenameChange/RenamePlan/newBeanPath/PlanRenameSlug/ApplyRename slug + applyRenameCascade-Stub), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 2.
