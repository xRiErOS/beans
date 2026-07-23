---
# beans-ejoz
title: T3 Layout-Primitive in Go
status: in-progress
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T21:09:15Z
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

- [ ] **SC-301** `TestRoadmapShortID` grün (4 Fälle: prefixed, multi-hyphen, bare, empty).
- [ ] **SC-302** `TestRoadmapRightBlock` grün (3 Fälle), jeder Fall exakt 27 Zeichen.
- [ ] **SC-303** `TestRoadmapWrapTitle` grün (5 Fälle inkl. Hard-Break und Umlaute).
- [ ] **SC-304** `TestRoadmapLine` grün — Ergebnis exakt 80 Zeichen, Titel bei Rune-Index 17.
- [ ] **SC-305** `TestRoadmapLineWrapsWithHangingIndent` grün — genau 3 Zeilen, Attribute nur
      auf Zeile 1, Fortsetzungen ohne `uswm`.
- [ ] **SC-306** `TestRoadmapLineOverlongPrefix` grün — genau ein Leerzeichen nach dem Präfix.
- [ ] **SC-307** Jeder Fehlschlag-Step wurde vor der Implementierung ausgeführt und zeigte den
      im Plan angegebenen `undefined:`-Fehler (RED vor GREEN).
- [ ] **SC-308** Commit `feat(roadmap): layout primitives for tty output` mit `Refs: <bean-id>`.

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
