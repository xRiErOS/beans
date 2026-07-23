---
# beans-zb00
title: T5 TTY-Weiche + Markdown-Regressionsschutz
status: completed
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T21:54:24Z
parent: beans-1ec3
blocked_by:
    - beans-h30q
---

**Plan-Referenz:** `docs/roadmap-tty-output/PLAN.md` → Task 5. Vollständiger Go-Quelltext und
beide Tests stehen dort.

## Objective (User Story)

Als Skript, das `beans roadmap > roadmap.md` aufruft, will ich weiterhin byte-identisches
GitHub-Markdown bekommen — während der Mensch am Terminal die gerenderte Tabelle sieht.

## Hintergrund

Das gh/bat/glow-Idiom: Terminal hübsch, Pipe roh. Die Weiche liegt bewusst **nicht** in `RunE`,
sondern in `roadmapOutput(...)` — `RunE` liest `os.Stdout`, was im Test nicht sauber austauschbar
ist. `roadmapOutput` bekommt den TTY-Zustand als Parameter und ist damit table-driven testbar.

**Alle Zeilenangaben sind Post-Merge** (nach T1 hat `roadmap.go` 604 statt 444 Zeilen):
- `roadmapCmd` / `RunE`: Zeile 67-99
- Kommentar `// Markdown output`: Zeile 88
- `renderRoadmapMarkdown`: Zeile 499
- Import-Block: Zeile 3-18 (durch den Merge unverändert), `os` bereits bei Zeile 8

Die Angaben in `REFERENCES.md` beziehen sich auf den Pre-Merge-Stand und gelten hier nicht.

Der zu ersetzende Block reicht ab `// Markdown output` **bis einschließlich** der schließenden
Klammern von `RunE` und des `cobra.Command`-Literals (Zeile 88-99, also inkl. `	},` und `}`) —
der Ersetzungsblock im Plan bringt diese Klammern selbst mit.

`roadmap_test.go` importiert bisher nur `"testing"`, `"time"`, `pkg/bean`, `pkg/config`
(Zeile 3-9). **`"strings"` muss ergänzt werden**, sonst `undefined: strings`.

## EARS-Anforderungen

- **EARS-1** WHEN stdout ein Terminal ist, THEN THE `roadmapOutput` SHALL
  `renderRoadmapPretty(data, roadmapClampWidth(cols))` liefern.
- **EARS-2** WHEN stdout kein Terminal ist, THEN THE `roadmapOutput` SHALL
  `renderRoadmapMarkdown(data, links, linkPrefix)` **unverändert** liefern.
- **EARS-3** THE `--json`-Zweig SHALL unverändert bleiben und Vorrang vor beiden Pfaden haben.
- **EARS-4** THE TTY-Ausgabe SHALL weder `img.shields.io` noch die Markdown-Link-Sequenz `](`
  enthalten.
- **EARS-5** THE Pipe-Ausgabe SHALL byte-identisch zum direkten Aufruf von
  `renderRoadmapMarkdown` mit denselben Argumenten sein (Regressionsschutz Q07/D02).
- **EARS-6** THE `RunE` SHALL die Terminalbreite nur ermitteln, WHEN stdout ein Terminal ist.

## Akzeptanzkriterien

- [x] **SC-501** `TestRoadmapOutputSwitchesOnTTY` grün — Pipe-Pfad enthält `img.shields.io` und
      `# Roadmap`; TTY-Pfad enthält weder Badges noch `](`, beginnt mit `Roadmap
` und enthält
      `■ Milestone`.
- [x] **SC-502** `TestRoadmapMarkdownByteIdentical` grün — `roadmapOutput(data, false, ...)` ist
      zeichengleich mit `renderRoadmapMarkdown(data, ...)`.
- [x] **SC-503** `command gofmt -l internal/commands/` gibt nichts aus — siehe Deviations/ERRATA
      (pre-existierende Dirtiness in anderen Dateien, nicht durch T5 verursacht).
- [x] **SC-504** `command go build ./internal/commands/` ist still.
- [x] **SC-505** `command go test ./internal/commands/` endet mit `ok`, keine `FAIL`-Zeile — die
      ti53-Bestandstests bleiben grün.
- [x] **SC-506** `command go run ./cmd/beans roadmap | head -5` beginnt mit `# Roadmap` und
      enthält `img.shields.io` (gepiped = Markdown).
