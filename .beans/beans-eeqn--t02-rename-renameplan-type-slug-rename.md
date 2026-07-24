---
# beans-eeqn
title: 'T02 rename: RenamePlan type + slug-rename'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T11:42:48Z
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

## Summary of Changes
- [x] `pkg/beancore/rename.go` neu: `RenameChange`, `RenamePlan`, `newBeanPath`, `repoRoot`, `PlanRenameSlug`, `ApplyRename` (dispatch), `applyRenameSlug`, `applyRenameCascade`-Stub (Fehler bis Task 5/6).
- [x] `pkg/beancore/rename_test.go` neu: `newTestCore`-Harness + `TestApplyRenameSlug_setsSlug` (SC-001), `TestApplyRenameSlug_idAndRefsUnchanged` (SC-002: ID/`# id`-Kommentar/Cross-Ref unverändert), `TestPlanRenameSlug_noopWhenPathsIdentical` (EARS-Punkt 3, identische Pfade → keine Änderung).
- [x] RED→GREEN verifiziert: Build-Fehler vor Implementierung (`undefined: PlanRenameSlug/ApplyRename`), danach `command go test ./pkg/beancore/ -run 'TestApplyRenameSlug|TestPlanRenameSlug' -v` GRÜN.
- [x] Regressionsfrei: `command go test ./...` (gesamtes Repo) GRÜN, `command go vet ./...` sauber, `command go build -o /dev/null ./cmd/beans` erfolgreich.
- Deviation: keine — Plan-Vorgabe (Task 2, Steps 1–5) 1:1 umgesetzt, zusätzlich zwei über das Minimal-Testbeispiel des Plans hinausgehende Tests ergänzt, um SC-002 und den No-op-Fall explizit als Test statt nur als Prosa-EARS abzudecken.
- Commit: `65bbb61` — `feat(rename): RenamePlan type and slug-rename path`.
