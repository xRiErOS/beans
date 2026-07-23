---
# beans-l36h
title: T1 fork/main als Integrationsbasis (ti53-Merge)
status: todo
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:28:32Z
parent: beans-1ec3
---

**Plan-Referenz:** `docs/roadmap-tty-output/PLAN.md` → Task 1. Dort stehen alle Schritte im Detail.

## Objective (User Story)

Als Entwickler des TTY-Renderers brauche ich eine Codebasis, auf der `featureGroup` bereits
existiert, damit der Renderer alle vier Ebenen abdecken kann und der ti53-Bug (Feature-Kinder
verschwinden) im neuen Pretty-Pfad nicht zurückkehrt.

## Hintergrund

`main` == `fork/main` == `origin/main` (0 Commits divergent). Der Nesting-Fix liegt 8 Commits
voraus auf `fix/beans-ti53-roadmap-nested-hierarchy` (bean `beans-ti53`, completed; PR #207
upstream seit 2026-07-17 offen). Eriks installiertes Binary `0.4.2-fork.ti53` hat den Fix bereits —
ohne diesen Merge wäre der TTY-Pfad ein Rückschritt gegenüber dem Alltag (D13/D14).

## EARS-Anforderungen

- **EARS-1** THE Branch `main` SHALL nach diesem Task die Typen `roadmapData`, `unscheduledGroup`,
  `milestoneGroup`, `epicGroup` und `featureGroup` in `internal/commands/roadmap.go` enthalten.
- **EARS-2** WHEN `main` von `fix/beans-ti53-roadmap-nested-hierarchy` divergiert ist (linke Zahl
  von `git rev-list --left-right --count` != 0), THEN THE Agent SHALL den Merge abbrechen und den
  PO fragen, statt einen Merge-Commit zu erzeugen.
- **EARS-3** THE Merge SHALL als Fast-Forward erfolgen (`git merge --ff-only`).
- **EARS-4** THE Agent SHALL `main` nach dem Remote `fork` pushen und NIEMALS nach `origin`.

## Akzeptanzkriterien

- [ ] **SC-101** `git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy`
      gibt vor dem Merge exakt `0	8` aus.
- [ ] **SC-102** `git merge --ff-only fix/beans-ti53-roadmap-nested-hierarchy` meldet `Fast-forward`.
- [ ] **SC-103** `grep -n 'type featureGroup' internal/commands/roadmap.go` liefert genau eine
      Trefferzeile: `62:type featureGroup struct {`.
- [ ] **SC-104** `command go test ./internal/commands/` endet mit `ok`.
- [ ] **SC-105** `git push fork main` erfolgreich; `git log origin/main..main` zeigt die Commits
      als nicht-gepusht nach origin (kein origin-Push erfolgt).

## Betroffene Pfade

Keine Quelldatei wird editiert — reine Branch-Operation in
`/Users/erik/Obsidian/tools/lean-stack/beans-src`.
