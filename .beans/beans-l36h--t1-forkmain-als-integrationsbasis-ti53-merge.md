---
# beans-l36h
title: T1 fork/main als Integrationsbasis (ti53-Merge)
status: completed
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:50:08Z
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

- [ ] **SC-101** (obsolet, ersetzt durch SC-101 im ERRATA-Abschnitt — `0	8` traf nicht mehr zu,
      siehe D20/ERRATA) `git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy`
      gibt vor dem Merge exakt `0	8` aus.
- [ ] **SC-102** (obsolet, ersetzt durch SC-102 im ERRATA-Abschnitt — `--ff-only` war unmöglich,
      siehe D20/ERRATA) `git merge --ff-only fix/beans-ti53-roadmap-nested-hierarchy` meldet `Fast-forward`.
- [x] **SC-103** `grep -n 'type featureGroup' internal/commands/roadmap.go` liefert genau eine
      Trefferzeile: `62:type featureGroup struct {`.
- [x] **SC-104** `command go test ./internal/commands/` endet mit `ok`.
- [x] **SC-105** `git push fork main` erfolgreich; `git log origin/main..main` zeigt die Commits
      als nicht-gepusht nach origin (kein origin-Push erfolgt).

## Betroffene Pfade

Keine Quelldatei wird editiert — reine Branch-Operation in
`/Users/erik/Obsidian/tools/lean-stack/beans-src`.

## Nachtrag 2026-07-23 (PO-Entscheid im Vorflug-Check) — D20 ersetzt EARS-3/SC-101/SC-102

**Befund des Supervisors vor Dispatch:**

```
git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy
3	8
```

Die Plan-Annahme "main == fork/main, 0 divergent" (D13) ist durch die Operationalisierung
dieses Epos selbst veraltet. main traegt drei Commits, die ti53 nicht hat — alle drei
reine `.beans/`-Pflege, **kein Go-Code**:

- `d69de0e chore(beans): roadmap-tty-output operationalisiert`
- `dda7140 chore(beans): gate-b-nachtraege f01-f03`
- `a24ec2c chore(beans): D19 test-gate + bt-xy2i scrapped`

`git merge --ff-only` ist damit unmoeglich.

**D20 (PO 2026-07-23): Integration per Rebase statt Fast-Forward-Merge.**

```
git rebase fix/beans-ti53-roadmap-nested-hierarchy
```

Die drei bean-Commits wandern auf die acht ti53-Commits obendrauf. Ergebnis: **lineare
Historie ohne Merge-Commit** — der Intent von EARS-3 ist erfuellt, nicht sein Wortlaut.

**Revidierte Akzeptanzkriterien** (ersetzen SC-101 und SC-102, EARS-3 gilt als
"lineare Historie, kein Merge-Commit"):

- [x] **SC-101 REVIDIERT** (Zahlen final im ERRATA-Abschnitt korrigiert auf `4	8`) Vor dem
      Rebase gibt `git rev-list --left-right --count
      main...fix/beans-ti53-roadmap-nested-hierarchy` exakt `3	8` aus. Weicht die linke
      Zahl davon ab (fremde Commits auf main), **abbrechen und PO fragen**.
- [x] **SC-102 REVIDIERT** (Zahlen final im ERRATA-Abschnitt korrigiert auf `4	0`) Nach dem
      Rebase gilt beides:
      `git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy`
      gibt `3	0` aus, **und** `git log --merges -1 --oneline HEAD~3..HEAD` ist leer
      (kein Merge-Commit entstanden).
- [x] **SC-102b NEU** (Commit-Reihenfolge final im ERRATA-Abschnitt auf 4 Commits korrigiert)
      Die drei bean-Commits liegen nach dem Rebase inhaltlich unveraendert
      obenauf: `git log --oneline -3` zeigt in dieser Reihenfolge die Subjects
      `chore(beans): D19 test-gate + bt-xy2i scrapped`,
      `chore(beans): gate-b-nachtraege f01-f03`,
      `chore(beans): roadmap-tty-output operationalisiert`.

SC-103, SC-104, SC-105 bleiben **unveraendert** gueltig (featureGroup vorhanden,
`go test ./internal/commands/` ok, Push nach `fork` und niemals nach `origin`).

**Test-Gate dieses Tasks** ist `go test ./...` bzw. `go test ./internal/commands/`,
nicht `mise test` — siehe D19 im Epic-bean `beans-1ec3`.

## ERRATA 2026-07-23 (Supervisor, unmittelbar nach D20) — Commit-Zaehler korrigiert

Der D20-Nachtrag oben ist selbst eine `.beans/`-Aenderung und muss **vor** dem Rebase
committet werden (ein dirty Working Tree blockiert `git rebase`). Damit steigt die Zahl der
bean-Pflege-Commits auf main von drei auf **vier**. Der vierte ist:

- `chore(beans): D20 rebase-deviation T1` (dieser Nachtrag + ERRATA)

Verbindlich sind daher folgende Zahlen — sie ersetzen die Werte in SC-101 REVIDIERT,
SC-102 REVIDIERT und SC-102b:

- [x] **SC-101** Vor dem Rebase: linke Zahl = **4**, rechte Zahl = **8**. Abweichung der linken
  Zahl bedeutet fremde Commits auf main → abbrechen und PO fragen.
- [x] **SC-102** Nach dem Rebase: linke Zahl = **4**, rechte Zahl = **0**; und
  `git log --merges --oneline HEAD~4..HEAD` ist leer.
