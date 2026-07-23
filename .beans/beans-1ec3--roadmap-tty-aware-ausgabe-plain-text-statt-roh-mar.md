---
# beans-1ec3
title: 'roadmap: TTY-aware Ausgabe (Plain-Text statt Roh-Markdown)'
status: todo
type: epic
priority: normal
tags:
    - to-review
created_at: 2026-07-23T20:26:08Z
updated_at: 2026-07-23T22:32:11Z
---

`beans roadmap` ist ein Markdown-Artefakt-Generator für GitHub/Files: shields.io-Image-Badges
pro Zeile plus `[id](path)`-Links. Es gibt keinen TTY-Check — interaktiv im Terminal kommt der
Rohquelltext. Ziel dieses Epos: TTY-aware dual-mode nach dem gh/bat/glow-Idiom.

**Plan (verbindlich, local-only):** `docs/roadmap-tty-output/PLAN.md`
**Format-Spec:** `docs/roadmap-tty-output/DESIGN.md` · **Layout-Referenz:** `render-prototype.py`
Der Plan ist `ce-plan-reviewer`-grün (2 Runden) und PO-freigegeben (2026-07-23).

## Kontext für alle Kinder (DRY — nicht in den Blättern wiederholt)

**Repo:** `/Users/erik/Obsidian/tools/lean-stack/beans-src` (Fork `xRiErOS/beans`, Upstream `hmans/beans`).
Remotes: `fork` = xRiErOS, `origin` = hmans. **Push nur nach `fork`, nie nach `origin`.**

**Architektur:** `buildRoadmap()` bleibt unverändert und liefert `*roadmapData`. Neu daneben
`renderRoadmapPretty(data, width)` — Tree-Walker über dieselbe Struktur, symmetrisch zu
`renderRoadmapMarkdown`. Die Weiche sitzt in `roadmapOutput(...)`, das `RunE` aufruft.

## Entscheidungen (stabile Kennungen, im Plan ausführlich)

- **D01** Fix im beans-src CLI-Command, nicht in beans-tui.
- **D02** TTY-aware dual-mode: stdout ist TTY → gerendert; Pipe/Redirect → Markdown wie bisher.
- **D04** stdlib-Plain-Rendering, **kein neuer Dependency** (kein glamour, kein lipgloss, kein tabwriter).
- **D07** Titel werden **nie** abgeschnitten — Wrap mit Hanging-Indent, Attribute nur auf Zeile 1.
- **D08** Breite `W = clamp(terminalCols, 80, 110)`.
- **D10** Priority `normal` ausgeblendet; Milestone/Epic ohne Priority-Zelle.
- **D12** Loses Leaf ohne Epic direkt unter Milestone, kein Miscellaneous-Bucket.
- **D13** (PO 2026-07-23) Basis ist `fork/main` **nach** Merge von `fix/beans-ti53-roadmap-nested-hierarchy`.
  Renderer deckt **4 Ebenen** ab. Layout-Variante β: `titleCol = 17` (nicht 15), Leafs unter
  Feature echt eingerückt. Ersetzt D11 (war: Epics-only-Äste).
- **D14** (PO 2026-07-23) **Der Fork ist das Produkt, nicht der PR.** Definition-of-Done ist das
  installierte Binary in `/opt/homebrew/bin/beans`, nicht "Tests grün". PR #207 upstream bleibt
  offen liegen und ist kein Gate.
- **D15** Feature-Ast-Zeilen zeigen Priority (Milestone/Epic nicht).
- **D16** `utf8.RuneCountInString` für alle Breitenrechnungen (stdlib, D04-konform).
- **D17** Typ-Wort nie abschneiden; Präfix >= 17 → genau ein Leerzeichen vor Titel.
- **D18** `No Milestone` als nackte Zeile an Spalte 0, Leerzeile davor.

## Layout-Konstanten (Variante β)

`titleCol = 17` · `prioW = 8` · `statusW = 11` · `idW = 4` · `rightW = 27` · `titleW = W - 46`

## Globale Constraints

- **Der Markdown-Pfad muss byte-identisch bleiben.** Keine Änderung an `renderRoadmapMarkdown`,
  `renderBeanRef`, `typeBadge`, `firstParagraph`, `roadmap.tmpl`, `buildRoadmap`.
