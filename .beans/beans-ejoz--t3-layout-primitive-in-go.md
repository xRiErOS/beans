---
# beans-ejoz
title: T3 Layout-Primitive in Go
status: completed
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T21:13:15Z
parent: beans-1ec3
blocked_by:
    - beans-l36h
    - beans-g5hz
---

**Plan-Referenz:** `docs/roadmap-tty-output/PLAN.md` → Task 3. Vollständiger Go-Quelltext und
alle Testfälle stehen dort — von dort übernehmen.

## Objective (User Story)

Als Tree-Walker (Task 4) brauche ich vier getestete Zeilen-Primitive, die je eine einzelne
Bean-Zeile korrekt setzen — damit der Walker sich nur um Baum-Traversierung kümmern muss und
Layout-Fehler an einer Stelle isoliert testbar sind.

## Hintergrund

TDD in vier Schritten: erst `roadmapShortID` + `roadmapRightBlock` (Step 1-4), dann
`roadmapWrapTitle` + `roadmapLine` (Step 5-8). Die Primitive kennen die Baumstruktur nicht.

**Import-Reihenfolge beachten** (Go lehnt ungenutzte Imports ab):
- Step 1 Testdatei: nur `"testing"` + `pkg/bean`.
- Step 3 `roadmap_pretty.go`: `"fmt"` + `"strings"` + `pkg/bean` — **noch kein** `"unicode/utf8"`.
- Step 5 Testdatei erweitern auf: `"strings"`, `"testing"`, `"unicode/utf8"`, `pkg/bean`.
- Step 7 `roadmap_pretty.go`: `"unicode/utf8"` ergänzen.

## EARS-Anforderungen

- **EARS-1** THE `roadmapShortID` SHALL bei einer ID mit Bindestrich das Segment nach dem letzten
  Bindestrich liefern und bei einer ID ohne Bindestrich die ID unverändert.
- **EARS-2** THE `roadmapRightBlock` SHALL für jede Eingabe exakt 27 Zeichen liefern.
- **EARS-3** WHEN `showPrio` false ist ODER die Priority `normal` lautet, THEN THE
  `roadmapRightBlock` SHALL die Priority-Zelle leer lassen (D10).
- **EARS-4** THE `roadmapWrapTitle` SHALL Wörter auf Wortgrenzen umbrechen; IF ein Wort breiter
  als die Zeilenbreite ist, THEN THE Funktion SHALL es hart brechen.
- **EARS-5** THE `roadmapWrapTitle` SHALL bei leerem Titel genau eine leere Zeile liefern
  (niemals ein leeres Slice).
- **EARS-6** THE `roadmapLine` SHALL den Titel an Spalte 17 beginnen lassen; IF das Präfix
  17 Zeichen oder länger ist, THEN THE Funktion SHALL genau ein Leerzeichen einfügen (D17).
- **EARS-7** THE `roadmapLine` SHALL Attribute nur auf der ersten Zeile ausgeben;
  Fortsetzungszeilen SHALL 17 Leerzeichen Hanging-Indent tragen und keine Attribute (D07).
- **EARS-8** THE Breitenrechnung SHALL `utf8.RuneCountInString` verwenden, nicht `len()` (D16).

## Akzeptanzkriterien

- [x] **SC-301** `TestRoadmapShortID` grün (4 Fälle: prefixed, multi-hyphen, bare, empty).
- [x] **SC-302** `TestRoadmapRightBlock` grün (3 Fälle), jeder Fall exakt 27 Zeichen.
- [x] **SC-303** `TestRoadmapWrapTitle` grün (5 Fälle inkl. Hard-Break und Umlaute).
- [x] **SC-304** `TestRoadmapLine` grün — Ergebnis exakt 80 Zeichen, Titel bei Rune-Index 17.
- [x] **SC-305** `TestRoadmapLineWrapsWithHangingIndent` grün — genau 3 Zeilen, Attribute nur
      auf Zeile 1, Fortsetzungen ohne `uswm`.
