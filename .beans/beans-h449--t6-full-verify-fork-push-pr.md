---
# beans-h449
title: T6 Full-Verify + Fork-Push + PR
status: completed
type: task
priority: normal
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T22:15:17Z
parent: beans-en7i
blocked_by:
    - beans-4q4t
---

Plan Task 6. Verifikation + Upstream-PR. PR-Erstellung = PO-Gate (sichtbare externe Aktion, erst nach Freigabe).

Akzeptanz:
- [x] mise test / go test ./... gruen
- [x] gofmt -l clean, go vet clean
- [x] Verifiziert gegen echtes beans-tui .beans (zuvor gedroppte Epics erscheinen wieder) — siehe T6-Ergebnis-Sektion unten, inkl. Fund B01
- [ ] Branch nach fork gepusht, PR gegen hmans/beans offen (PO-Gate)
- [ ] beans-ti53 auf completed (PO-Gate)

## PRELUDE (Supervisor, 2026-07-17, Quelle: T2-Review) — gofmt-Gate-Korrektur

Plan Task 6 Step 2 fordert `command gofmt -l internal/commands/roadmap.go internal/commands/roadmap_test.go` → leer erwartet. **Das ist so nicht haltbar:** `roadmap_test.go` ist BEREITS auf origin/main gofmt-unclean (vorbestehende Struct-Tag-Ausrichtung in TestBuildRoadmap/TestFirstParagraph, ~Z33-131). Verifiziert via `git show origin/main:internal/commands/roadmap_test.go | gofmt -l` → unclean.

**Regel für T6:** Upstream-Code NICHT umformatieren (sonst unerwünschte Formatierungs-Churn im PR-Diff an hmans/beans). Das gofmt-Gate NUR auf `internal/commands/roadmap.go` anwenden (unsere Änderungen, clean). `roadmap_test.go` nur auf die von UNS angehängten Test-Funktionen prüfen (ans Dateiende, gofmt-clean), nicht die Datei als Ganzes. `command go vet` + Voll-Testlauf bleiben Pflicht.

## PRELUDE-2 (Supervisor, 2026-07-17, Quelle: T5-ERRATA + Gate-Politik)

**Korrigierte Kommandos (Plan Task 6 ist hier falsch):**
- Binary-Build: `command go build -o /tmp/beans-fixed ./cmd/beans` — NICHT `go build ... .` (kein root main.go). Und IMMER `command go` (go geshadowed).
- Fixtures: `beans init` hat KEIN `-y`-Flag und schreibt `.beans`/`.beans.yml` ins cwd → `mkdir -p <dir> && (cd <dir> && beans init)`, danach `--beans-path <dir>/.beans` auf allen create/roadmap-Calls.

**gofmt-Gate (aus PRELUDE-1):** NUR `command gofmt -l internal/commands/roadmap.go` (unsere Datei, clean). `roadmap_test.go` ist package-weit vorbestehend upstream-unclean → NICHT prüfen/umformatieren. `command go vet ./...` + `command go test ./...` bleiben Pflicht.

**GATE-TEILUNG (zwingend):** T6-Agent führt NUR die Verifikation + Real-Repo-Smoke + PR-Body-Entwurf aus und STOPPT. `git push` + `gh pr create` an hmans/beans sind extern/irreversibel = **PO-Gate** (Erik gibt frei). Agent setzt NICHT `beans-ti53` completed und pusht NICHT. Agent setzt am Ende das EPIC beans-en7i auf Tag `to-review` (NIE completed, §6.7) und liefert den fertigen PR-Body-Text + Bestätigung „branch ready".



## T6-Ergebnis (Implementer-Agent, 2026-07-17)

**Voll-Suite ×2** (`command go test ./... -count=1`, dist-Stub bereits vorhanden): beide Läufe grün, keine Flakes. Alle Packages `ok` (u.a. `internal/commands 0.735s` / `0.617s`, `internal/gitutil ~12s` beide Läufe, `internal/tui`, `pkg/bean`, `pkg/beancore` etc. — komplette Liste im Supervisor-Report).

