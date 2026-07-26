# SSTD — beans-src (Fork xRiErOS/beans)

Pointer-Manifest. Kein State-Block — der Work-State lebt in `.beans/`.

## Was dieses Repo ist

Erik-Fork von `hmans/beans`. Remotes: `fork` = xRiErOS (Produkt), `origin` = hmans (Upstream).
**Push nur nach `fork`.** Der Fork ist das Produkt, nicht der PR (D14, 2026-07-23).

**hmans hat beans öffentlich eingestellt (bestätigt 2026-07-24).** Upstream ist tot — wir
entwickeln beans selbst weiter, **keine Abhängigkeit mehr zu hmans**. `origin` wird nicht mehr
verfolgt, keine Upstream-PRs mehr. Fork-Delta ist ab jetzt erwünscht (auch in `CLAUDE.md`,
die damit **unsere** Datei ist — s. Fork-Header dort).

**Backlog-Schnitt 2026-07-24:** Der geerbte hmans-`.beans/`-Backlog wurde beschnitten — 90
irrelevante beans hart entfernt (`git rm`, Commit `ae6cf8c`). Behalten: 8 relevante Stränge
(Epics `mmyp`/`oe8n`/`oyic` + Children, Features `gkgc`/`ntus`/`8olg`/`iggk`, bug `36fa`)
plus der komplette Fork-eigene ti53/1ec3-Strang. Roadmap zeigt jetzt nur noch relevante Arbeit.

Das lokal installierte `/opt/homebrew/bin/beans` ist ein Build aus diesem Fork, **nicht** das
Homebrew-Cask (das wurde deinstalliert). Aktuell: **`0.4.3-fork.rename`** (rename-Command, PO-installiert
2026-07-24; Vorgänger `0.4.2-fork.tty`, Backup unter `/tmp/beans-backup-preRename-20260724`).
**Build-Hinweis:** `mise build` ist in der Agent-Sandbox unbrauchbar (Frontend-Codegen `pnpm install` stirbt SIGTERM).
CLI-Build direkt via `command go build -ldflags "$LDFLAGS" -o beans ./cmd/beans` (Frontend-Embed aus letztem
erfolgreichem Build). Kein formaler git-Version-Tag gesetzt — offene Release-Entscheidung.

## Aktueller Strang

| Thema | Ort | Stand |
|---|---|---|
| beans rename command | `docs/beans-rename-command/` | **PO-REVIEW abgeschlossen 2026-07-24 — Epos `beans-e040` Tag `accepted`, 10/10 US accepted, 0 Rejects.** Binary `0.4.3-fork.rename` global installiert. Follow-up-Epic `beans-a29l` (Hardening): `beans-6ap8` D01 stageAndSwap-Crash-Recovery, `beans-pmu1` D02 atomarer .beans.yml-Write. **Offen: Q01** (Co-Authored-By-Layer-Konflikt — noch keine PO-Entscheidung), formaler Version-Tag. (Historie:) **Realisierung war abgeschlossen — Epos war auf `to-review`.** 9/9 In-Scope-Leaves (T01–T08, T10) `completed` + je `ce-specs-reviewer`-GRÜN (alle erste Runde, kein CHANGES_REQUIRED). **T09 `beans-lok4` (renameBean GraphQL/UI) bewusst deferred/todo** — außerhalb dieses Runs (Direct-Core-Architektur, GraphQL optional). Voll-Gate grün (`command go test ./...` alle Pakete, vet clean, Binary `rename --help` OK). Feature-Doku in `beans-src/CLAUDE.md` §`# Renaming Beans`. **Offene PO-Punkte am Gate** (im Epic-bean verankert): D01 (stageAndSwap-Crash-Fenster, keine `.beans.bak-*`-Waisen-Recovery), D02 (`.beans.yml`-Config-Write nicht-atomar → Mixed-Prefix nach Teilausfall), Q01 (Co-Authored-By-Layer-Konflikt, Historie inkonsistent T01-06 mit / T07+ ohne). Lessons: LL-20…LL-23. |
| roadmap TTY-Output | `docs/roadmap-tty-output/` | Realisierung abgeschlossen, Epos `beans-1ec3` auf Tag `to-review` — wartet auf `/ce-po-review` |

