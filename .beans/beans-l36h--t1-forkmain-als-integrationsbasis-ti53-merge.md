---
# beans-l36h
title: T1 fork/main als Integrationsbasis (ti53-Merge)
status: in-progress
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:47:02Z
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

- [ ] **SC-101 REVIDIERT** Vor dem Rebase gibt `git rev-list --left-right --count
      main...fix/beans-ti53-roadmap-nested-hierarchy` exakt `3	8` aus. Weicht die linke
      Zahl davon ab (fremde Commits auf main), **abbrechen und PO fragen**.
- [ ] **SC-102 REVIDIERT** Nach dem Rebase gilt beides:
      `git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy`
      gibt `3	0` aus, **und** `git log --merges -1 --oneline HEAD~3..HEAD` ist leer
      (kein Merge-Commit entstanden).
- [ ] **SC-102b NEU** Die drei bean-Commits liegen nach dem Rebase inhaltlich unveraendert
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

- **SC-101** Vor dem Rebase: linke Zahl = **4**, rechte Zahl = **8**. Abweichung der linken
  Zahl bedeutet fremde Commits auf main → abbrechen und PO fragen.
- **SC-102** Nach dem Rebase: linke Zahl = **4**, rechte Zahl = **0**; und
  `git log --merges --oneline HEAD~4..HEAD` ist leer.
- **SC-102b** `git log --oneline -4` zeigt von neu nach alt:
  `chore(beans): D20 rebase-deviation T1`,
  `chore(beans): D19 test-gate + bt-xy2i scrapped`,
  `chore(beans): gate-b-nachtraege f01-f03`,
  `chore(beans): roadmap-tty-output operationalisiert`.
