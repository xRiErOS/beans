---
# beans-w1dn
title: T6 Binary bauen, installieren, Alltag verifizieren
status: completed
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T22:25:00Z
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

- [x] **SC-601** (Ersatz-Gate D19) `command go test ./... -count=1` — EXIT=0, kein FAIL. `mise
      test` nicht ausgeführt (P-1, hängt an fehlendem Playwright-Browser).
- [x] **SC-602** `/tmp/beans-fork version` → `beans 0.4.2-fork.tty (9e11e67) built
      2026-07-23T22:17:29Z`.
- [x] **SC-603** `/tmp/beans-fork roadmap` am echten pty (tmux, keine Stdout-Umleitung) zeigt
      `Roadmap`-Kopf, `═`-Linie, Glyphen `■ ▸ ▪ -`, keine `img.shields.io`, keine `](`. Da die
      echten Repo-Daten aktuell kein offenes Milestone/Feature-Container haben (verifiziert),
      zusätzlich per Demo-`.beans/`-Baum außerhalb des Repos (`/tmp/beans-demo-roadmap`,
      danach gelöscht) end-to-end bestätigt.
- [x] **SC-604** (Ersatz-Messung D22) 80-Spalten-tmux-Smoke via `tmux capture-pane` + Python-
      Rune-Zählung/`wc -m`: max. Zeilenbreite = 80, keine Zeile > 80. Der wörtliche
      `awk '{print length($0)}'`-Befehl liefert wie im Prelude vorhergesagt `240` (Byte- statt
      Zeichenzählung, nicht multibyte-aware) — als Falsifikations-Beleg mitgeführt, nicht als
      Nachweis verwendet.
- [x] **SC-605** `diff /tmp/roadmap-old.md /tmp/roadmap-new.md` → `IDENTISCH`, SHA-256 beider
      Dateien identisch (`96e05802…`).
- [x] **SC-606** `which beans` → `/opt/homebrew/bin/beans`; `beans version` →
      `beans 0.4.2-fork.tty (9e11e67) built 2026-07-23T22:17:29Z`.
- [x] **SC-607** `beans roadmap` in `beans-tui-repository` (EXIT=0, leerer Body — alle 149 dortigen
      beans completed/scrapped, keine Regression) und in `lean-stack` (EXIT=0, volle Tabelle mit
      `■ ▸ -`) ohne Fehler.
- [x] **SC-608** `bt-xy2i` bereits `scrapped` mit `## Reasons for Scrapping` → Verweis auf
      `beans-f1t4` (aus einem vorgelagerten Vorflug-Check, vor Start dieses Tasks erledigt).
