---
# beans-h30q
title: T4 Tree-Walker renderRoadmapPretty
status: completed
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T21:51:24Z
parent: beans-1ec3
blocked_by:
    - beans-ejoz
---

**Plan-Referenz:** `docs/roadmap-tty-output/PLAN.md` → Task 4. Vollständiger Go-Quelltext, die
Fixture `prettyFixture()` und das `want`-Literal stehen dort.

## Objective (User Story)

Als Nutzer von `beans roadmap` im Terminal will ich meine Roadmap als lesbare Plain-Text-Tabelle
mit allen vier Ebenen sehen — damit ich Relevanz beurteilen kann, ohne Markdown-Rohquelltext zu
entziffern.

## Hintergrund

`renderRoadmapPretty` ist das Gegenstück zu `renderRoadmapMarkdown`: gleiche Daten, andere
Oberfläche. Es sortiert **nichts** selbst, sondern folgt exakt der Slice-Reihenfolge aus
`buildRoadmap` und rendert innerhalb einer Gruppe erst `.Items`, dann `.Features` — identisch zu
`roadmap.tmpl`, damit beide Pfade dieselbe Abfolge zeigen.

**Achtung Layout-Literale:** Das `want`-Literal in `TestRenderRoadmapPrettyAt80` ist rechnerisch
erzeugt (nicht getippt) und zeichengleich mit dem DESIGN.md-Beispielblock aus T2. Weicht der
tatsächliche Output ab, prüfe zuerst gegen `python3 docs/roadmap-tty-output/render-prototype.py 80`,
welcher recht hat — der Prototyp ist die Referenz. Passe dann **den Test** an, nicht die
Layout-Konstanten.

## EARS-Anforderungen

- **EARS-1** THE `roadmapClampWidth` SHALL Werte unter 80 auf 80 und über 110 auf 110 begrenzen;
  WHEN der Wert 0 ist (kein Terminal), THEN THE Funktion SHALL 80 liefern (D08).
- **EARS-2** THE `renderRoadmapPretty` SHALL mit der Zeile `Roadmap` beginnen, gefolgt von einer
  Linie aus W `═`-Zeichen.
- **EARS-3** THE Renderer SHALL Milestones mit `■`, Epics mit `▸`, Feature-Äste mit `▪` und Leafs
  mit `-` präfixen.
- **EARS-4** THE Renderer SHALL Feature-Ast-Zeilen mit Priority rendern, Milestone- und
  Epic-Zeilen ohne (D15/D10).
- **EARS-5** THE Renderer SHALL lose Leafs direkt unter dem Milestone rendern, ohne
  Miscellaneous-Bucket (D12).
- **EARS-6** WHEN `data.Unscheduled` nicht nil ist, THEN THE Renderer SHALL die Zeile
  `No Milestone` an Spalte 0 mit Leerzeile davor und danach ausgeben (D18).
- **EARS-7** THE Ausgabe SHALL bei jeder Breite W keine Zeile enthalten, die länger als W Runen ist.
- **EARS-8** THE Ausgabe SHALL mit einem Newline enden (symmetrisch zu `renderRoadmapMarkdown`).
- **EARS-9** WHEN `data` leer ist, THEN THE Renderer SHALL nur Kopfzeile und Trennlinie ausgeben.

## Akzeptanzkriterien

- [x] **SC-401** `TestRoadmapClampWidth` grün (6 Fälle: 40, 0, 80, 96, 110, 200).
- [x] **SC-402** `TestRenderRoadmapPrettyAt80` grün — Output zeichengleich mit dem `want`-Literal.
- [x] **SC-403** `TestRenderRoadmapPrettyLineWidths` grün für 80, 96 und 110 — keine Zeile über W.
- [x] **SC-404** `TestRenderRoadmapPrettyTitleColumn` grün — jede Bean-Zeile hat an Rune-Index 17
      kein Leerzeichen.
- [x] **SC-405** `TestRenderRoadmapPrettyEmpty` grün.
- [x] **SC-406** Der Renderer enthält keine eigene Sortierung (kein `sort.` in
      `renderRoadmapPretty`) — Reihenfolge kommt aus `buildRoadmap`.
