---
# beans-g5hz
title: T2 Layout-Spec auf Variante beta nachziehen
status: in-progress
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T20:53:36Z
parent: beans-1ec3
---

**Plan-Referenz:** `docs/roadmap-tty-output/PLAN.md` → Task 2. Der vollständige Prototyp-Quelltext
und die Präfix-Tabelle stehen dort — von dort übernehmen, nicht neu erfinden.

## Objective (User Story)

Als Implementierer des Go-Renderers brauche ich eine ausführbare, korrekte Layout-Referenz,
gegen die ich meine erwarteten Testausgaben prüfen kann — damit ich Layout-Literale nicht von
Hand tippe und der PO das Format abnehmen kann, bevor Go-Code entsteht.

## Hintergrund

DESIGN.md bezeichnet `render-prototype.py` als verbindliche Layout-Referenz. Der Prototyp kennt
bisher nur drei Ebenen und `TITLE_COL = 15`. D13 hat auf Variante β (`titleCol = 17`, vier Ebenen)
umgestellt. Ohne diesen Task implementieren die Folge-Tasks gegen eine veraltete Spec.

**Warum das kritisch ist:** Runde 1 des Plan-Reviews fiel über genau diesen Fehlertyp — von Hand
getippte Layout-Literale waren 2-5 Zeichen zu kurz (3 blockierende Findings).

## EARS-Anforderungen

- **EARS-1** THE Prototyp `render-prototype.py` SHALL `TITLE_COL = 17` verwenden und die vier
  Ebenen Milestone, Epic, Feature-Ast und Leaf rendern.
- **EARS-2** WHEN der Prototyp mit einer festen Breite W aufgerufen wird, THEN THE Ausgabe SHALL
  keine Zeile länger als W enthalten, und jede Zeile mit Right-Block SHALL exakt W Zeichen lang sein.
- **EARS-3** THE DESIGN.md SHALL die Präfix-Tabelle aller acht Zeilentypen mit ihren Längen und
  Padding-Werten enthalten.
- **EARS-4** THE DESIGN.md-Beispielblock SHALL zeichengleich mit dem `want`-Literal in
  `TestRenderRoadmapPrettyAt80` (Task 4) sein — Spec und Test dürfen nicht auseinanderlaufen.
- **EARS-5** THE DECISIONS.md SHALL D13-D18 enthalten und D11 als `🔴 Überholt durch D13` markieren.
- **EARS-6** THE TASKS.md T04 SHALL nicht länger "Epics-only-Gruppierung" fordern.
- **EARS-7** IF ein `git add docs/...` versucht wird, THEN THE Agent SHALL erkennen, dass `docs/`
  per `.git/info/exclude` ausgeschlossen ist, und **keinen** Commit für diesen Task erzeugen.

## Akzeptanzkriterien

- [ ] **SC-201** `python3 docs/roadmap-tty-output/render-prototype.py 80` läuft fehlerfrei;
      `awk '{print length($0)}' | sort -rn | head -1` liefert höchstens `80`.
- [ ] **SC-202** Dasselbe für Breite `110` (höchstens `110`).
- [ ] **SC-203** DESIGN.md enthält die Präfix-Tabelle mit allen acht Zeilentypen.
- [ ] **SC-204** DESIGN.md enthält `Titel-Start | fixe Spalte **17**`.
- [ ] **SC-205** DECISIONS.md enthält D13, D14, D15, D16, D17, D18; D11 trägt `🔴 Überholt`.
- [ ] **SC-206** TASKS.md T04 nennt "Epic- **und** Feature-Äste".
- [ ] **SC-207** `git status --short docs/` gibt nichts aus (Verzeichnis ignoriert) — kein Commit
      in diesem Task, das ist erwartet und kein Fehler.

## Betroffene Pfade

- `docs/roadmap-tty-output/render-prototype.py` (ersetzen)
- `docs/roadmap-tty-output/DESIGN.md` (Layout-Regeln, Ziel-Format-Block, Gruppierungs-Abschnitt)
- `docs/roadmap-tty-output/DECISIONS.md` (D13-D18 anhängen, D11 markieren)
- `docs/roadmap-tty-output/TASKS.md` (T04-Zeile)


## Nachtrag 2026-07-23 (Gate-B-Verifikation, F01/F02)

Die Verifikation der Operationalisierung fand zwei Plan-Anforderungen aus Task 2 Step 3, die
oben keine eigene EARS/SC-Entsprechung hatten. Sie sind hiermit verbindlich ergänzt.

### Zusätzliche EARS-Anforderungen

- **EARS-8** THE DESIGN.md SHALL im Abschnitt „### Gruppierung" den Satz „**Nur Epics sind Äste.**
  Features/Tasks bleiben Blätter …" nicht mehr enthalten; er SHALL durch die D13-Formulierung
  ersetzt sein („**Epics und Features sind Äste**", inkl. Hinweis auf flaches `featureGroup.Items`
  und fixe Render-Tiefe 4).
- **EARS-9** THE DESIGN.md SHALL im Abschnitt „## Bewusst ausgeklammert" die Zeile
  „- Rekursive Äste (Feature-Branches)." nicht mehr enthalten (durch D13 erledigt).

### Zusätzliche Akzeptanzkriterien

- [ ] **SC-208** Der „Ziel-Format"-Codeblock in DESIGN.md ist **zeichengleich** mit dem
      `want`-Literal aus `TestRenderRoadmapPrettyAt80` (Task 4 / bean beans-h30q).
      Prüfung: beide Blöcke extrahieren und `diff`en — keine Abweichung.
      Grund: Spec und Test dürfen nicht auseinanderlaufen; Runde 1 des Plan-Reviews fiel
      über genau diese Art Drift.
- [ ] **SC-209** `grep -c 'Nur Epics sind Äste' docs/roadmap-tty-output/DESIGN.md` liefert `0`.
- [ ] **SC-210** `grep -c 'Rekursive Äste (Feature-Branches)' docs/roadmap-tty-output/DESIGN.md`
      liefert `0`.

## Prelude 2026-07-23 (aus T1-Review, vor der eigentlichen Task-Arbeit erledigen)

Non-blocking Findings des `ce-specs-reviewer` zu T1 (`beans-l36h`). Quelle: T1-Review,
Verdict APPROVED, keine Blocker.

- **P-1** Go-Aufrufe **immer** als `command go ...` — die Shell hat eine `go`-Funktion, die
  den Compiler verdeckt und mit Exit 0 durchlaeuft, ohne einen Test auszufuehren. Siehe **D21**
  im Epic-bean `beans-1ec3`. Ein Beweis ohne `command`-Praefix zaehlt nicht.
- **P-2** Verlasse dich nicht auf die in T1 notierte Commit-Zahl ("12 unpushed gegen origin").
  Sie ist durch den T1-Abschluss-Commit `67ea3a5` bereits off-by-one. Zaehle bei Bedarf frisch:
  `git log origin/main..main --oneline | wc -l`.