- [x] **SC-306** `TestRoadmapLineOverlongPrefix` grün — genau ein Leerzeichen nach dem Präfix.
- [x] **SC-307** Jeder Fehlschlag-Step wurde vor der Implementierung ausgeführt und zeigte den
      im Plan angegebenen `undefined:`-Fehler (RED vor GREEN).
- [x] **SC-308** Commit `feat(roadmap): layout primitives for tty output` mit `Refs: <bean-id>`.

## Betroffene Pfade

- Neu: `internal/commands/roadmap_pretty.go`
- Neu: `internal/commands/roadmap_pretty_test.go`

## Produziert (für Task 4)

```go
const (
    roadmapTitleCol = 17
    roadmapPrioW    = 8
    roadmapStatusW  = 11
    roadmapIDW      = 4
    roadmapRightW   = 27
    roadmapMinWidth = 80
    roadmapMaxWidth = 110
)
func roadmapShortID(id string) string
func roadmapRightBlock(b *bean.Bean, showPrio bool) string
func roadmapWrapTitle(title string, width int) []string
func roadmapLine(prefix string, b *bean.Bean, showPrio bool, width int) string
```

`roadmapLine` liefert mehrzeilig, mit `
` verbunden, **ohne** Trailing-Newline.

## Prelude 2026-07-23 (aus T1- und T2-Review, vor der eigentlichen Task-Arbeit lesen)

Non-blocking Findings der `ce-specs-reviewer`-Laeufe zu T1 (`beans-l36h`) und T2 (`beans-g5hz`),
beide APPROVED, keine Blocker.

- **P-1 Umgebungsfallen (D21/D22 im Epic-bean `beans-1ec3`).** Zwei Standard-Kommandos liefern
  hier still Falsches:
  - `go` ist eine **Shell-Funktion** (dotfiles-Sync), die den Compiler verdeckt und mit Exit 0
    durchlaeuft, **ohne einen Test auszufuehren** → immer `command go test ./...`.
  - `awk` misst **Bytes statt Zeichen** → fuer Breitenpruefungen `wc -m` oder Rune-Zaehlung.
  Ein Beweis aus einem dieser Kommandos ohne Gegenprobe wird vom Review zurueckgewiesen.

- **P-2 PLAN.md ist NICHT die Layout-Referenz.** Der Quelltext-Block in `PLAN.md` Task 2 Step 1
  ist nachweislich lueckenhaft (keine `No Milestone`-Verarbeitung; 277 von 278 beans fielen
  raus). Maßgeblich sind allein:
  - die Datei `docs/roadmap-tty-output/render-prototype.py` (ausfuehren, Ausgabe uebernehmen),
  - der DESIGN.md-Abschnitt **"## Ziel-Format (eingefroren)"** — wird von T4 woertlich als
    `want`-Literal uebernommen.

- **P-3 Kein Demo-Datensatz im Repo.** Der `.beans`-Demo-Datensatz, aus dem das eingefrorene
  Ziel-Format erzeugt wurde, existiert nicht mehr. Wer die Referenz regenerieren will, baut ihn
  aus den in "Notes for T3" von `beans-g5hz` gelisteten IDs/Titeln neu auf — **ausserhalb des
  Repos** (z. B. `/tmp`) und mit `--beans-path`, niemals im Projekt-`.beans/`.

- **P-4 Redaktionell:** die T2-Deviation nennt "D12/D18" als Quelle der No-Milestone-Sektion;
  tragend ist allein **D18**. Keine Handlungsrelevanz.

## Summary