- Conventional Commits, Titel <= 50 Zeichen, `Refs: <bean-id>` im Body, **kein** `Co-Authored-By`.
- Table-driven Tests, erwartete Ausgaben als String-Literale — es gibt kein `internal/commands/testdata/`.
- **Hand-getippte Layout-Literale sind verboten.** Erwartete Render-Ausgaben immer aus dem
  Prototyp bzw. dem tatsächlichen Algorithmus erzeugen. Runde 1 des Plan-Reviews fiel genau
  hierüber (3 Blocker, Literale 2-5 Zeichen zu kurz).
- **`docs/` ist per `.git/info/exclude` von git ausgeschlossen** — Doku-Änderungen bleiben lokal,
  `git add docs/...` schlägt fehl. Das ist gewollt.

## Risiken

- **R01** Glyphen `■`/`▸`/`▪` sind East-Asian-Ambiguous — bei doppelter Breite verschieben sich
  Spalten um 1. Am echten Terminal prüfen (T6).
- **R02** `utf8.RuneCountInString` zählt CJK/Emoji als eine Zelle — bekannte, akzeptierte Grenze.
- **R04** `brew install`/`upgrade` überschreibt das Fork-Binary — dann T6 Build+Install wiederholen.

## Definition of Done

- `main` enthält ti53-Merge **und** TTY-Renderer, `fork/main` gepusht (nicht `origin`).
- `mise test` grün.
- `beans roadmap` gepiped byte-identisch zum Stand vorher.
- `beans roadmap` am Terminal: vier Ebenen, bündige Titel, keine Badges/Links, bei 80 Spalten
  kein Umbruch.
- `/opt/homebrew/bin/beans` meldet `0.4.2-fork.tty` und wirkt in `beans-tui` und `lean-stack`.


## Nachtrag 2026-07-23 (Gate-B-Verifikation, F03)

Risikoregister vervollständigt — R05 fehlte:

- **R05** Das Fork-Delta gegen `hmans/beans` wächst um einen zweiten Commit-Strang (Nesting-Fix
  plus TTY-Renderer). Ein späterer Upstream-Merge wird dadurch aufwendiger.
  **Umgang:** bewusst akzeptiert (D01/D14) — PR #207 bleibt offen liegen, der Fork ist das
  Produkt. Kein Aktionsbedarf, nur Registrierung.

## Nachtrag 2026-07-23 (PO, Vorflug-Check Realisierung) — D19 Test-Gate praezisiert

**Befund:** `mise test` ist `depends = ["codegen", "test:e2e"]` + `run = "go test ./..."`.
Der e2e-Teil ist lokal rot, aber nicht wegen Code:
`browserType.launch: Executable doesn't exist at .../ms-playwright/chromium_headless_shell-1208/...`
— das Playwright-Browser-Binary fehlt auf dieser Maschine. Alle Specs failen in 0-1 ms
(Setup-Fail, kein Assertion-Fail). `go test ./...` allein: **EXIT=0, gruen**.

**D19 (PO 2026-07-23):** Das Test-Gate dieses Epos ist **`go test ./...` gruen**, nicht
`mise test`. Der e2e-Pfad ist **explizit ausgeklammert**: dieser Epos beruehrt ausschliesslich
`internal/commands/roadmap*.go`, kein Frontend, kein GraphQL-Schema, kein Codegen-Input.
Das e2e-Rot ist preexistierend und umgebungsbedingt. Kein Playwright-Browser-Download
als Vorbedingung.

**Ersetzt** in der Definition of Done die Zeile "mise test gruen" durch:
- `go test ./...` gruen (EXIT=0).
- `beans roadmap` gepiped byte-identisch zum Stand vorher.
- Terminal-Smoke am echten TTY (T6).

Die uebrigen DoD-Punkte bleiben unveraendert.

## Nachtrag 2026-07-23 (ce-specs-reviewer T1, Finding fuer ALLE Kinder) — D21 `command go`