- [x] **SC-407** Fehlschlag-Steps vor Implementierung ausgeführt (RED vor GREEN).
- [x] **SC-408** Commit `feat(roadmap): pretty tree walker for tty` mit `Refs: <bean-id>`.

## Betroffene Pfade

- `internal/commands/roadmap_pretty.go` (erweitern)
- `internal/commands/roadmap_pretty_test.go` (erweitern)

## Produziert (für Task 5)

```go
func renderRoadmapPretty(data *roadmapData, width int) string
func roadmapClampWidth(cols int) int
```

## Prelude 2026-07-23 (aus T1-T3-Reviews, vor der Task-Arbeit lesen)

T3 (`beans-ejoz`) ist APPROVED — aber erst in Runde 2. Der erste Lauf war CHANGES_REQUIRED bei
**komplett gruener Suite**. Was T4 daraus mitnimmt:

- **P-1 Umgebungsfallen (D21/D22 im Epic-bean `beans-1ec3`).**
  - `go` ist eine **Shell-Funktion** (dotfiles-Sync), verdeckt den Compiler, laeuft mit Exit 0
    durch **ohne einen Test auszufuehren** → immer `command go test ./...`.
  - `awk` misst **Bytes statt Zeichen** → Breitenpruefungen mit `wc -m` oder Rune-Zaehlung.

- **P-2 PLAN.md ist NICHT die Layout-Referenz.** Der Quelltext-Block in `PLAN.md` Task 2 Step 1
  ist nachweislich lueckenhaft (keine `No Milestone`-Verarbeitung; 277 von 278 beans fielen raus).
  Maßgeblich sind allein `docs/roadmap-tty-output/render-prototype.py` (ausfuehren, Ausgabe
  uebernehmen) und der DESIGN.md-Abschnitt **"## Ziel-Format (eingefroren)"** — letzterer wird
  von dir woertlich als `want`-Literal uebernommen.

- **P-3 Kein Demo-Datensatz im Repo.** Brauchst du einen, baue ihn **ausserhalb** des Repos
  (z. B. `/tmp`) mit `--beans-path`, niemals im Projekt-`.beans/`. IDs/Titel stehen in
  `## Notes for T3` von `beans-g5hz`.

- **P-4 Ein gruener Test beweist nicht, dass die Zeile getestet ist.** T3 fiel ueber genau das:
  - Der D17-Grenzfall (`prefixW == 17`) war ungetestet — der vorhandene Fall nutzte 26 Runen,
    weit jenseits der Grenze. Mutation `>=`→`>` liess die Suite gruen.
  - Ein Umlaut-Testfall suggerierte Rune-Abdeckung, hatte aber zu grosse Margin:
    `"Pruefung"` ist 7 Runen **und** 8 Bytes, beide <= 8 — Byte- und Rune-Zaehlung lieferten
    zufaellig dasselbe. Erst `"ab é"` bei Breite 4 (Rune-Summe 4, Byte-Summe 5) trennte sie.

  **Konsequenz fuer T4:** Fuer jede load-bearing Zeile (Guards, Grenzwerte, Zaehl-Logik,
  Inclusion-Bedingungen) selbst pruefen: *Zeile brechen → failt mindestens ein Test?* Wenn nein,
  fehlt der Test. Konstruiere Testfaelle an der **Grenze**, nicht bequem daneben. Der Reviewer
  wird genau das per Mutation nachpruefen.

- **P-5 Zahlen zaehlen, nicht schaetzen.** Die T3-Summary behauptete "17 Subtests"; tatsaechlich
  8 Testfunktionen / 12 Subtests. Zahlen frisch aus dem `-v`-Output ziehen
  (`grep -c "^    --- PASS"`).


## Summary