Vier Layout-Primitive in `internal/commands/roadmap_pretty.go` implementiert: `roadmapShortID`
(ID-Suffix nach letztem Bindestrich), `roadmapRightBlock` (27-Zeichen-Attributblock,
Priority/Status/ID, D10-Ausblendung von `normal`), `roadmapWrapTitle` (Wortgrenzen-Wrap mit
Hard-Break für überlange Wörter, Rune-basiert nach D16), `roadmapLine` (eine Bean-Zeile,
Titel-Start Spalte 17, D17-Overflow-Regel, D07-Hanging-Indent ohne Attribute auf
Fortsetzungszeilen). TDD strikt in den zwei Plan-Schritten (Step 1-4, dann 5-8) durchgeführt,
RED vor jeder Implementierung belegt. Alle 6 Testfunktionen (17 Subtests) grün, `command go
test ./...` grün, `command go build ./...` grün. Markdown-Pfad unangetastet (nur zwei neue
Dateien, `git status --short internal/commands/` zeigt ausschließlich die beiden `??`-Einträge).

## Test-Output

Alle Kommandos mit `command`-Präfix (D21). Zusätzlich wurden alle Layout-Literale vor dem
Schreiben in den Test unabhängig per `command python3` aus der jeweiligen Formel nachgerechnet
(nicht nur aus PLAN.md abgeschrieben) — siehe „Herkunft der Literale" im Abschluss-Report an den
Supervisor.

**RED — Step 2 (`roadmapShortID`/`roadmapRightBlock` noch nicht implementiert):**

```
$ command go test ./internal/commands/ -run 'TestRoadmapShortID|TestRoadmapRightBlock'
# github.com/hmans/beans/internal/commands [github.com/hmans/beans/internal/commands.test]
internal/commands/roadmap_pretty_test.go:22:14: undefined: roadmapShortID
internal/commands/roadmap_pretty_test.go:57:11: undefined: roadmapRightBlock
internal/commands/roadmap_pretty_test.go:61:19: undefined: roadmapRightW
internal/commands/roadmap_pretty_test.go:62:67: undefined: roadmapRightW
FAIL	github.com/hmans/beans/internal/commands [build failed]
FAIL
```

**GREEN — Step 4:**

```
$ command go test ./internal/commands/ -run 'TestRoadmapShortID|TestRoadmapRightBlock' -v
--- PASS: TestRoadmapShortID (0.00s)  (4 Subtests PASS)
--- PASS: TestRoadmapRightBlock (0.00s)  (3 Subtests PASS)
PASS
ok  	github.com/hmans/beans/internal/commands	0.522s
```

**RED — Step 6 (`roadmapWrapTitle`/`roadmapLine` noch nicht implementiert):**

```
$ command go test ./internal/commands/ -run 'TestRoadmapWrapTitle|TestRoadmapLine'
# github.com/hmans/beans/internal/commands [github.com/hmans/beans/internal/commands.test]
internal/commands/roadmap_pretty_test.go:99:11: undefined: roadmapWrapTitle
internal/commands/roadmap_pretty_test.go:117:9: undefined: roadmapLine
internal/commands/roadmap_pretty_test.go:143:9: undefined: roadmapLine
internal/commands/roadmap_pretty_test.go:167:9: undefined: roadmapLine
FAIL	github.com/hmans/beans/internal/commands [build failed]
FAIL
```

**GREEN — Step 8 (alle sechs Testfunktionen, 17 Subtests):**

```
$ command go test ./internal/commands/ -run 'TestRoadmap' -v
--- PASS: TestRoadmapShortID (4 Subtests)
--- PASS: TestRoadmapRightBlock (3 Subtests)
--- PASS: TestRoadmapWrapTitle (5 Subtests)
--- PASS: TestRoadmapLine
--- PASS: TestRoadmapLineWrapsWithHangingIndent
--- PASS: TestRoadmapLineOverlongPrefix
PASS
ok  	github.com/hmans/beans/internal/commands	0.624s
```

**Gate — voller Build + volle Testsuite (D19, `go test ./...`, nicht `mise test`):**

