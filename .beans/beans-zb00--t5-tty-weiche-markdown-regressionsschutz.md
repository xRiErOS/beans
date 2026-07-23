---
# beans-zb00
title: T5 TTY-Weiche + Markdown-Regressionsschutz
status: todo
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:28:43Z
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

- [ ] **SC-501** `TestRoadmapOutputSwitchesOnTTY` grün — Pipe-Pfad enthält `img.shields.io` und
      `# Roadmap`; TTY-Pfad enthält weder Badges noch `](`, beginnt mit `Roadmap
` und enthält
      `■ Milestone`.
- [ ] **SC-502** `TestRoadmapMarkdownByteIdentical` grün — `roadmapOutput(data, false, ...)` ist
      zeichengleich mit `renderRoadmapMarkdown(data, ...)`.
- [ ] **SC-503** `command gofmt -l internal/commands/` gibt nichts aus.
- [ ] **SC-504** `command go build ./internal/commands/` ist still.
- [ ] **SC-505** `command go test ./internal/commands/` endet mit `ok`, keine `FAIL`-Zeile — die
      ti53-Bestandstests bleiben grün.
- [ ] **SC-506** `command go run ./cmd/beans roadmap | head -5` beginnt mit `# Roadmap` und
      enthält `img.shields.io` (gepiped = Markdown).
- [ ] **SC-507** `command go run ./cmd/beans roadmap --json | head -3` liefert JSON.
- [ ] **SC-508** Commit `feat(roadmap): render plain table on tty` mit `Refs: <bean-id>`.

## Betroffene Pfade

- `internal/commands/roadmap.go` — neue Funktion `roadmapOutput` (vor `buildRoadmap`, nach
  Zeile 99), `RunE`-Block ersetzen, Import `golang.org/x/term` ergänzen
- `internal/commands/roadmap_test.go` — Import `"strings"` ergänzen, zwei Tests anhängen

## Produziert

```go
func roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string) string
```
