---
# beans-txez
title: 'T10 rename: documentation'
status: todo
type: task
priority: low
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:37:34Z
parent: beans-e040
blocked_by:
    - beans-xsai
---

Document rename modes + non-goals. Plan Task 10.

## Objective
Als künftiger Nutzer/Agent will ich die `beans rename`-Konvention und ihre Non-Goals dokumentiert, damit die Grenzen (kein auto-git, externe Refs out of scope, Rebrand-Guards) klar sind.

## EARS
- WHEN die Doku aktualisiert ist, THE SYSTEM (Doku) SHALL die drei Modi, die Plain-FS-Natur (beans committet nie selbst, D10), den Non-Goal externe-ID-Refs (D11) und die Rebrand-Guards (D05) beschreiben.

## Success Criteria
- SC-001: `beans-src/CLAUDE.md` (oder README) enthält einen "Renaming beans"-Abschnitt mit den 3 Modi + Non-Goals.

## Betroffene Pfade
- `beans-src/CLAUDE.md`. Details: PLAN.md Task 10.

## Prelude aus T08-Review (Supervisor, 2026-07-24)
- I01 (low): `beans rename --json` bei Apply (non-dry-run) schreibt ZWEI getrennte JSON-Dokumente hintereinander auf stdout (Plan-JSON aus renderRenamePlan, dann Result-JSON {"success":...}). Für Scripting-Konsumenten mit einfachem json.loads(stdout) ein Multi-Document-Stream. Bei der Doku dieses Verhalten explizit dokumentieren (oder als Follow-up-bean für sauberes Single-Doc-JSON vermerken). SPEC.md spezifiziert --json-Shape nicht präzise — Doku-Chance.