**Befund:** Die lokale Shell hat eine **Funktion namens `go`** (aus dem `~/.claude`/dotfiles-Sync).
Sie verdeckt den Go-Compiler. Ein blosses `go test ./...` ruft das Sync-Skript, **nicht** den
Compiler — und endet mit Exit 0, ohne dass je ein Test lief. Ein Agent, der das nicht weiss,
meldet gruene Tests, die nie stattgefunden haben.

**D21 (verbindlich fuer alle Tasks dieses Epos):** Jeder Go-Aufruf **immer** mit `command`-Praefix:

```
command go test ./...
command go test ./internal/commands/
command go build ...
```

Ein Test-Beweis ohne `command`-Praefix ist **kein Beweis** und wird vom Review zurueckgewiesen.
Belegt am T1-Review (2026-07-23); der T1-Implementer hatte es korrekt, weil SC-104 es
zufaellig vorgab.

**Zaehler-Hinweis:** Die in bean-Bodies fixierten Commit-Zahlen (`git log origin/main..main`)
veralten, sobald ein bean-Abschluss-Commit dazukommt. Jeder Task zaehlt selbst frisch, statt
sich auf eine im bean notierte Zahl zu verlassen.

## Nachtrag 2026-07-23 (ce-specs-reviewer T2) — D22 `awk` misst Bytes + PLAN-Luecke

**D22 (verbindlich fuer alle Tasks dieses Epos):** `/usr/bin/awk` auf dieser Maschine ist
**nicht multibyte-aware**, trotz UTF-8-Locale. `awk "{print length(\$0)}"` misst **Bytes**, nicht
Zeichen. Bei einer Ausgabe mit den Glyphen `■ ▸ ▪` meldet es **240 statt 80**.

Fuer jede Breitenpruefung stattdessen:

```
wc -m                      # Zeichen, nicht Bytes
command python3 -c "..."   # Rune-Zaehlung
```

Mehrere Akzeptanzkriterien dieses Epos zitieren woertlich einen `awk`-Befehl. **Der Buchstabe
dieser Kriterien ist auf dieser Maschine untauglich — die Absicht zaehlt (Zeilenbreite in
Zeichen).** Wer den awk-Wert als Beweis meldet, meldet einen Fehlbefund.

Zusammen mit **D21** (`go` ist eine Shell-Funktion, verdeckt den Compiler) ist das die zweite
Stelle, an der ein naiv abgesetztes Standard-Kommando hier still das Falsche misst. **Generelle
Regel:** Bevor ein Kommando als Beweis zitiert wird, verifizieren, dass es misst, was es messen
soll.

## Nachtrag 2026-07-23 — PLAN.md Task 2 Step 1 ist lueckenhaft (bestaetigt)

Der `ce-specs-reviewer` hat unabhaengig bestaetigt: der in `docs/roadmap-tty-output/PLAN.md`
Task 2 Step 1 (Zeilen 169-260) woertlich vorgegebene Python-Quelltext iteriert **nur** ueber
Milestones und kennt **keine** Verarbeitung von `kids[""]`. Derselbe Plan-Abschnitt zeigt in
Step 3 aber eine Zielausgabe **mit** `No Milestone`-Sektion (D18). **Der Plan ist intern
inkonsistent** — der gegebene Quelltext kann die vom selben Plan geforderte Ausgabe nicht
erzeugen. Konkrete Auswirkung an echten Daten: 277 von 278 Nicht-Milestone-beans waeren
kommentarlos aus der Ausgabe gefallen.

**Behoben** in `docs/roadmap-tty-output/render-prototype.py` (T2), inkl. Ausschluss des
Milestone-beans selbst aus `kids[""]` (sonst Doppel-Render).

**Verbindlich fuer alle Folge-Tasks:** Maßgebliche Layout-Referenz ist die **Datei**
`docs/roadmap-tty-output/render-prototype.py` und der DESIGN.md-Block "Ziel-Format
(eingefroren)" — **nicht** der Quelltext-Block in PLAN.md. Wer PLAN.md Task 2 Step 1 als
Vorlage nimmt, laeuft in dieselbe Luecke.

## Abschluss der Realisierung 2026-07-24 — Tag `to-review`

