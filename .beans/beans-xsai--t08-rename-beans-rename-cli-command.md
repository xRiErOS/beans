---
# beans-xsai
title: 'T08 rename: beans rename CLI command'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:32:24Z
parent: beans-e040
blocked_by:
    - beans-9yif
---

Cobra command, flags, dry-run, --yes, --json. Plan Task 8.

## Objective
Als CLI-Nutzer will ich `beans rename` mit allen Modi und Sicherheits-Flags (dry-run überall, --yes-Confirm beim Rebrand, --json), sodass ich Änderungen vorab sehe und kontrolliert anwende.

## EARS
- WHEN `buildRenamePlan` Flags/Args erhält, die mehr als einen Modus anfordern, THE SYSTEM SHALL einen Konflikt-Fehler zurückgeben (Mutual-Exclusivity).
- WHEN `--suffix` auf einer ID ohne den konfigurierten Prefix genutzt wird, THE SYSTEM SHALL verweigern statt eine korrupte ID zu bilden.
- WHEN `--dry-run` gesetzt ist, THE SYSTEM SHALL den Plan rendern und OHNE Mutation zurückkehren.
- WHEN Modus `prefix` UND nicht `--yes`, THE SYSTEM SHALL vor Ausführung eine Bestätigung einholen.

## Success Criteria
- SC-001: `go test ./internal/commands/ -run TestBuildRenamePlan` GRÜN — Dispatch aller Modi, Mutual-Exclusivity, --suffix-Wrong-Prefix-Refusal.
- SC-002: Command via `RegisterRenameCmd` in `register.go` registriert (nicht root.go); nutzt package-level `core`.

## Betroffene Pfade
- `internal/commands/rename.go` (create), `internal/commands/register.go` (+RegisterRenameCmd), `internal/commands/rename_test.go`. Details: PLAN.md Task 8.

## Prelude aus T07-Review (Supervisor, 2026-07-24)
- I01 (low, aber jetzt relevant): T07 verdrahtet die Guards (checkServerNotRunning/checkNoActiveWorktrees) nur in `PlanRebrand` (Dry-Run). Zwischen Plan und `ApplyRename`/`applyRenameCascade` (tatsächliches Staging+Swap) öffnet sich ein TOCTOU-Fenster — besonders wenn dieser Task (T08) einen interaktiven `--yes`-Confirm einbaut (User bestätigt → in der Zwischenzeit startet ein `beans serve`). EMPFEHLUNG: Guards unmittelbar VOR dem echten Staging/Swap (in ApplyRename bzw. direkt vor dem Apply-Call im CLI-Command) erneut aufrufen, nicht nur beim Plan. Dann ist das Fenster geschlossen.
- Q01 (akzeptiert): Server-Detection ist Port-Heuristik (net.DialTimeout 127.0.0.1:port) — kann False-Positive bei fremdem Prozess auf dem Port liefern. Bewusst dokumentiertes Restrisiko, kein Handlungszwang.

## Todos
- [x] `internal/commands/rename.go` — Cobra command `beans rename`, drei Modi (slug/id/prefix), `--dry-run`, `--yes`, `--json`
- [x] `internal/commands/register.go` — `RegisterRenameCmd(root)` in `RegisterCoreCommands` verdrahtet
- [x] `internal/commands/rename_test.go` — RED→GREEN, `buildRenamePlan`-Dispatch + I01-Regressionstests
- [x] I01 adressiert: `applyRenameWithGuards` re-checkt D05-Guards unmittelbar vor dem Apply

## Summary of Changes
- `internal/commands/rename.go` (neu): `RegisterRenameCmd` registriert `beans rename [id] [new-id]` mit Flags `--slug`/`--no-slug`/`--reslug`/`--suffix`/`--prefix`/`--dry-run`/`--yes`/`--json`. `buildRenamePlan(core, args, flags)` ist der testbare, cobra-freie Kern: mapped Flags/Args auf genau einen `beancore.RenamePlan`-Modus (Mutual-Exclusivity via `modeCount`), verweigert `--suffix` auf IDs ohne konfigurierten Prefix. `renderRenamePlan` gibt den Plan als Tabelle oder `--json` aus; `confirmRename` liest ein y/N vom Command-Input.
- `internal/commands/register.go`: `RegisterRenameCmd(root)` in `RegisterCoreCommands` ergänzt (alphabetisch zwischen Prime und Roadmap).
- `internal/commands/rename_test.go` (neu): `TestBuildRenamePlan_dispatch` (9 Subtests: alle drei Modi + Mutual-Exclusivity + No-Args), `TestBuildRenamePlan_suffixWrongPrefixErrors`, plus zwei I01-Regressionstests (`TestApplyRenameWithGuards_reChecksGuardsBeforeApply`, `TestApplyRenameWithGuards_nonPrefixModeSkipsGuards`).
- **I01 (Guards-TOCTOU) adressiert:** `applyRenameWithGuards(c, plan)` — für `Mode=="prefix"` wird der Plan unmittelbar vor dem echten `ApplyRename`-Aufruf per `c.PlanRebrand(plan.NewPrefix)` neu berechnet; das re-invoked `checkServerNotRunning`/`checkNoActiveWorktrees` als ersten Akt und schließt so das TOCTOU-Fenster zwischen Plan (ggf. vor einem wartenden `--yes`-Prompt) und Apply. Slug/Einzel-ID-Modi sind laut D05-Scope nicht guard-pflichtig und laufen unverändert durch (Test: `TestApplyRenameWithGuards_nonPrefixModeSkipsGuards`).
- **Deviation vom PLAN.md-Wortlaut (begründet):** die im Plan-Snippet für Task 8 gezeigten Test-Fixtures (`"# id\n---\n...\n---\n"`) folgen NICHT dem tatsächlichen `Bean.Render()`-Format (`---\n# id\n...\n---\n`, siehe bereits gemergte `pkg/beancore/rename_test.go` aus T01–T07). Beim manuellen CLI-Smoke-Test mit der Plan-Reihenfolge parsten Title/Status leer. Fixtures in `rename_test.go` auf das kanonische Format korrigiert; funktional beeinflusste dies keinen Testfall (ID kommt aus dem Dateinamen), aber Konsistenz mit der SSTD-Fixture-Konvention war geboten.
- Kein GraphQL, kein Docs — außerhalb T08-Scope (T09/T10).

## Validation
- `command go test ./... -count=1` — alle Pakete grün (inkl. `internal/commands`, `pkg/beancore`, `pkg/bean`).
- `command go vet ./...` — clean.
- `command go build -o /tmp/beans-t08 ./cmd/beans && /tmp/beans-t08 rename --help` — Command im gebauten Binary vorhanden, alle Flags gelistet.
- Manueller End-to-End-Smoke (Scratch-Repo, korrekte Fixtures): dry-run slug (keine Mutation), echtes ID-Rename mit Ref-Kaskade (`parent`+`blocked_by` in Child-Bean korrekt auf neue ID umgeschrieben), Prefix-Rebrand dry-run, `n`-Abbruch (State unverändert), `--yes`-Apply (Dateien + `.beans.yml`-Prefix geschrieben), Mutual-Exclusivity-Fehler, `--suffix`-Wrong-Prefix-Guard — alle wie erwartet.
