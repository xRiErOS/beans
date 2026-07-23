---
# beans-w1dn
title: T6 Binary bauen, installieren, Alltag verifizieren
status: todo
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:28:43Z
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
