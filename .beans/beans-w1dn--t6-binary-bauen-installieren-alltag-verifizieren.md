---
# beans-w1dn
title: T6 Binary bauen, installieren, Alltag verifizieren
status: in-progress
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T22:15:21Z
parent: beans-1ec3
blocked_by:
    - beans-zb00
---

**Plan-Referenz:** `docs/roadmap-tty-output/PLAN.md` → Task 6. Alle Kommandos stehen dort.

## Objective (User Story)

Als Erik will ich `beans roadmap` in **jedem** meiner Repos gerendert sehen — nicht nur in der
Test-Suite. Ohne diesen Task wirkt die gesamte Arbeit nicht (D14: der Fork ist das Produkt,
nicht der PR; Definition-of-Done ist das installierte Binary).

## Hintergrund

Die Build- und Installationsprozedur stammt aus dem abgeschlossenen Epic `beans-f1t4` und seinen
Tasks `beans-lk3p` (Build) und `beans-d7y9` (Install):

- Build-Target ist `./cmd/beans`, **nicht** das Repo-Root.
- Version-Stamp per ldflags auf `internal/version.{Version,Commit,Date}`.
- Der eingebettete Frontend-Stub unter `internal/web/dist/` muss vorhanden sein, damit das
  Backend kompiliert — er ist es bereits (`index.html`, `_app/`).
- Fork als echte Datei nach `/opt/homebrew/bin/beans` kopieren, **kein** Symlink.
- Das offizielle Homebrew-Cask wurde bereits deinstalliert (`brew uninstall --cask beans`).
- Neue Version: `0.4.2-fork.tty` (Vorgänger: `0.4.2-fork.ti53`).

Aufräumen: `bt-xy2i` ist ein in-progress-Duplikat des completed Epics `beans-f1t4` — gleicher
Titel, gleicher Body, im falschen ID-Namensraum angelegt (I01).

## EARS-Anforderungen

- **EARS-1** THE Binary SHALL aus `./cmd/beans` mit ldflags-Version-Stamp gebaut werden und
  `0.4.2-fork.tty` melden.
- **EARS-2** THE gepipte Ausgabe des neuen Binaries SHALL byte-identisch zur gepipten Ausgabe des
  bisher installierten Binaries sein.
- **EARS-3** IF der Pipe-Diff eine Abweichung zeigt, THEN THE Agent SHALL **nicht** installieren
  und stattdessen melden (D02/Q07 verletzt).
- **EARS-4** WHEN das Terminal 80 Spalten breit ist, THEN THE Ausgabe SHALL keine Zeile über
  80 Zeichen enthalten (Umbruch-Falle bei Grenzbreite).
- **EARS-5** THE installierte Binary SHALL in fremden Repos (`beans-tui`, `lean-stack`) ohne
  Fehler die gerenderte Tabelle liefern.
- **EARS-6** THE `.beans/`-Dateien SHALL nur mit expliziten Einzelpfaden gestaget werden, nie per
  Glob — das Repo trägt fremde uncommittete bean-Änderungen.
- **EARS-7** THE Agent SHALL nach `fork` pushen und niemals nach `origin`.

## Akzeptanzkriterien

- [ ] **SC-601** `mise test` ohne `FAIL` (Vorbestehende Fehler außerhalb `internal/commands` per
      `git stash` gegenprüfen — nur eigene Regressionen sind Blocker).
- [ ] **SC-602** `/tmp/beans-fork version` meldet `beans 0.4.2-fork.tty (<sha>) built <datum>`.
- [ ] **SC-603** `/tmp/beans-fork roadmap` am echten Terminal zeigt `Roadmap`-Kopf, `═`-Linie,
      Glyphen `■ ▸ ▪ -`, keine `img.shields.io`, keine `](`.
- [ ] **SC-604** tmux-Smoke bei 80 Spalten: `awk '{print length($0)}' | sort -rn | head -3`
      liefert keinen Wert über 80.
- [ ] **SC-605** `diff /tmp/roadmap-old.md /tmp/roadmap-new.md` meldet keine Unterschiede
      (`IDENTISCH`).
- [ ] **SC-606** `which beans` = `/opt/homebrew/bin/beans`, `beans version` = `0.4.2-fork.tty`.
- [ ] **SC-607** `beans roadmap` in `beans-tui-repository` und in `lean-stack` liefert die
      gerenderte Tabelle ohne Fehler.
- [ ] **SC-608** `bt-xy2i` hat Status `scrapped` und im Body eine `## Reasons for Scrapping`-
      Sektion mit Verweis auf `beans-f1t4`.