`renderRoadmapPretty(data *roadmapData, width int) string` und `roadmapClampWidth(cols int) int`
implementiert in `internal/commands/roadmap_pretty.go`. Der Walker ist ein reiner Konsument der
T3-Primitive (`roadmapLine`, `roadmapRightBlock`, `roadmapWrapTitle`, `roadmapShortID`) — er trägt
selbst nur Präfix/Indent/showPrio-Ableitung pro Zeilentyp bei (`roadmapLeafPrefix` +
`renderRoadmapEpicGroup`/`renderRoadmapFeatureGroup` als Helfer), keine eigene Breiten- oder
Wrap-Logik. Reihenfolge: `.Items` vor `.Features` je Gruppe, `Epics` → `Features` → `Other` je
Milestone/Unscheduled — identisch zu `roadmap.tmpl`. `No Milestone` rendert unbedingt bei
`data.Unscheduled != nil` (D18/EARS-6), unabhängig von `len(Milestones)`.

## Test-Output

**RED** (vor Implementierung, Compile-Fehler da `renderRoadmapPretty`/`roadmapClampWidth` noch
nicht existierten):

```
$ command go test ./internal/commands/ -run 'TestRoadmapClampWidth|TestRenderRoadmapPretty' -v
# github.com/hmans/beans/internal/commands [github.com/hmans/beans/internal/commands.test]
internal/commands/roadmap_pretty_test.go:275:14: undefined: roadmapClampWidth
internal/commands/roadmap_pretty_test.go:309:9: undefined: renderRoadmapPretty
internal/commands/roadmap_pretty_test.go:319:10: undefined: renderRoadmapPretty
internal/commands/roadmap_pretty_test.go:334:9: undefined: renderRoadmapPretty
internal/commands/roadmap_pretty_test.go:349:9: undefined: renderRoadmapPretty
FAIL	github.com/hmans/beans/internal/commands [build failed]
FAIL
```

**GREEN** (nach Implementierung):

```
$ command go test ./internal/commands/ -run 'TestRoadmapClampWidth|TestRenderRoadmapPretty' -v
--- PASS: TestRoadmapClampWidth (0.00s)
    --- PASS: TestRoadmapClampWidth/below_floor (0.00s)
    --- PASS: TestRoadmapClampWidth/zero:_no_terminal_(D08) (0.00s)
    --- PASS: TestRoadmapClampWidth/at_floor (0.00s)
    --- PASS: TestRoadmapClampWidth/within_range (0.00s)
    --- PASS: TestRoadmapClampWidth/at_cap (0.00s)
    --- PASS: TestRoadmapClampWidth/above_cap (0.00s)
--- PASS: TestRenderRoadmapPrettyAt80 (0.00s)
--- PASS: TestRenderRoadmapPrettyLineWidths (0.00s)
--- PASS: TestRenderRoadmapPrettyTitleColumn (0.00s)
--- PASS: TestRenderRoadmapPrettyPriorityVisibility (0.00s)
--- PASS: TestRenderRoadmapPrettyEmpty (0.00s)
PASS
ok  	github.com/hmans/beans/internal/commands	0.628s
```

6 Testfunktionen, davon `TestRoadmapClampWidth` mit 6 Subtests (`-v`-Output frisch gezählt, nicht
geschätzt, P-5). `TestRenderRoadmapPrettyPriorityVisibility` ist zusätzlich zu den im bean
genannten Tests entstanden (siehe Deviations).

`command go test ./...` gesamt: EXIT=0, alle Pakete `ok`.

## Mutations-Proben