```
$ command go build ./... && echo BUILD_OK
BUILD_OK
$ command go test ./...
ok  	github.com/hmans/beans/internal/agent	(cached)
ok  	github.com/hmans/beans/internal/commands	0.492s
ok  	github.com/hmans/beans/internal/cors	(cached)
ok  	github.com/hmans/beans/internal/gitutil	(cached)
ok  	github.com/hmans/beans/internal/graph	(cached)
ok  	github.com/hmans/beans/internal/portalloc	(cached)
ok  	github.com/hmans/beans/internal/search	(cached)
ok  	github.com/hmans/beans/internal/terminal	(cached)
ok  	github.com/hmans/beans/internal/tui	(cached)
ok  	github.com/hmans/beans/internal/ui	(cached)
ok  	github.com/hmans/beans/internal/web	(cached)
ok  	github.com/hmans/beans/internal/worktree	(cached)
ok  	github.com/hmans/beans/pkg/bean	(cached)
ok  	github.com/hmans/beans/pkg/beancore	(cached)
ok  	github.com/hmans/beans/pkg/config	(cached)
ok  	github.com/hmans/beans/pkg/forge	(cached)
ok  	github.com/hmans/beans/pkg/safepath	(cached)
EXIT=0
```

**Markdown-Pfad unangetastet:**

```
$ git status --short internal/commands/
?? internal/commands/roadmap_pretty.go
?? internal/commands/roadmap_pretty_test.go
$ git diff --stat -- internal/commands/roadmap.go internal/commands/roadmap.tmpl
(leer)
```

## Deviations/ERRATA

Keine. `PLAN.md` Task 3 war (anders als Task 2 Step 1, siehe P-2) vollständig und intern
konsistent — Go-Quelltext und Testfälle wurden wörtlich übernommen (die Task-bean weist das
explizit an: „von dort übernehmen"). Trotzdem wurde jedes literale `want`-Zeichenkettenliteral
vor Verwendung unabhängig per `command python3` aus der jeweiligen Formel nachgerechnet statt
blind abgeschrieben — siehe Herkunft-der-Literale-Abschnitt im Supervisor-Report. Keine
Abweichung von Signatur, Konstanten oder Testfällen des Produzieren-Blocks.

## Notes for T4

- Alle vier Primitive sind fertig und getestet: `roadmapShortID`, `roadmapRightBlock`,
  `roadmapWrapTitle`, `roadmapLine` in `internal/commands/roadmap_pretty.go`. Signaturen exakt
  wie im „Produziert"-Block oben — Task 4 kann sie direkt importieren, keine Anpassung nötig.
- `roadmapLine` kennt keine Baumstruktur — der Tree-Walker (Task 4) muss selbst entscheiden,
  welcher Präfix (`■ Milestone`, `▸ Epic`, `▪ Feature`, `- <typ>`) und welches `indent`/`showPrio`
  pro Zeilentyp gilt. Die Präfix-Tabelle in DESIGN.md „### Zeilen-Präfixe" ist dafür die
  verbindliche Quelle, nicht neu herzuleiten.
- `roadmapLine` gibt **keinen** Trailing-Newline zurück — der Walker muss beim Aneinanderreihen
  mehrerer `roadmapLine`-Aufrufe selbst `\n` zwischen den Zeilen einfügen (nicht am Ende).
- Maßgebliche Ziel-Ausgabe für `TestRenderRoadmapPrettyAt80` bleibt der DESIGN.md-Block
  „## Ziel-Format (eingefroren)" bzw. die tatsächliche `render-prototype.py`-Ausgabe — nicht der
  Python-Quelltext in `PLAN.md` Task 2 (P-2). Für Task 3 galt das nicht, weil hier die Go-Primitive
  selbst und nicht der Tree-Walker/das No-Milestone-Rendering betroffen sind.
- `roadmapMinWidth`/`roadmapMaxWidth` (80/110) sind als Konstanten vorhanden, aber von T3 noch
  nirgends verwendet (kein `clamp`) — das ist Aufgabe des Tree-Walkers (`W = clamp(terminalCols,
  80, 110)`, DESIGN.md „Breite W").
- Import-Fallen aus der Aufgabenbeschreibung sind jetzt irrelevant für Folge-Tasks: beide Dateien
  haben ihren finalen Import-Satz (`fmt`+`strings`+`unicode/utf8`+`pkg/bean` bzw.
  `strings`+`testing`+`unicode/utf8`+`pkg/bean`).