- [ ] **SC-609** Commit `chore(beans): bt-xy2i scrapped (dup of beans-f1t4)` mit explizitem
      Einzelpfad.
- [ ] **SC-610** `git push fork main` erfolgreich, kein Push nach `origin`.

## Betroffene Pfade

Keine Quelldatei. Build, Install, `.beans/bt-xy2i--*.md`.

## Prelude 2026-07-24 (aus T1-T5-Reviews) — drei SCs sind woertlich untauglich

T3, T4 und T5 waren **jeweils erst in Runde 2** gruen. Jedes Mal fand die Mutations-Probe eine
load-bearing Zeile ohne Test, bei komplett gruener Suite. T6 ist der Abschluss-Task — hier
zaehlt zusaetzlich, dass die Pruef-Kommandos selbst messen, was sie messen sollen.

### P-1 Drei Akzeptanzkriterien dieses beans sind auf dieser Maschine nicht woertlich erfuellbar

- **SC-601 sagt `mise test`.** Das ist **kein** Gate (D19): `mise test` haengt an `test:e2e`,
  und der Playwright-Browser fehlt lokal (`browserType.launch: Executable doesn't exist`).
  Alle e2e-Specs failen in 0-1 ms als Setup-Fail, unabhaengig vom Code. **Ersatz-Gate:**
  `command go test ./...` mit EXIT=0. `mise test` **nicht** ausfuehren.

- **SC-604 nutzt `awk "{print length($0)}"`.** `/usr/bin/awk` ist hier **nicht multibyte-aware**
  (D22). Bei der Glyphen-Ausgabe (`■ ▸ ▪`) meldet es **240 statt 80** — der Wert waere
  scheinbar hart verletzt, obwohl die Ausgabe zeichengenau korrekt ist. **Ersatz:** `wc -m`
  oder Rune-Zaehlung in `command python3`. Die **Absicht** des Kriteriums (keine Zeile ueber
  80 Zeichen) gilt unveraendert, nur sein Buchstabe nicht.

- **Jeder Go-Aufruf braucht `command`-Praefix** (D21). `go` ist eine Shell-Funktion
  (dotfiles-Sync), die den Compiler verdeckt und mit **Exit 0** durchlaeuft, **ohne einen Test
  auszufuehren**. Ein Beweis ohne `command` ist wertlos.

### P-2 Vor dem Ueberschreiben von /opt/homebrew/bin/beans

Das installierte Binary ist Eriks Alltags-Werkzeug. **Vor** dem Kopieren:

- aktuelle Version festhalten (`/opt/homebrew/bin/beans version`) — erwartet `0.4.2-fork.ti53`,
- das vorhandene Binary nach `/tmp/beans-backup-<version>` sichern und den Pfad im Report nennen,
- **EARS-3 ist bindend:** zeigt der Pipe-Diff (SC-605) eine Abweichung, **nicht installieren**,
  sondern melden. Byte-Identitaet schlaegt Installation.

### P-3 Was ein Agent hier NICHT abschliessend verifizieren kann

**R01** (Glyphen `■ ▸ ▪` sind East-Asian-Ambiguous — bei doppelter Breite verschieben sich
Spalten um 1) laesst sich nur an Eriks **echtem** Terminal-Emulator abschliessend beurteilen,
nicht in tmux/pty eines Agents. Pruefe, was du kannst (tmux 80 Spalten, pty), und **melde
explizit als offenen PO-Verifikationspunkt**, was du nicht abschliessend zeigen konntest.
Keine Behauptung ueber das echte Terminal, die du nicht belegt hast.

### P-4 Weiteres

- **EARS-6 ist ernst:** `.beans/` traegt fremde uncommittete Aenderungen. Nur explizite
  Einzelpfade stagen, **nie** per Glob, **nie** `git add -A`.
- `stash@{0}` ("pnpm autogen allowBuilds") ist **nicht** Task-Arbeit — nicht poppen, nicht droppen.
- Bekannte Grenzen, kein Defekt dieses Epos: kinderlose Orphan-Epics fehlen in **beiden**
  Ausgabepfaden (bug `beans-36fa`, Ursache in `buildRoadmap`).
- **N01 aus dem T5-Review:** `fmt.Print` vs. `Println` in `RunE` ist durch keinen Test
  festgenagelt (strukturell, `RunE` braucht echten `os.Stdout`). Per pty als korrekt
  verifiziert. Nur relevant, falls hier ein Refactor ansetzt — tu das nicht.