| Mutation | welcher Test failte (Name + Ausgabe) | Zeile getestet ja/nein |
|---|---|---|
| `roadmapClampWidth`: floor-Zweig `return roadmapMinWidth` → `return cols` | `TestRoadmapClampWidth/below_floor`: `roadmapClampWidth(40) = 40, want 80`; `TestRoadmapClampWidth/zero:_no_terminal_(D08)`: `roadmapClampWidth(0) = 0, want 80` | ja |
| `roadmapClampWidth`: cap-Zweig `return roadmapMaxWidth` → `return cols` | `TestRoadmapClampWidth/above_cap`: `roadmapClampWidth(200) = 200, want 110` | ja |
| D18-Guard invertiert: `if data.Unscheduled != nil` → `== nil` | `TestRenderRoadmapPrettyAt80` (fehlende `No Milestone`-Sektion im Diff) **und** `TestRenderRoadmapPrettyEmpty` (nil-Pointer-Panic auf `data.Unscheduled.Epics`, da leeres `&roadmapData{}` jetzt fälschlich den Unscheduled-Zweig betritt) | ja |
| Milestone-Zeile `showPrio` `false` → `true` | `TestRenderRoadmapPrettyPriorityVisibility`: `milestone row must not show priority (D10): "■ Milestone      M ... high  todo         aaaa"` — **`TestRenderRoadmapPrettyAt80` allein failt hier NICHT** (Milestone-Priority im DESIGN-Fixture leer, daher unsichtbar unabhängig vom Flag) — deshalb wurde `TestRenderRoadmapPrettyPriorityVisibility` ergänzt | ja (nach Ergänzung) |
| Epic-Zeile `showPrio` `false` → `true` | `TestRenderRoadmapPrettyPriorityVisibility`: `epic row must not show priority (D10): "  ▸ Epic         E ... high  todo         bbbb"` | ja |
| Feature-Zeile `showPrio` `true` → `false` | `TestRenderRoadmapPrettyAt80` (Priority-Spalte fehlt bei beiden Feature-Zeilen im Diff) **und** `TestRenderRoadmapPrettyPriorityVisibility`: `feature row must show priority (D15): "    ▪ Feature    F ...  todo         cccc"` | ja |
| `renderRoadmapEpicGroup`-Leaf-Indent `indent+2` → `indent+1` | `TestRenderRoadmapPrettyAt80` (Einrückung der Epic-Leafs im Diff um 1 verschoben) | ja |
| `renderRoadmapEpicGroup`-Reihenfolge Items/Features vertauscht | `TestRenderRoadmapPrettyAt80` (Zeilenreihenfolge weicht vom `want`-Literal ab) | ja |

Rückbau nach jeder Mutation: `diff <backup> internal/commands/roadmap_pretty.go` → leer
("REVERT OK") vor der nächsten Mutation. Nach der letzten Mutation zusätzlich
`git status --porcelain` geprüft:

```
$ git status --porcelain
 M internal/commands/roadmap_pretty.go
 M internal/commands/roadmap_pretty_test.go
```

— genau die zwei absichtlich geänderten Dateien, keine Restspuren einer Mutation.

## Herkunft der Literale

- **`want`-Literal in `TestRenderRoadmapPrettyAt80`**: wörtlich aus
  `docs/roadmap-tty-output/DESIGN.md` Abschnitt `## Ziel-Format (eingefroren)` extrahiert via
  `command python3` (Regex auf den Fence-Block), nicht abgetippt. Übereinstimmung mit dem Go-Test
  programmatisch geprüft:
  `command python3 -c "..."` → `design_block == go_want: True` (1155 Runen, identisch).
- **Layout-Korrektheit gegen die ausführbare Referenz**: Demo-`.beans`-Verzeichnis mit der exakten
  Bean-Hierarchie aus `beans-g5hz` „Notes for T3" außerhalb des Repos unter `/tmp/roadmap_demo`
  gebaut (`beans init`/`beans create ... --beans-path /tmp/roadmap_demo/.beans`, danach entfernt).
  `command python3 docs/roadmap-tty-output/render-prototype.py 80` gegen dieses Verzeichnis
  ausgeführt und mit dem DESIGN.md-Block ID-normalisiert verglichen
  (`command python3 -c "..."` → `normalized match: True`) — bestätigt, dass das eingefrorene
  DESIGN.md-Literal tatsächlich die aktuelle `render-prototype.py`-Ausgabe ist (nur die
  zufallsgenerierten IDs unterscheiden sich, was erwartet ist).
- **Fixture-IDs/Titel/Typen/Status/Priority/Parent** in `prettyFixture()`: aus `beans-g5hz`
  „Notes for T3" (ID-Liste `fexy, 9m0d, 9zpz, wa9y, 1vvd, b58r, 9bi1, lnff, 635g, h5km, xm6j, nfun`)
  und dem DESIGN.md-Block selbst (Titel/Typ/Status/Priority direkt ablesbar aus den Spalten).
