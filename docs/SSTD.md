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

Das lokal installierte `/opt/homebrew/bin/beans` ist ein Build aus diesem Fork, **nicht** das Homebrew-Cask (das wurde deinstalliert). Aktuell: **`0.4.4-fork`** (custom front matter + manual ordering, Epos `beans-xej5`, installiert 2026-08-10 über `just install`). Vorgänger informell `0.4.3-fork.rename` (rename-Command, PO-installiert 2026-07-24, Backup unter `/tmp/beans-backup-preRename-20260724`) und `0.4.2-fork.tty` — **keiner der beiden je als echter git-Tag gesetzt**. `0.4.4-fork` ist der **erste formale git-Tag** in diesem Repo (`git tag -a`, lokal, nicht gepusht); `0.4.3-fork` wurde kurz gesetzt und wieder gelöscht, weil die Nummer mit `0.4.3-fork.rename` kollidierte.

**Build/Install:** `just install` (baut + installiert nach `BEANS_BIN_DIR`, default `/opt/homebrew/bin` — siehe `.claude/rules/tools.md`; `mise` bleibt darunter die Build-Implementierung). **Build-Hinweis (Agent-Sandbox):** `mise build` kann dort unbrauchbar sein, wenn Frontend-Codegen `pnpm install` mit SIGTERM stirbt — dann CLI-Build direkt via `command go build -ldflags "$LDFLAGS" -o beans ./cmd/beans` (Frontend-Embed aus letztem erfolgreichem Build).

## Aktueller Strang

| Thema | Ort | Stand |
|---|---|---|
| beans rename command | `docs/beans-rename-command/` | **PO-REVIEW abgeschlossen 2026-07-24 — Epos `beans-e040` Tag `accepted`, 10/10 US accepted, 0 Rejects.** Binary `0.4.3-fork.rename` global installiert. Follow-up-Epic `beans-a29l` (Hardening): `beans-6ap8` D01 stageAndSwap-Crash-Recovery, `beans-pmu1` D02 atomarer .beans.yml-Write. **Offen: Q01** (Co-Authored-By-Layer-Konflikt — noch keine PO-Entscheidung), formaler Version-Tag. (Historie:) **Realisierung war abgeschlossen — Epos war auf `to-review`.** 9/9 In-Scope-Leaves (T01–T08, T10) `completed` + je `ce-specs-reviewer`-GRÜN (alle erste Runde, kein CHANGES_REQUIRED). **T09 `beans-lok4` (renameBean GraphQL/UI) bewusst deferred/todo** — außerhalb dieses Runs (Direct-Core-Architektur, GraphQL optional). Voll-Gate grün (`command go test ./...` alle Pakete, vet clean, Binary `rename --help` OK). Feature-Doku in `beans-src/CLAUDE.md` §`# Renaming Beans`. **Offene PO-Punkte am Gate** (im Epic-bean verankert): D01 (stageAndSwap-Crash-Fenster, keine `.beans.bak-*`-Waisen-Recovery), D02 (`.beans.yml`-Config-Write nicht-atomar → Mixed-Prefix nach Teilausfall), Q01 (Co-Authored-By-Layer-Konflikt, Historie inkonsistent T01-06 mit / T07+ ohne). Lessons: LL-20…LL-23. |
| roadmap TTY-Output | `docs/roadmap-tty-output/` | **PO-Review abgeschlossen 2026-08-10 — Epos `beans-1ec3` Tag `accepted`, 2/2 US accepted, 0 Rejects.** R01 (Glyphen-Rendering) live per tmux-PTY verifiziert, siehe Epic-Body. |
| beans planning primitives | `docs/beans-planning-primitives/BRIEFING.md` | **PO-Review abgeschlossen 2026-08-10 — Milestone `beans-xej5`, Epics `beans-2ark` + `beans-zb0r` je Tag `accepted`, 9/9 US accepted, 0 Rejects.** Binary `0.4.4-fork` global installiert. Milestone selbst bleibt `in-progress` — weitere Kinder außerhalb dieses Scopes offen: `beans-mmyp`, `beans-3dvs`, `beans-a29l`, `beans-36fa`, `beans-13ae`. |

**Nächster Schritt:** Milestone `beans-xej5` hat nach den drei abgeschlossenen Epics noch offene Kinder (`beans-mmyp`, `beans-3dvs`, `beans-a29l`, `beans-36fa`, `beans-13ae`, alle `todo`) — `beans list --ready --parent beans-xej5` für den Einstieg. `beans-36fa` (Orphan-Epic verschwindet aus `buildRoadmap`-Output) braucht vor einem Fix einen PO-Entscheid, da er den Markdown-Output ändert.

**Workspace (2026-08-13):** Der vollständige Checkout liegt in der Workspace-Hülle unter `repo/`; der gemeinsame private Store liegt ausschließlich in `../.beans`. `repo/.beans.yml` setzt `beans.path: ../.beans`, die äußere `.envrc` zusätzlich `BEANS_PATH`. Die Entfernung der ehemals getrackten `repo/.beans/`-Dateien gehört zum Workspace-Split-Commit; Wrapper-Dateien selbst werden nicht im Fork versioniert.

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

- **Korrigiert 2026-08-10 — der `docs/`-Ausschluss-Eintrag hier war veraltet/falsch:** `docs/` ist normal git-getrackt (`git ls-files docs/` listet es, `git add docs/SSTD.md` griff anstandslos), **kein** `.git/info/exclude`-Eintrag existiert dafür. Diese Datei und der Plan sind versioniert.
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
