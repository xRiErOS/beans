---
# beans-xsai
title: 'T08 rename: beans rename CLI command'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T10:11:50Z
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