- [x] **SC-507** `command go run ./cmd/beans roadmap --json | head -3` liefert JSON.
- [x] **SC-508** Commit `feat(roadmap): render plain table on tty` mit `Refs: <bean-id>`.

## Betroffene Pfade

- `internal/commands/roadmap.go` — neue Funktion `roadmapOutput` (vor `buildRoadmap`, nach
  Zeile 99), `RunE`-Block ersetzen, Import `golang.org/x/term` ergänzen
- `internal/commands/roadmap_test.go` — Import `"strings"` ergänzen, zwei Tests anhängen

## Produziert

```go
func roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string) string
```

## Prelude 2026-07-23 (aus T1-T4-Reviews, vor der Task-Arbeit lesen)

T3 und T4 waren jeweils erst in Runde 2 gruen — beide Male fand die **Mutations-Probe** eine
load-bearing Zeile ohne Test, bei komplett gruener Suite. Was T5 mitnimmt:

- **P-1 Umgebungsfallen (D21/D22, Epic-bean `beans-1ec3`).**
  - `go` ist eine **Shell-Funktion** (dotfiles-Sync), verdeckt den Compiler, laeuft mit Exit 0
    durch **ohne einen Test auszufuehren** → immer `command go test ./...`.
  - `awk` misst **Bytes statt Zeichen** → Breitenpruefungen mit `wc -m` oder Rune-Zaehlung.
  - `mise test` ist **kein** Gate (D19) — zieht `test:e2e` mit, Playwright-Browser fehlt lokal.

- **P-2 Mutations-Selbstpruefung ist Abschlussbedingung, nicht Kuer.** Fuer jede load-bearing
  Zeile deiner Weiche (die TTY-Bedingung selbst, der Fallback bei unbestimmbarer Breite, die
  Auswahl zwischen den beiden Renderern): Zeile brechen → failt mindestens ein Test? Wenn nein,
  fehlt der Test. Mutation setzen, Rot-Ausgabe zitieren, byte-identisch zurueckbauen. Der
  Reviewer reproduziert das und ergaenzt eigene Mutationen.

  Konkret gefunden in T3/T4, damit du dieselben Gattungen vermeidest:
  - Grenzwert-Testfall lag **neben** der Grenze (26 statt 17) → Mutation blieb gruen.
  - Multibyte-Testfall hatte **zufaellig gleiche Margin** (7 Runen == 8 Bytes, beide <= 8) →
    Rune-vs-Byte war nicht unterscheidbar.
  - Ein ganzer **Loop-Zweig** (`Unscheduled.Features`) war von keiner Fixture erreicht → No-Op
    lief durch die ganze Suite gruen.

- **P-3 `prettyFixture()` nicht erweitern, um neue Zweige abzudecken.** Sie speist
  `TestRenderRoadmapPrettyAt80`, dessen `want` der **eingefrorene** DESIGN.md-Block ist
  (1155 Runen, dreifach verifiziert: Prototyp == DESIGN.md == Go-Literal). Eine
  Fixture-Aenderung verschoebe die Spec-Referenz. Muster aus T4: eigenstaendiges
  `roadmapData`-Literal im neuen Test bauen.

- **P-4 Byte-Identitaet ist die Kern-Anforderung dieses Tasks.** Der Markdown-Pfad
  (`renderRoadmapMarkdown`, `renderBeanRef`, `typeBadge`, `firstParagraph`, `roadmap.tmpl`,
  `buildRoadmap`) darf sich **nicht um ein Byte** aendern. Golden-Snapshot vorher/nachher,
  zweifach verglichen. Ein Diff-freier `git diff` allein genuegt nicht als Beweis — die
  Ausgabe selbst vergleichen.

- **P-5 Bekannte Grenzen, nicht dein Scope:** kinderlose Orphan-Epics fehlen in **beiden**
  Ausgabepfaden (bug `beans-36fa`, Ursache in `buildRoadmap`, aelter als der Pretty-Pfad).
  `buildRoadmap` bleibt unberuehrt. Die Clamp-Grenzoperatoren sind Equivalent-Mutanten
  (`cols == 80` liefert in beiden Zweigen 80) — kein Coverage-Loch, kein Fix noetig.

- **P-6 Zahlen zaehlen, nicht schaetzen** (`grep -c "^    --- PASS"`).

## Summary

`roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string) string`
implementiert exakt wie im Produziert-Block spezifiziert, in `internal/commands/roadmap.go`
direkt nach dem `roadmapCmd`-Literal (vor `buildRoadmap`). `RunE` bleibt duenn: JSON-Zweig
unveraendert (EARS-3), danach TTY-Erkennung via `term.IsTerminal(int(os.Stdout.Fd()))`,
Terminalbreite nur bei TTY via `term.GetSize` ermittelt (EARS-6), Aufruf `roadmapOutput(...)`,
Ausgabe via `fmt.Print` (kein `Println`, da `renderRoadmapPretty`/`renderRoadmapMarkdown` bereits
ein abschliessendes `\n` liefern). `renderRoadmapMarkdown`, `renderBeanRef`, `typeBadge`,
`firstParagraph`, `roadmap.tmpl`, `buildRoadmap` — keine Zeile angefasst.

## Test-Output

RED (Compile-Fehler, `roadmapOutput` referenziert bevor implementiert):

```
$ command go test ./internal/commands/
# github.com/hmans/beans/internal/commands [github.com/hmans/beans/internal/commands.test]
internal/commands/roadmap_test.go:651:10: undefined: roadmapOutput
internal/commands/roadmap_test.go:661:10: undefined: roadmapOutput
internal/commands/roadmap_test.go:683:9: undefined: roadmapOutput
internal/commands/roadmap_test.go:696:9: undefined: roadmapOutput
FAIL	github.com/hmans/beans/internal/commands [build failed]
FAIL
```

GREEN (nach Implementierung, unge-cached voller Suite-Lauf nach `command go clean -testcache`):

```
$ command go test ./...
ok  	github.com/hmans/beans/internal/agent	0.525s
ok  	github.com/hmans/beans/internal/commands	2.638s
ok  	github.com/hmans/beans/internal/cors	0.314s
ok  	github.com/hmans/beans/internal/gitutil	10.300s
ok  	github.com/hmans/beans/internal/graph	3.483s
ok  	github.com/hmans/beans/internal/portalloc	1.869s
ok  	github.com/hmans/beans/internal/search	1.117s
ok  	github.com/hmans/beans/internal/terminal	3.997s
ok  	github.com/hmans/beans/internal/tui	0.754s
ok  	github.com/hmans/beans/internal/ui	2.720s
ok  	github.com/hmans/beans/internal/web	2.606s
ok  	github.com/hmans/beans/internal/worktree	7.375s
ok  	github.com/hmans/beans/pkg/bean	2.089s
ok  	github.com/hmans/beans/pkg/beancore	3.965s
ok  	github.com/hmans/beans/pkg/config	1.592s
ok  	github.com/hmans/beans/pkg/forge	1.528s
ok  	github.com/hmans/beans/pkg/safepath	1.607s
```

Alle Pakete `ok`, keine `FAIL`-Zeile, EXIT=0.

## Byte-Identitaets-Nachweis

Golden-Snapshot vor/nach, gebaut via `command go build -o <bin> ./cmd/beans` (Binary-Target, nicht
Repo-Root), gegen das echte `.beans/` dieses Repos, gepiped, je zweifach:

Vorher (Stand ohne T5-Aenderung, via `git stash push --keep-index` isoliert):

```
$ command go build -o beans-before ./cmd/beans
$ ./beans-before roadmap > before-run1.md
$ ./beans-before roadmap > before-run2.md
```

Nachher (Stand mit T5-Implementierung, `git stash pop`):

```
$ command go build -o beans-after ./cmd/beans
$ ./beans-after roadmap > after-run1.md
$ ./beans-after roadmap > after-run2.md
```

Pruefsummen (`shasum -a 256`), alle vier Dateien identisch:

```
ad13a79a59b8977e34a72e731982a5beed45520c611bda96682c8652e4742501  before-run1.md
ad13a79a59b8977e34a72e731982a5beed45520c611bda96682c8652e4742501  before-run2.md
ad13a79a59b8977e34a72e731982a5beed45520c611bda96682c8652e4742501  after-run1.md
ad13a79a59b8977e34a72e731982a5beed45520c611bda96682c8652e4742501  after-run2.md
```

Vergleich (beide Durchlaeufe):

```
$ cmp before-run1.md after-run1.md && echo "IDENTISCH (cmp exit 0)"
IDENTISCH (cmp exit 0)
$ diff before-run1.md after-run1.md && echo "diff: KEIN UNTERSCHIED"
diff: KEIN UNTERSCHIED
$ cmp before-run2.md after-run2.md && echo "IDENTISCH (cmp exit 0)"
IDENTISCH (cmp exit 0)
$ diff before-run2.md after-run2.md && echo "diff: KEIN UNTERSCHIED"
diff: KEIN UNTERSCHIED
```

