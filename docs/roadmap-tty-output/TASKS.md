# TASKS — roadmap-tty-output

Format eingefroren (D05-D12). Umsetzung Mode A, stdlib. Referenz: `render-prototype.py`, DESIGN.md.

> **Status-Hinweis (2026-07-24):** Diese Liste stammt aus der Think-Phase. Ihre `Txx`-Nummern
> sind **nicht** deckungsgleich mit den Task-beans des Epos `beans-1ec3` — die Spalte
> „umgesetzt in" stellt die Zuordnung her. Maßgeblich für den Status ist **beans**, nicht diese
> Datei. Realisierung abgeschlossen, Epos steht auf Tag `to-review`.

| Txx | Prio | Aufgabe | umgesetzt in | Status |
|-----|------|---------|--------------|--------|
| T01 | hoch | TTY-Detect + Breiten-Resolution: `term.IsTerminal(os.Stdout.Fd())`, `clamp(cols,80,110)` | `beans-zb00` (T5), Clamp in `beans-h30q` (T4) | 🟢 Done |
| T02 | hoch | `renderRoadmapPretty(data, width)` — Tree-Walker über `roadmapData`, Layout exakt nach DESIGN/Prototyp (Spalten, Right-Block 27, Titel-Wrap Hanging-Indent) | `beans-h30q` (T4), Primitive in `beans-ejoz` (T3) | 🟢 Done |
| T03 | hoch | `roadmapCmd.RunE` verdrahten: `--json` → JSON; sonst TTY? pretty : markdown (Ist-Pfad) | `beans-zb00` (T5) — Weiche als testbare Funktion `roadmapOutput(...)` | 🟢 Done |
| T04 | mittel | Kurz-ID-Helper (Prefix strippen), Priority-`normal`-Filter, Gruppierung über Epic- **und** Feature-Äste aus buildRoadmap ableiten (D13, war Epics-only) | `beans-ejoz` (T3) + `beans-h30q` (T4) | 🟢 Done |
| T05 | hoch | Tests: Markdown-Pfad **byte-identisch** (bestehende Golden), Pretty-Snapshot bei fixer Breite (80 + 110), Wrap-Fall (langer Titel) | `beans-zb00` (T5), Snapshot in `beans-h30q` (T4) | 🟢 Done |
| T06 | niedrig | (vertagt) Flags Q02, Farbe Q08, Upstream-PR Q06, rekursive Äste | — | 🟣 Offen (bewusst vertagt) |

## Nach der Realisierung entstanden

| Code | Befund | Senke |
|------|--------|-------|
| B01 | Kinderlose Orphan-**Epic** verschwindet aus **beiden** Ausgabepfaden (Ursache in `buildRoadmap`, älter als der Pretty-Pfad; Zwilling von `beans-n8zw`, dem Feature-Fall) | bean `beans-36fa` |
| Q01 | **R01 offen:** Rendert Eriks echter Terminal-Emulator `■ ▸ ▪` einspaltig? In tmux/pty ja — am echten Terminal nicht agentisch belegbar | PO-Gate |