- **Denk-Kette:** `docs/roadmap-tty-output/{DESIGN,DECISIONS,QUESTIONS,TASKS,REFERENCES}.md`
- **Plan:** `docs/roadmap-tty-output/PLAN.md` — `ce-plan-reviewer` grün (2 Runden),
  PO-freigegeben 2026-07-23, Gate-B-verifiziert
- **Layout-Referenz:** `docs/roadmap-tty-output/render-prototype.py` (ausführbar).
  **Nicht** der Quelltext-Block in `PLAN.md` Task 2 Step 1 — der ist lückenhaft (LL-12).
- **Arbeit:** Epos `beans-1ec3`, T1–T6 alle `completed` und `ce-specs-reviewer`-grün
  (T3/T4/T5 je in Runde 2, siehe LL-15/LL-16).

**Nächster Schritt: PO-Review** — `/ce-po-review beans-1ec3`. Das Epic trägt Tag `to-review`
und wartet auf Abnahme. **Ein Punkt ist nur am echten Terminal abnehmbar (R01):** rendert der
Terminal-Emulator die Glyphen `■ ▸ ▪` einspaltig? In tmux/pty ja — real nicht agentisch
belegbar. Bei doppelter Breite verschieben sich alle Spalten um 1.

**Offen aus diesem Epos:** bug `beans-36fa` (kinderlose Orphan-Epic fehlt in **beiden**
Ausgabepfaden, Ursache in `buildRoadmap`; ein Fix ändert den Markdown-Output und braucht
PO-Entscheid).

## Nicht-Ableitbarkeiten

### Zwei Kommandos, die auf dieser Maschine still das Falsche tun

Beide sind wiederholt aufgetreten. Jeder Agent-Dispatch in diesem Repo muss sie nennen —
ein Beweis aus einem dieser Kommandos ohne Gegenprobe ist wertlos.

- **`go` ist eine Shell-Funktion** (dotfiles-Sync), die den Compiler verdeckt. Ein blosses
  `go test ./...` ruft das Sync-Skript, endet mit **Exit 0** und führt **keinen einzigen Test**
  aus. Immer `command go test` / `command go build` / `command go vet`.
  (LL-02 vom 2026-07-17, erneut als D21 im Epic `beans-1ec3` am 2026-07-23 — der ursprüngliche
  Forward-Guard war nie verdrahtet worden, siehe LL-10.)
- **`awk` misst Bytes, nicht Zeichen.** `/usr/bin/awk` ist hier nicht multibyte-aware, trotz
  UTF-8-Locale. Bei Ausgaben mit Glyphen wie `■ ▸ ▪` meldet `awk '{print length($0)}'`
  **240 statt 80**. Für Breitenprüfungen `wc -m` oder Rune-Zählung in `command python3`.
  (D22, LL-11.)

### Weitere

- **`docs/` ist per `.git/info/exclude` von git ausgeschlossen** — bewusst, damit die Denk-Kette
  nicht in Upstream-PR-Diffs landet. `git add docs/...` schlägt fehl; das ist kein Fehler.
  Konsequenz: diese Datei und der Plan sind **nicht versioniert**.
- **`mise test` ist hier kein brauchbares Gate** — es zieht `test:e2e` mit, und der
  Playwright-Browser fehlt lokal (`browserType.launch: Executable doesn't exist`). Alle e2e-Specs
  failen in 0–1 ms als Setup-Fail, unabhängig vom Code. Backend-Gate ist `command go test ./...`.
  (D19.)
- **`CLAUDE.md` ist eine Upstream-Datei** (hmans/beans). Ergänzungen dort vergrössern das
  Fork-Delta und brauchen PO-Freigabe — repo-lokale Nicht-Ableitbarkeiten gehören deshalb
  hierher, nicht dorthin.
- Build-Target ist `./cmd/beans`, nicht das Repo-Root. Version-Stamp per ldflags auf
  `internal/version.{Version,Commit,Date}`.
- `brew install`/`upgrade` überschreibt das Fork-Binary — dann neu bauen und nach
  `/opt/homebrew/bin/beans` kopieren (Prozedur in bean `beans-f1t4`).
- Repo-Konventionen (mise, GraphQL-Codegen, Testkommandos) stehen in `CLAUDE.md`.

## Offene Fork-Delta-Stränge

- `fix/beans-ti53-roadmap-nested-hierarchy` — via T1 nach `main` gemerged. PR #207 upstream
  ist **obsolet** (hmans entwickelt nicht weiter, s.o.) — nicht mehr verfolgen.