**Repo-Gates:** `command go vet ./...` exit 0, keine Ausgabe. `command gofmt -l internal/commands/roadmap.go` leer (clean).

**Real-Repo-Smoke** (Fix-Binary vs. Baseline aus `origin/main`, beide gegen `/Users/erik/Obsidian/tools/beans-tui/beans-tui-repository/.beans`, read-only): kein Crash/Panic auf beiden Seiten (exit 0/0). beans-tui hat aktuell keine tiefe Feature-Nesting unter E1-E8 (diese sind bereits `completed`/archiviert, nur E9/bt-tct9 ist offen) — die eigentliche Epic-Nesting-Fix-Wirkung zeigt sich hier NICHT (Nicht-Regress-Nachweis, kein Positiv-Nachweis).

**Diff (base → fix):**
```
24,28d23
< ### Miscellaneous
<
<
< - ![feature](...) Tag-Management-Page (zentrale Tag-Definition) ([bt-6oyy](.beans/bt-6oyy--...))
<
```

**Fund B01 (medium, code-gelesen + empirisch bestätigt):** Der Fix routet Feature-typisierte Beans IMMER über `buildFeatureGroup`/das Container-Modell (`unscheduledFeatures`, `eg.Features`, `group.Features`) und verwirft sie, sobald `len(fg.Items) == 0` — überall (orphan, unter Epic, unter Milestone), nicht nur im Orphan-Fall. Vor dem Fix wurden Feature-Beans wie jeder andere Bean als flache Zeile in `Other`/Miscellaneous gerendert, unabhängig von eigenen Kindern (siehe `origin/main:internal/commands/roadmap.go` `buildMilestoneGroup`, `other`-Liste ohne Feature-Ausschluss). Empirisch: `bt-6oyy` (Feature, `in-progress`, kein Parent, 0 Kinder — verifiziert via `beans show`) erscheint in der Baseline unter „Miscellaneous", im Fix-Output gar nicht mehr. Das ist über die im Plan antizipierte „legitimately absent"-Klausel hinaus (die bezog sich auf Epics ohne offene Nachkommen) — hier verschwindet ein kinderloser, offener Feature-Bean komplett aus dem Roadmap-Output, wo er vorher sichtbar war. Kein Unit-Test deckt „orphan/parented Feature ohne Kinder" ab (`roadmap_test.go` hat nur den Fall „orphan bean" für Typ `task`, keinen für Typ `feature`). Nicht selbst gefixt (Verifikations-Scope) — PO-Entscheid nötig: Verhalten akzeptieren (Konsistenz mit Epic/Milestone-Semantik: leere Container werden ausgeblendet) oder Nachfolge-Task für „kinderlose Features weiter als Blattzeile rendern".

**Commits (origin/main..HEAD, 6):**
```
7c71631 feat(roadmap): render Feature sections in Markdown output
c0bc49d fix(roadmap): resolve Feature nesting for unscheduled epics and orphan features
ca4ab06 test(roadmap): pin milestone-direct-feature inclusion line
78b2a6a fix(roadmap): resolve Feature nesting under Epic and Milestone
6fdb7b1 feat(roadmap): add splitByContainerType and collectLeafDescendants helpers
f0f31fb feat(roadmap): add featureGroup to data model
```

**PR-Body:** fertig entworfen, im Supervisor-Report (nicht hier dupliziert) — Summary-Bullet zu B01 ergänzt.

**Cleanup:** `/tmp/t6-base` Worktree entfernt, alle Temp-Binaries/-Outputs gelöscht. `git worktree list` zeigt nur Main-Clone + diesen Worktree.

**Status dieses beans:** bleibt `in-progress` — Push/PR-Checkbox + `beans-ti53`-Completion sind PO-Gate, nicht von diesem Agent gesetzt.

## Push/PR erledigt 2026-07-18
Branch nach fork xRiErOS/beans gepusht, PR an hmans/beans: https://github.com/hmans/beans/pull/207. Pre-Push-Gate: go test ./... gruen, vet/gofmt clean, 8 Commits.