- [x] **SC-609** Erfüllt via bereits vorhandenem Commit `3bbf1fa` ("chore(beans): D19 test-gate +
      bt-xy2i scrapped", explizite Einzelpfade, kein Glob) — Message weicht vom hier zitierten
      Literal ab, siehe Deviations.
- [x] **SC-610** `git push fork main` → `67ea3a5..9e11e67 main -> main` nach `xRiErOS/beans`.
      `origin/main` vor und nach `git fetch origin` unverändert bei `99260bf`.

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

## Summary

Fork-Binary `0.4.2-fork.tty` (SHA `9e11e67`) aus `./cmd/beans` gebaut, gegen das vorherige
installierte Binary (`0.4.2-fork.ti53`) im Pipe-Pfad byte-identisch verifiziert (EARS-2/3), und
erst danach nach `/opt/homebrew/bin/beans` installiert. TTY-Rendering (vier Ebenen, alle Glyphen,
80-Spalten-Grenze) am echten pty (tmux) bestätigt — mit einer Einschränkung: die echten
`beans-src`-Repo-Daten enthalten aktuell kein offenes Milestone und keinen Feature-Container mit
offenen Kindern, daher zusätzlich per Wegwerf-Demo-Baum außerhalb des Repos verifiziert.
Cross-Repo-Wirksamkeit in `beans-tui-repository` und `lean-stack` bestätigt (EARS-5). `bt-xy2i`
war bereits vor Task-Start `scrapped` (Vorflug-Check). Fork gepusht, `origin/main` nachweislich
unverändert (EARS-7). Kein Produktionscode geändert.

## Test-Output

Ersatz-Gate (D19, Prelude P-1) statt `mise test`:

```
$ bash -c 'set -o pipefail; command go test ./... -count=1 > /tmp/go-test-t6.log 2>&1; echo "REAL_EXIT=$?"'
REAL_EXIT=0
```

`/tmp/go-test-t6.log` (Auszug, alle Pakete `ok` oder `[no test files]`):

```
ok  	github.com/hmans/beans/internal/agent	0.566s
ok  	github.com/hmans/beans/internal/commands	0.575s
ok  	github.com/hmans/beans/internal/cors	1.508s
ok  	github.com/hmans/beans/internal/gitutil	10.494s
ok  	github.com/hmans/beans/internal/graph	1.917s
ok  	github.com/hmans/beans/internal/portalloc	2.831s
ok  	github.com/hmans/beans/internal/search	1.879s
ok  	github.com/hmans/beans/internal/terminal	4.550s
ok  	github.com/hmans/beans/internal/tui	2.637s
ok  	github.com/hmans/beans/internal/ui	2.534s
ok  	github.com/hmans/beans/internal/web	1.958s
ok  	github.com/hmans/beans/internal/worktree	7.202s
ok  	github.com/hmans/beans/pkg/bean	2.166s
ok  	github.com/hmans/beans/pkg/beancore	4.231s
ok  	github.com/hmans/beans/pkg/config	1.935s
ok  	github.com/hmans/beans/pkg/forge	1.926s
ok  	github.com/hmans/beans/pkg/safepath	1.874s
```

Kein `FAIL` in der vollen Ausgabe. `mise test` (mit `test:e2e`) **nicht** ausgeführt (Prelude
P-1: e2e-Setup-Fail unabhängig vom Code, kein Gate laut D19).

Build:

```
$ command go build -ldflags "-X .../internal/version.Version=0.4.2-fork.tty \
    -X .../internal/version.Commit=9e11e67 \
    -X .../internal/version.Date=2026-07-23T22:17:29Z" -o /tmp/beans-fork ./cmd/beans
BUILD_EXIT=0
$ /tmp/beans-fork version
beans 0.4.2-fork.tty (9e11e67) built 2026-07-23T22:17:29Z
```

## Byte-Identitaets-Nachweis

```
$ which beans && beans version   # vor Install
/opt/homebrew/bin/beans
beans 0.4.2-fork.ti53 (3419e8a) built 2026-07-21T18:37:01Z

$ beans roadmap > /tmp/roadmap-old.md          # altes installiertes Binary
$ /tmp/beans-fork roadmap > /tmp/roadmap-new.md
$ diff /tmp/roadmap-old.md /tmp/roadmap-new.md && echo "IDENTISCH"
IDENTISCH

$ shasum -a 256 /tmp/roadmap-old.md /tmp/roadmap-new.md
96e058021d1817526053fc8aef69e8806459f57bb36a63ce0fe867ecea59455b  /tmp/roadmap-old.md
96e058021d1817526053fc8aef69e8806459f57bb36a63ce0fe867ecea59455b  /tmp/roadmap-new.md
```

Beide Dateien 88 Zeilen, identische Prüfsumme. EARS-3-Gate erfüllt → Installation freigegeben.

## Backup / Installation

```
$ cp /opt/homebrew/bin/beans /tmp/beans-backup-0.4.2-fork.ti53
$ /tmp/beans-backup-0.4.2-fork.ti53 version
beans 0.4.2-fork.ti53 (3419e8a) built 2026-07-21T18:37:01Z

$ cp /tmp/beans-fork /opt/homebrew/bin/beans && chmod +x /opt/homebrew/bin/beans
$ which beans
/opt/homebrew/bin/beans
$ beans version
beans 0.4.2-fork.tty (9e11e67) built 2026-07-23T22:17:29Z
```

Backup-Pfad: `/tmp/beans-backup-0.4.2-fork.ti53` (ausführbar, geprüft).

## Smoke

**80-Spalten-Grenze (tmux pty, `beans-src`-Repo-Daten, kein Milestone/Feature-Container offen):**

```
$ tmux new-session -d -s roadmapsmoke80 -x 80 -y 400 "/tmp/beans-fork roadmap; sleep 3"
$ tmux capture-pane -t roadmapsmoke80 -p -S -400 > /tmp/roadmap80-raw.txt
$ command python3 -c "print(max(len(l) for l in [...] if l.strip()))"
80
$ awk '{print length($0)}' /tmp/roadmap80-raw.txt | sort -rn | head -3   # zum Vergleich, D22
240
240
82
```
(Die `82`-Zeile aus `awk` ist die Leerzeile-Kopfzeile mit Steuerzeichen-Artefakt aus der
Byte-Zählung, nicht Zeichenbreite — Bestätigung, dass `awk` hier unbrauchbar ist.)

**Glyphen-Vollcheck (Demo-`.beans/`-Baum außerhalb des Repos, `/tmp/beans-demo-roadmap`, danach
gelöscht):**

```
Roadmap
════════════════════════════════════════════════════════════════════════════════════════════════════

■ Milestone      Demo Milestone                                                    todo         omy2
  ▸ Epic         Demo Epic                                                         todo         jhc9
    - task       Demo leaf under epic                                              todo         60f4
  ▪ Feature      Demo Feature Branch                                         high  todo         7qhh
    - task       Demo leaf under feature                                           todo         2sbm
  - task         Loose leaf under milestone                                        todo         t1i7
```

`grep -c "img.shields.io"` und `grep -c "]("` → beide `0`.

**Cross-Repo (EARS-5):**

```
$ cd /Users/erik/Obsidian/tools/beans-tui/beans-tui-repository && beans roadmap
# Roadmap
                                          # (leer — alle 149 dortigen beans completed/scrapped, EXIT=0, kein Fehler)

$ cd /Users/erik/Obsidian/tools/lean-stack && beans roadmap   # via tmux pty
Roadmap
════════════════════════════════════════════════════════════════════════════════════════════════════

■ Milestone      Claude-Code-Config-Umstellung (TPIC-Rollen)                       completed    jsz1
  ▸ Epic         Phase A — Rollen-Set-Umbau (cc/profiles/boot-prompts)             completed    84wv
    - feature    Command: cc-Realization/Implementer-Session starten         high  todo         nhty
                 (herdr)
...
EXITCODE=0
```

## Offene PO-Verifikationspunkte

- **R01 (East-Asian-Ambiguous-Glyphen `■ ▸ ▪`):** Nur an Eriks echtem Terminal-Emulator
  abschließend beurteilbar. tmux/pty (dieser Task) rendert die Glyphen einspaltig — ob Eriks
  konkreter Terminal-Emulator (iTerm2/Terminal.app/Ghostty/etc.) sie ebenfalls einspaltig
  rendert oder doppelt breit (dann verschieben sich Spalten um 1), ist **nicht** durch diesen
  Task belegt. Bitte am echten Terminal mit `beans roadmap` gegenprüfen.
- **Terminalbreiten außerhalb 80/110** (< 80 oder > 110 Spalten, `roadmapClampWidth`) wurden
  in diesem Task nicht am echten Terminal geprüft, nur die 80-Spalten-Grenze und ein 100-Spalten-
  Demo-Lauf.

## Deviations/ERRATA

1. **SC-601:** `mise test` nicht ausgeführt, stattdessen `command go test ./... -count=1`
   (Prelude P-1/D19) — Playwright-Browser fehlt lokal, e2e ist kein Gate für dieses Epos.
2. **SC-604:** `awk '{print length($0)}'` nicht als Beweis verwendet (liefert `240`, Byte- statt
   Zeichenzählung, D22) — stattdessen `wc -m`/Python-Rune-Zählung über `tmux capture-pane`
   (max. 80). Zusätzlich das wörtliche `awk`-Ergebnis als Falsifikations-Beleg dokumentiert.
3. **SC-603:** wörtlicher Nachweis am realen `beans-src`-Repo unvollständig, weil die
   Repo-Daten aktuell kein offenes Milestone/keinen offenen Feature-Container enthalten (0/0,
   verifiziert per `beans list --json`) — kein Code-Defekt, reiner Datenzustand. Ergänzt um
   Demo-`.beans/`-Baum außerhalb des Repos (angelegt und wieder gelöscht, keine Test-beans im
   Projekt-`.beans/`).
4. **SC-609:** Der wörtlich zitierte Commit `chore(beans): bt-xy2i scrapped (dup of
   beans-f1t4)` existiert nicht als eigenständiger Commit — `bt-xy2i` wurde bereits vor
   Beginn dieses Tasks in einem vorgelagerten Vorflug-Check gemeinsam mit dem D19-Nachtrag
   committet (`3bbf1fa`, explizite Einzelpfade, kein Glob, Body referenziert `beans-f1t4`).
   Intent erfüllt, Commit-Message weicht ab. Kein redundanter Leer-Commit erzeugt.
5. **Abschluss-Status:** Task-Prompt instruiert explizit `beans-w1dn` auf `completed` (nicht
   `to-review`) zu setzen, wenn alle SC erfüllt sind — abweichend vom generischen
   Leaf-Autonomie-Standard. Als explizite Task-spezifische Anweisung befolgt; das Epic
   `beans-1ec3` bleibt unangetastet (PO-Sache).