- Keine anderen hand-getippten Layout-Literale — `TestRenderRoadmapPrettyPriorityVisibility` und
  `TestRenderRoadmapPrettyEmpty` prüfen strukturell (Substring/Konkatenation aus
  `roadmapMinWidth`/`roadmapMaxWidth`-Konstanten), nicht gegen abgetippte Vollzeilen.

## Deviations/ERRATA

1. **Zusätzlicher Test `TestRenderRoadmapPrettyPriorityVisibility`** (nicht in den ursprünglichen
   SC-401..408 benannt) ergänzt, weil die Mutations-Selbstprüfung zeigte: ein invertiertes
   `showPrio` auf Milestone- oder Epic-Zeilen wird von `TestRenderRoadmapPrettyAt80` NICHT
   erkannt, da die Priority der Milestone/Epic-Beans im eingefrorenen DESIGN-Fixture leer ist —
   sichtbar/unsichtbar sehen dort identisch aus. Der neue Test nutzt vier Beans mit
   `Priority: "high"` auf allen Ebenen, um EARS-4/D10/D15 pro Zeilentyp unabhängig zu pinnen.
   Kein SC verletzt (SC-401..408 alle weiterhin erfüllt), reine Zusatz-Absicherung.
2. **Der Ausschluss von Milestone-typisierten Beans aus der `No Milestone`-Sektion** (im Prelude
   als load-bearing genannt) liegt in `buildRoadmap`'s `orphanItems`-Schleife
   (`internal/commands/roadmap.go:216-219`, `if b.Type == "milestone" || ... { continue }`) —
   **unverändert, außerhalb des T4-Scopes** (harte Grenze: `buildRoadmap` darf nicht angefasst
   werden). `renderRoadmapPretty` rendert `data.Unscheduled.Other` symmetrisch zum
   Markdown-Template ungefiltert; die Milestone-Exklusion ist bereits durch bestehende
   `TestBuildRoadmap`-Coverage in `roadmap_test.go` (Zeile 544/566, "milestone.Other") abgedeckt.
   Kein neuer Test hierzu in `roadmap_pretty_test.go`, da das die falsche Datei/Funktion treffen
   würde.
3. Demo-`.beans`-Verzeichnis unter `/tmp/roadmap_demo` wurde nach der Verifikation gelöscht
   (`rm -rf /tmp/roadmap_demo /tmp/roadmap_bp.txt`) — kein Repo-Artefakt.

## Notes for T5

- `renderRoadmapPretty(data *roadmapData, width int) string` und `roadmapClampWidth(cols int) int`
  sind fertig, getestet, in `internal/commands/roadmap_pretty.go`. Signaturen exakt wie im
  „Produziert"-Block dieses beans — T5 kann sie direkt in `roadmapCmd.RunE` verdrahten.
- Die Weiche gehört laut DESIGN.md nach `--json`-Zweig: `term.IsTerminal(int(os.Stdout.Fd()))` →
  bei TTY `renderRoadmapPretty(data, roadmapClampWidth(terminalWidth))`, sonst
  `renderRoadmapMarkdown` (Ist-Pfad, unverändert). `roadmapClampWidth` selbst nimmt KEINEN
  Terminal-Query vor — der Aufrufer ermittelt `terminalCols` (z. B. via
  `golang.org/x/term.GetSize`) und übergibt 0, wenn keine Terminalgröße ermittelbar ist;
  `roadmapClampWidth(0)` liefert dann 80 (D08).
- `renderRoadmapPretty` gibt bereits einen abschließenden `
` zurück (EARS-8) — beim Ausgeben
  `fmt.Print(...)` verwenden, nicht `Println` (sonst doppelter Newline, analog zum bestehenden
  `fmt.Print(md)`-Aufruf für den Markdown-Pfad).
- Kein neuer Import in `roadmap_pretty.go` nötig (fmt, strings, unicode/utf8, pkg/bean bereits
  vorhanden) — T5 fügt die Weiche vermutlich in `roadmap.go`. `golang.org/x/term` ist bereits
  **direkte** Dependency in `go.mod` (Zeile 25, `golang.org/x/term v0.38.0`) — `term.IsTerminal`/
  `term.GetSize` sind D04-konform ohne neuen Dependency verwendbar, verifiziert via
  `grep -n "x/term" go.mod`.


