---
# beans-txez
title: 'T10 rename: documentation'
status: completed
type: task
priority: low
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:40:30Z
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



## Summary of Changes
- [x] SC-001: `beans-src/CLAUDE.md` new "Renaming Beans" section documents all 3 modes (slug/id/prefix), flags, dry-run/--yes-confirm flow, prefix guards (server-running + active-worktree), and the two non-goals (no auto-git/D10, external refs not rewritten/D11).
- [x] I01 (T08-review prelude): `--json` two-document-stdout-stream behavior on apply documented explicitly, verified against the built binary for both id and prefix modes (plan doc + separate result doc, not one JSON object).
- Verification: built `./cmd/beans`, ran `beans rename --help` + live dry-run/apply/--json/guard runs in a scratch `.beans` project (not this repo's own data) to ground every claim in actual binary output, not the plan. `go test ./pkg/beancore/... ./internal/commands/...` green (no code touched — docs-only change).
- Deviation: none from PLAN.md Task 10 — placed the section in CLAUDE.md (not README), per the plan's stated preference and because CLAUDE.md already documents other architecture/behavior sections (Worktree State Architecture, Agent Architecture) in this same style; README stays a high-level pointer to `beans help`.
- Note: `docs/` is git-ignored in this repo (`.git/info/exclude`) — not used as the doc target; CLAUDE.md is git-tracked.