Alle sechs Task-beans `completed` und `ce-specs-reviewer`-gruen. T3, T4 und T5 jeweils erst in
**Runde 2** — jedes Mal fand die Mutations-Probe eine load-bearing Zeile ohne Test bei
komplett gruener Suite.

| bean | Task | Runden | Kern-Befund der roten Runde |
| --- | --- | --- | --- |
| `beans-l36h` | T1 ti53-Integration | 1 | — |
| `beans-g5hz` | T2 Layout-Spec beta | 1 | (Plan-Luecke gefunden, s.u.) |
| `beans-ejoz` | T3 Layout-Primitive | 2 | D17-Grenzfall + Rune-Counting ungetestet |
| `beans-h30q` | T4 Tree-Walker | 2 | `Unscheduled.Features`-Loop ungetestet |
| `beans-zb00` | T5 TTY-Weiche | 2 | `links`-Parameter nur einseitig beobachtet |
| `beans-w1dn` | T6 Binary + Alltag | 1 | — |

## Definition of Done — Nachweis

| DoD-Punkt | Beleg |
| --- | --- |
| main enthaelt ti53-Merge + TTY-Renderer | `git merge-base --is-ancestor` bestaetigt |
| fork/main gepusht, origin unberuehrt | `main == fork/main == f58cc59`; `git ls-remote origin main` = `99260bf`, unveraendert |
| Test-Gate gruen (D19: `command go test ./...`) | reviewer-eigener Lauf, alle Pakete `ok`, EXIT=0 |
| `beans roadmap` gepiped **byte-identisch** | Reviewer-eigener Diff Backup-Binary vs. installiertes, SHA-256 gleich |
| vier Ebenen am Terminal, keine Badges/Links | Reviewer-eigener Demo-Baum via tmux pty, `grep -c "img.shields.io\|]("` = 0 |
| bei 80 Spalten kein Umbruch | tmux + Rune-Zaehlung, max = 80 |
| `/opt/homebrew/bin/beans` meldet `0.4.2-fork.tty` | verifiziert; wirkt in beans-tui und lean-stack (EXIT=0) |
| `bt-xy2i` scrapped | `status: scrapped` + Reasons-Sektion, versioniert in `3bbf1fa` |

## Was der PO abnehmen muss (nicht agentisch belegbar)

- **R01 — der einzige strukturell offene Punkt des Epos.** Die Glyphen `■ ▸ ▪` sind
  East-Asian-Ambiguous. tmux und pty rendern sie einspaltig; ob Eriks **echter**
  Terminal-Emulator das ebenso tut, ist durch keinen Agenten-Test beweisbar. Bei doppelter
  Breite verschieben sich alle Spalten um 1. Bitte `beans roadmap` am echten Terminal ansehen.
- Terminalbreiten **ausserhalb** 80/110 wurden nicht am echten Terminal geprueft (nur die
  80-Spalten-Grenze und ein 100-Spalten-Lauf).

## Nach der Realisierung entstanden

- **bug `beans-36fa`** — kinderlose Orphan-**Epic** verschwindet aus **beiden** Ausgabepfaden
  (real betroffen: `beans-en7i`). Ursache in `buildRoadmap`, **aelter** als der Pretty-Pfad und
  im Markdown identisch — **keine Regression dieses Epos**. Zwilling von `beans-n8zw` (der
  Feature-Fall, bereits gefixt). Ein Fix wuerde den Markdown-Output **aendern** und damit dessen
  Byte-Identitaets-Garantie beruehren → **PO-Entscheid noetig**, bevor jemand es anfasst.
- **LESSONS-LEARNED LL-10 bis LL-19** fortgeschrieben. LL-10 ist unangenehm: der Forward-Guard
  von LL-02 (`go`-Shell-Falle, 2026-07-17) war nie verdrahtet worden, weshalb dieselbe Falle
  erneut zuschlug. Jetzt in `docs/SSTD.md` § Nicht-Ableitbarkeiten verankert — nicht in
  `CLAUDE.md`, das ist eine Upstream-Datei und vergroessert das Fork-Delta.
- **Aufgeraeumt:** stale git-worktree `beans-src-worktrees/fix-ti53` entfernt (clean, Branch
  vollstaendig in main, keine unique Commits; der Branch selbst bleibt bestehen).