## Blocker-Behebung (Runde 2, ce-specs-reviewer B01)

**Blocker:** `data.Unscheduled.Features`-Loop (`roadmap_pretty.go:189-191`) war ungetestet —
`prettyFixture()` (DESIGN.md-Fixture) setzt `Unscheduled.Features` nie, nur `Unscheduled.Epics`
und `Unscheduled.Other`. Der Reviewer ersetzte den Loop-Body durch ein No-Op: `command go test
./...` blieb komplett grün.

**Fix:** Kein Produktionscode-Fehler — die Loop-Logik war korrekt, nur ungetestet. Neuer Test
`TestRenderRoadmapPrettyUnscheduledFeature` ergänzt: eine Orphan-Feature (kein Milestone-, kein
Epic-Parent) mit einem Leaf-Kind, direkt als `roadmapData{Unscheduled: &unscheduledGroup{Features:
...}}` konstruiert (kein `prettyFixture()`-Umbau, um das eingefrorene `want`-Literal in
`TestRenderRoadmapPrettyAt80` nicht zu gefährden). Prüft: Zeilenanzahl (Fatalf bei No-Op, da die
Zeilen dann fehlen), Präfix `  ▪ Feature` mit Priority (D15), Präfix `    - task` der Leaf-Zeile
(Einrückung `indent+2`), Titel/Short-ID beider Zeilen, Breiten-Invariante (EARS-7).

**Mutations-Rot-Ausgabe** (Reviewer-Mutation exakt reproduziert: Loop-Body durch No-Op ersetzt):

```
$ command go test ./internal/commands/ -run 'TestRenderRoadmapPretty' -v
=== RUN   TestRenderRoadmapPrettyAt80
--- PASS: TestRenderRoadmapPrettyAt80 (0.00s)
=== RUN   TestRenderRoadmapPrettyLineWidths
--- PASS: TestRenderRoadmapPrettyLineWidths (0.00s)
=== RUN   TestRenderRoadmapPrettyTitleColumn
--- PASS: TestRenderRoadmapPrettyTitleColumn (0.00s)
=== RUN   TestRenderRoadmapPrettyPriorityVisibility
--- PASS: TestRenderRoadmapPrettyPriorityVisibility (0.00s)
=== RUN   TestRenderRoadmapPrettyUnscheduledFeature
    roadmap_pretty_test.go:420: expected at least 7 lines, got 6:
        Roadmap
        ════════════════════════════════════════════════════════════════════════════════

        No Milestone

--- FAIL: TestRenderRoadmapPrettyUnscheduledFeature (0.00s)
=== RUN   TestRenderRoadmapPrettyEmpty
--- PASS: TestRenderRoadmapPrettyEmpty (0.00s)
FAIL
FAIL	github.com/hmans/beans/internal/commands	0.625s
FAIL
```

Nur `TestRenderRoadmapPrettyUnscheduledFeature` schlägt an — Mutation ist präzise isoliert, alle
anderen Tests bleiben unberührt korrekt (kein Kollateral-Fehlschlag, der den Befund verwässern
würde).

**Rückbau:** `diff` zwischen Backup und mutierter Datei bestätigte genau die eine geänderte
Loop-Body-Zeile; nach `cp`-Rückbau `diff <backup> roadmap_pretty.go` leer. `git diff --stat` zeigt
danach ausschließlich die beabsichtigten Dateien (Test + bean), `roadmap_pretty.go` selbst
unverändert gegenüber dem Runde-1-Commit `229fe6a` (keine Produktionscode-Änderung nötig).

**Danach:** `command go test ./...` erneut grün (alle Pakete `ok`, EXIT=0).

**Ergänzung Mutations-Proben-Tabelle** (9. Zeile, zusätzlich zu den 8 aus Runde 1 + reviewer-
bestätigten):

| Mutation | welcher Test failte | Zeile getestet |
|---|---|---|
| `Unscheduled.Features`-Loop-Body → No-Op | `TestRenderRoadmapPrettyUnscheduledFeature`: `expected at least 7 lines, got 6` | ja (nach Ergänzung) |