89 Zeilen in allen vier Dateien, byte-identisch, zweifach reproduziert.

## Mutations-Proben

| Mutation | welcher Test failte | Zeile getestet |
|---|---|---|
| `roadmapOutput`: `if isTTY` → `if !isTTY` (Bedingung invertiert) | `TestRoadmapOutputSwitchesOnTTY` (beide Subtests) + `TestRoadmapMarkdownByteIdentical` + `TestRoadmapOutputZeroColsFallsBackTo80` — alle 4 | ja |
| `roadmapOutput`: `renderRoadmapPretty(data, roadmapClampWidth(cols))` → `renderRoadmapPretty(data, cols)` (Clamp-Fallback entfernt) | `TestRoadmapOutputZeroColsFallsBackTo80`: `separator width = 0, want 80` | ja |
| `roadmapOutput`: Markdown-Zweig `renderRoadmapMarkdown(data, links, linkPrefix)` → `renderRoadmapMarkdown(data, false, linkPrefix)` (links hart auf false) | `TestRoadmapMarkdownByteIdentical`: `got` ohne Link-Klammern, `want` mit — Diff zitiert | ja |

Jede Mutation isoliert nur die erwarteten Tests, keine Kollateral-Fehlschlaege in anderen Paketen.
Rueckbau nach jeder Mutation via `cp /tmp/roadmap.go.backup internal/commands/roadmap.go`,
`diff` gegen Backup leer (`RUECKBAU IDENTISCH`), danach volle Suite erneut gruen bestaetigt.

## Deviations/ERRATA

- **SC-503-Praezisierung:** `command gofmt -l internal/commands/` listet 17 Dateien, darunter
  `roadmap_test.go` (Struct-Feld-Alignment in `TestBuildRoadmap`/`TestFirstParagraph`, Zeilen 35
  und 147 — **weit vor** dem T5-Anhang ab Zeile ~620) und `roadmap_pretty.go` (aus T4). Verifiziert
  via `git show HEAD:internal/commands/roadmap_test.go` + `gofmt -l` auf dem Pre-T5-Snapshot:
  bereits dort dirty. Diese Dirtiness ist **pre-existierend**, nicht durch T5 verursacht, und laut
  Harte-Grenzen-Klausel ("kein Scope-Creep") nicht mein Fix-Scope. `internal/commands/roadmap.go`
  (mein einziger Produktionscode-Eingriff) ist gofmt-clean (`gofmt -l roadmap.go` → kein Output).
  Mein Anhang an `roadmap_test.go` selbst ist ebenfalls gofmt-clean (verifiziert per `gofmt -d`:
  der Diff endet vor meinem Anhang). SC-503 als woertlich formuliertes Kriterium ist auf diesem
  Repo-Stand technisch nicht erfuellbar, ohne unbeteiligte Dateien anzufassen — die Absicht
  (T5-Code ist gofmt-konform) ist erfuellt.
- Kein sonstiger Deviation vom bean/Design.

## Notes for T6

- Binary-Buildtarget ist `./cmd/beans`, nicht Repo-Root (`go build .` baut ein anderes Package).
- `beans roadmap` am echten Terminal (kein Pipe) zeigt jetzt den Pretty-Renderer aus T4
  (`renderRoadmapPretty`) automatisch — keine Flag noetig, reiner TTY-Check
  (`term.IsTerminal(int(os.Stdout.Fd()))`). Zum manuellen Smoke-Test: `command go run ./cmd/beans
  roadmap` direkt im Terminal ausfuehren (nicht gepiped) — R01 (East-Asian-Ambiguous-Glyphen
  ■/▸/▪) ist dort zu pruefen, T5 hat das nicht getestet (kein echtes TTY im Testlauf verfuegbar).
- Terminalbreite kommt aus `term.GetSize`; bei sehr schmalem/breitem Terminal greift
  `roadmapClampWidth` (80-110, D08) — am echten Terminal mit `< 80` und `> 110` Spalten pruefen.
- `/opt/homebrew/bin/beans` (D14 Definition-of-Done) ist von T5 **nicht** aktualisiert — das ist
  T6-Scope (Build+Install, DoD-Zeile "meldet 0.4.2-fork.tty").
- Fork-Push (D01/Kontext-Abschnitt Epic-bean) ebenfalls nicht Teil von T5 — "Kein Push. Der
  Supervisor entscheidet über Pushes."