- [x] **SC-102b** `git log --oneline -4` zeigt von neu nach alt:
  `chore(beans): D20 rebase-deviation T1`,
  `chore(beans): D19 test-gate + bt-xy2i scrapped`,
  `chore(beans): gate-b-nachtraege f01-f03`,
  `chore(beans): roadmap-tty-output operationalisiert`.

## Summary

`fork/main` ist als Integrationsbasis hergestellt: die vier bean-Pflege-Commits von `main`
wurden per `git rebase fix/beans-ti53-roadmap-nested-hierarchy` auf die acht ti53-Commits
umgesetzt (lineare Historie, kein Merge-Commit — Intent von EARS-3 erfüllt, D20). `main` trägt
jetzt `featureGroup` und die drei weiteren Datenmodell-Typen aus `internal/commands/roadmap.go`.
Nach `fork` gepusht (`99260bf..c2f252d`), `origin` unberührt. Working Tree blieb tracked-clean,
der Stash (`pnpm autogen allowBuilds`) und die 16 untracked ti53-bean-Dateien wurden nicht
angefasst.

## Test-Output

```
$ git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy   # vor Rebase
4	8

$ git rebase fix/beans-ti53-roadmap-nested-hierarchy
Rebasing (1/4)Rebasing (2/4)Rebasing (3/4)Rebasing (4/4)Successfully rebased and updated refs/heads/main.

$ git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy   # nach Rebase
4	0

$ git log --merges --oneline HEAD~4..HEAD
(leer)

$ git log --oneline -4
c2f252d chore(beans): D20 rebase-deviation T1
3bbf1fa chore(beans): D19 test-gate + bt-xy2i scrapped
7ce8a9f chore(beans): gate-b-nachtraege f01-f03
68c1b55 chore(beans): roadmap-tty-output operationalisiert

$ grep -n 'type featureGroup' internal/commands/roadmap.go
62:type featureGroup struct {

$ command go test ./internal/commands/
ok  	github.com/hmans/beans/internal/commands	0.562s

$ git push fork main
To https://github.com/xRiErOS/beans.git
   99260bf..c2f252d  main -> main

$ git log origin/main..main --oneline
c2f252d chore(beans): D20 rebase-deviation T1
3bbf1fa chore(beans): D19 test-gate + bt-xy2i scrapped
7ce8a9f chore(beans): gate-b-nachtraege f01-f03
68c1b55 chore(beans): roadmap-tty-output operationalisiert
3419e8a refactor(roadmap): route orphan-feature classification through classifyFeatureChild
8cadcdf fix(roadmap): render childless feature as flat leaf instead of dropping it
7c71631 feat(roadmap): render Feature sections in Markdown output
c0bc49d fix(roadmap): resolve Feature nesting for unscheduled epics and orphan features
ca4ab06 test(roadmap): pin milestone-direct-feature inclusion line
78b2a6a fix(roadmap): resolve Feature nesting under Epic and Milestone
6fdb7b1 feat(roadmap): add splitByContainerType and collectLeafDescendants helpers
f0f31fb feat(roadmap): add featureGroup to data model
(12 Commits nicht auf origin — origin unberührt, wie gefordert)
```

## Deviations/ERRATA

- **D20/ERRATA (PO/Supervisor, bereits im bean dokumentiert vor Task-Start):** Integration per
  `git rebase` statt `git merge --ff-only` (EARS-3-Wortlaut abgewichen, Intent erfüllt — lineare
  Historie, kein Merge-Commit). Grund: `main` trug vier bean-Pflege-Commits, die ti53 nicht hatte;
  Fast-Forward war unmöglich. Verbindliche Zahlen aus dem ERRATA-Abschnitt (`4	8` vor, `4	0` nach
  Rebase) bestätigt durch tatsächlichen Kommando-Output oben.
- Keine weiteren Abweichungen. Test-Gate war `go test ./internal/commands/` (D19), `mise test`
  wurde nicht ausgeführt.
- Der Stash `pnpm autogen allowBuilds (vor T1-rebase)` wurde nicht angefasst; die 16 untracked
  `.beans/*.md`-Dateien aus dem ti53-Strang blieben unverändert untracked (16 vor und nach dem
  Task, per `git status --short` verifiziert).

## Notes for T2

- `main` (== `fork/main`) enthält jetzt `roadmapData`, `unscheduledGroup`, `milestoneGroup`,
  `epicGroup`, `featureGroup` in `internal/commands/roadmap.go` (Zeile 62 für `featureGroup`).
  Die maximale Render-Tiefe ist 4 (Milestone → Epic → Feature → Leaf), `featureGroup.Items` ist
  flach (Leafs unter verschachtelten Features hineingeflattet).
- `git log origin/main..main` zeigt 12 unpushed Commits gegen `origin` (hmans/beans) — das ist
  gewollt, PR #207 bleibt offen liegen (D14). Kein Push nach `origin` in Folge-Tasks.
- Die 16 untracked `.beans/*.md`-Dateien aus dem ti53-Strang liegen weiterhin im Working Tree —
  laut Auftrag versorgt der Supervisor sie separat, nicht anfassen.
- Der Stash `pnpm autogen allowBuilds (vor T1-rebase)` liegt weiterhin in `stash@{0}` — nicht
  poppen/droppen, das ist außerhalb dieses Tasks.
- Test-Gate bleibt `go test ./...` bzw. `go test ./internal/commands/` (D19), nicht `mise test`.
