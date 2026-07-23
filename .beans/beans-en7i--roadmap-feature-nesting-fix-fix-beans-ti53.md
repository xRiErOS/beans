---
# beans-en7i
title: roadmap Feature-Nesting-Fix (fix beans-ti53)
status: todo
type: epic
priority: high
tags:
    - roadmap
    - accepted
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T22:15:17Z
---

Sprint: fix beans roadmap so Milestone -> Epic -> Feature -> Leaf resolves cleanly. Full-scope fix (featureGroup in data model + template), shipped as upstream PR to hmans/beans.

Plan (local, not in PR diff): docs/plans/beans-ti53-roadmap-nested-hierarchy/plan.md
Bug: beans-ti53. Worktree: ../beans-src-worktrees/fix-ti53 (branch fix/beans-ti53-roadmap-nested-hierarchy). Fork remote: fork -> github.com/xRiErOS/beans.
Adversarial Opus plan-review passed; 2 medium + 5 low findings all patched into the plan before this breakdown.

Execution: TDD, one task per plan-Task, sequential (blocked_by chain). Merge/PR = PO gate.

## PF-Preflight (Supervisor, 2026-07-17) — Umgebungs-Invarianten für ALLE Agents

Diese Fakten sind NICHT ableitbar und dürfen NICHT ins committete CLAUDE.md/PR (nur lokale Env):

- **REPO-SPLIT (kritisch):** Code-Arbeit + git-Commits laufen im **Worktree** `/Users/erik/Obsidian/tools/lean-stack/beans-src-worktrees/fix-ti53` (Branch `fix/beans-ti53-roadmap-nested-hierarchy`). bean-State (dieses Epic + T1-T6 + Bug beans-ti53) liegt im **Main-Clone** `.beans/`. Jeder `beans`-Befehl MUSS `--beans-path /Users/erik/Obsidian/tools/lean-stack/beans-src/.beans` tragen — sonst greift das Worktree-lokale `.beans/` (kennt diese beans NICHT). Grund: tracking-beans dürfen nie in den Upstream-PR-Diff zu hmans/beans.
- **`go` ist Shell-Shadowed** (Funktion macht git pull-Rauschen). IMMER `command go` (build/test/vet/fmt).
- **Test-Substrat braucht dist-Stub:** `internal/web/embed.go` hat `//go:embed dist/*`; ohne gebautes Frontend kompiliert KEIN Backend-Package. Stub existiert bereits lokal (`internal/web/dist/index.html`, gitignored). Falls weg: `mkdir -p internal/web/dist && printf %s "<!doctype html>" > internal/web/dist/index.html`. Danach `command go test ./internal/commands/` grün (~0.6s warm).
- **git add nur explizite Source-Pfade** (`internal/commands/...`), NIE `.beans` oder `internal/web/dist`. `docs/` ist lokal via `.git/info/exclude` ausgeschlossen.
- **Plan = Quelle der Wahrheit** (kein separates design-spec.md): `docs/plans/beans-ti53-roadmap-nested-hierarchy/plan.md` im Worktree. Adversarial-Opus-review-gehärtet (2 medium + 5 low Findings eingearbeitet).
- **jq-Shapes:** `beans show --json` → bean top-level; `create --json` → {success,bean:{...}}; `list --json` → array.

## Review 2026-07-17
NB · Kosmetik: "#### Feature"-Sektion klebt ohne Leerzeile nach einem Leaf bzw. nach "## No Milestone" (0 statt 1 Leerzeile) · PO akzeptiert as-is — rein optisch, feature-lose Ausgabe bleibt byte-identisch (Garantie-Constraint verhindert sicheren Einzeiler). Kein Arbeits-bean.

US-01 · Leaf unter Feature unter Epic unter Milestone erscheint (#### Feature-Sektion) · a → beans-d223/beans-4q4t
US-02 · Epic mit nur feature-genesteter Arbeit verschwindet nicht mehr · a → beans-d223
US-03 · Orphan-Feature mit Kindern rendert unter "No Milestone" · a → beans-4lui
US-04 · Feature direkt unter Milestone rendert seine Leafs · a → beans-d223

US-05 · Kinderloses offenes Feature erscheint als flache Zeile (B01-Fix) · a → beans-n8zw
US-06 · Feature-lose Ausgabe byte-identisch (kein Regress) · a → beans-4q4t

## PR 2026-07-18
https://github.com/hmans/beans/pull/207 (8 Commits inkl. B01-Fix + I01-Refactor). Epic accepted.
