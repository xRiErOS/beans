---
# beans-4q4t
title: T5 Template — Feature-Sektionen + Whitespace-Regression-Check
status: completed
type: task
priority: high
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T21:12:52Z
parent: beans-en7i
blocked_by:
    - beans-4lui
---

Plan Task 5. roadmap.tmpl.

Akzeptanz:
- [ ] featureGroup-Template-Block (#### Feature:) referenziert aus epicGroup/milestone/unscheduled
- [ ] beide neuen .Features-Ranges left-trimmed ({{- range .Features -}})
- [ ] Miscellaneous-Guards nutzen or (len .Epics) (len .Features)
- [ ] Feature-Fixture rendert genestet; Feature-lose Fixture byte-identisch zu origin/main (diff leer)


[x] featureGroup-Template-Block (#### Feature:) referenziert aus epicGroup/milestone/unscheduled
[x] beide neuen .Features-Ranges left-trimmed ({{- range .Features -}})
[x] Miscellaneous-Guards nutzen or (len .Epics) (len .Features)
[x] Feature-Fixture rendert genestet; Feature-lose Fixture byte-identisch zu origin/main (diff leer)

## Summary (2026-07-17)

Task 5 aus dem Plan umgesetzt: `internal/commands/roadmap.tmpl` komplett durch den Plan-Block ersetzt. Neu: `featureGroup`-Template-Define (`#### Feature: ...`), referenziert aus `epicGroup` (nach `.Items`) sowie aus dem Milestone-Block und dem Unscheduled-Block. Beide neuen `{{range .Features -}}`-Blöcke sind left-trimmed. Die beiden Miscellaneous-Guards (`.Other`) wurden von `{{- if len .Epics}}` / `{{- if len .Unscheduled.Epics}}` auf `{{- if or (len .Epics) (len .Features)}}` / `{{- if or (len .Unscheduled.Epics) (len .Unscheduled.Features)}}` erweitert.

## Test-Output (2026-07-17)

Feature-Fixture-Render (Milestone -> Epic -> Feature -> Leaf task, via echtes Binary `/tmp/beans-fix` aus `./cmd/beans`):

```
# Roadmap

## Milestone: Milestone ([roadmap-fixture-hu9i](./roadmap-fixture-hu9i--milestone.md))

### Epic: Epic ([roadmap-fixture-tp4x](./roadmap-fixture-tp4x--epic.md))


#### Feature: Feature ([roadmap-fixture-44n3](./roadmap-fixture-44n3--feature.md))


- ![task](https://img.shields.io/badge/task-1d76db?style=flat-square) Leaf task ([roadmap-fixture-l8fl](./roadmap-fixture-l8fl--leaf-task.md))
```

Reihenfolge wie erwartet: `## Milestone:` -> `### Epic:` -> `#### Feature:` -> `- ... Leaf task`.

Regression-Diff (Baseline aus origin/main via temp-worktree `/tmp/beans-baseline`, Feature-lose Fixture Milestone -> Epic -> task):

```
$ diff <(/tmp/beans-baseline-bin roadmap --beans-path /tmp/roadmap-plain-fixture) \
       <(/tmp/beans-fix roadmap --beans-path /tmp/roadmap-plain-fixture)
DIFF_EXIT=0
```
(leer — keine Whitespace-Regression im Feature-losen Fall.)

Package-Testlauf:

```
$ command go test ./internal/commands/ -count=1
ok  	github.com/hmans/beans/internal/commands	0.565s
$ command go vet ./internal/commands/...
(exit 0, keine Ausgabe)
```

## Smoke

Echtes Binary (`command go build -o /tmp/beans-fix ./cmd/beans`), echte Fixtures via `beans init`/`beans create`/`beans roadmap` gegen echte `.beans/`-Verzeichnisse unter `/tmp/roadmap-fixture` und `/tmp/roadmap-plain-fixture`. Baseline-Binary analog aus `/tmp/beans-baseline` (Temp-Worktree `origin/main`, dist-Stub ergänzt). Beide Temp-Artefakte (Fixtures, Binaries, Worktree) nach Gebrauch aufgeräumt (`git worktree remove /tmp/beans-baseline` erfolgreich, `git worktree list` zeigt nur Haupt-Worktree + Main-Clone).

## Deviations/ERRATA (2026-07-17)

- **ERRATUM 1 (Build-Kommando):** Plan-Step-2/3-Snippets nutzen `go build -o ... .` (repo-root als Main-Package). Dieses Repo hat keinen `main.go` im Root — die Main-Packages liegen unter `cmd/beans`, `cmd/beans-serve`, `cmd/beans-tui` (siehe `mise.toml` Task `beans`: `go run ./cmd/beans`). Verwendet: `command go build -o /tmp/beans-fix ./cmd/beans` bzw. `... ./cmd/beans` für den Baseline-Build. Betrifft Step 2 und Step 3 gleichermaßen.
- **ERRATUM 2 (init-Flag):** Plan-Snippet nutzt `init -y --beans-path <dir>`. `beans init` kennt kein `-y`-Flag und legt `.beans`/`.beans.yml` im aktuellen Arbeitsverzeichnis an (nicht via `--beans-path` steuerbar für `init` selbst). Verwendet: `mkdir -p <dir> && cd <dir> && beans init` (ohne `-y`, ohne `--beans-path`); alle nachfolgenden `create`/`roadmap`-Aufrufe nutzen weiterhin `--beans-path <dir>` wie im Plan. Betrifft Step 2 (Feature-Fixture) und Step 3 (Plain-Fixture, Baseline-Binary).
- Alle übrigen Plan-Snippets (Template-Inhalt, `diff`-Aufbau, Cleanup, Commit-Message) 1:1 übernommen, keine weiteren Abweichungen.

## Notes for T6 (2026-07-17)

Zustand vor Final-Verify+PR (Task 6): Alle 5 Fix-Commits stehen im Worktree-Branch `fix/beans-ti53-roadmap-nested-hierarchy` (aktuell HEAD `7c71631`, davor `c0bc49d`, `ca4ab06`, ...). Template `internal/commands/roadmap.tmpl` ist final (featureGroup + beide Features-Ranges + erweiterte Miscellaneous-Guards). Datenmodell (roadmap.go) aus T1-T4 unverändert von T5 berührt — nur das Template wurde angefasst. T6 muss: (1) `mise test`/`go test ./...` volles Repo, (2) `gofmt -l`/`go vet ./internal/commands/...` repo-weit, (3) Verifikation gegen echtes beans-tui-Repo (`/Users/erik/Obsidian/tools/beans-tui/beans-tui-repository/.beans`) — dabei ebenfalls die Build-Pfad-ERRATA (`./cmd/beans` statt `.`) beachten, (4) Push zu Fork + PR gegen `hmans/beans`, (5) `beans-ti53` (das Bug-bean, nicht dieses Task-bean) auf `completed` setzen. Working tree ist clean, kein Temp-Worktree übrig.
