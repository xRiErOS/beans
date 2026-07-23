---
# beans-h30q
title: T4 Tree-Walker renderRoadmapPretty
status: todo
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:28:43Z
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

- [ ] **SC-401** `TestRoadmapClampWidth` grün (6 Fälle: 40, 0, 80, 96, 110, 200).
- [ ] **SC-402** `TestRenderRoadmapPrettyAt80` grün — Output zeichengleich mit dem `want`-Literal.
- [ ] **SC-403** `TestRenderRoadmapPrettyLineWidths` grün für 80, 96 und 110 — keine Zeile über W.
- [ ] **SC-404** `TestRenderRoadmapPrettyTitleColumn` grün — jede Bean-Zeile hat an Rune-Index 17
      kein Leerzeichen.
- [ ] **SC-405** `TestRenderRoadmapPrettyEmpty` grün.
- [ ] **SC-406** Der Renderer enthält keine eigene Sortierung (kein `sort.` in
      `renderRoadmapPretty`) — Reihenfolge kommt aus `buildRoadmap`.
- [ ] **SC-407** Fehlschlag-Steps vor Implementierung ausgeführt (RED vor GREEN).
- [ ] **SC-408** Commit `feat(roadmap): pretty tree walker for tty` mit `Refs: <bean-id>`.

## Betroffene Pfade

- `internal/commands/roadmap_pretty.go` (erweitern)
- `internal/commands/roadmap_pretty_test.go` (erweitern)

## Produziert (für Task 5)

```go
func renderRoadmapPretty(data *roadmapData, width int) string
func roadmapClampWidth(cols int) int
```
