# DECISIONS — roadmap-tty-output

Think-Sammlung: `beans roadmap` erzeugt im Terminal rohes GitHub-Markdown ("wildes
Ergebnis"). Ziel: TTY-aware Ausgabe. Repo: `lean-stack/beans-src` (Fork `xRiErOS/beans`,
Upstream `hmans/beans`).

| Dxx | Hintergrund | Entscheidung | Status |
|-----|-------------|--------------|--------|
| D01 | roadmap-Command lebt in beans-src (Fork), Alternative wäre beans-tui-View | Fix im **beans-src CLI-Command** (`internal/commands/roadmap.go`). Fork-Delta gegen Upstream wird in Kauf genommen. Sammlung liegt hier. | 🟢 Fest |
| D02 | `gh`/`bat`/`glow`-Idiom: Terminal hübsch, Pipe roh | **Mode A — TTY-aware dual-mode**: stdout ist TTY → gerendert; Pipe/Redirect → Markdown wie bisher. | 🟢 Fest |
| D03 | Kern von Mode A | Default-Format = **auto** (TTY-Detect entscheidet). Explizite Override-Flags → **vertagt** (Q02). Auto-Detect deckt den Default-Fall. | 🟢 Fest |
| D04 | Render-Engine — User: "ohne Abhängigkeiten, guter Output reicht, nicht fancy" | **stdlib-Plain-Rendering** — kein glamour, kein lipgloss. TTY-Pfad: Image-Badges → Typ-Wort, URLs weg, Struktur geglättet, eingerückt. Farbe optional/vertagt (Q08). | 🟢 Fest |
| D05 | Darstellungsform (mehrere Iterationen: tree-tail → table → typ-wort-spalte) | **Typ-Wort-Spalte auf jeder Zeile** + Glyph (`■`/`▸`/`-`), Titel bündig Spalte 15, Priority/Status/id in festen Spalten. Format eingefroren, User an 80/110/160 selbst verifiziert. Details: DESIGN.md. | 🟢 Fest |
| D06 | Feld-Reihenfolge | Glyph · Typ · **Titel** · Priority · Status · id | 🟢 Fest |
| D07 | Titel-Behandlung (Relevanzbewertung braucht volle Titel) | **Nie abschneiden** → Wrap mit Hanging-Indent Spalte 15, Attribute nur Zeile 1. | 🟢 Fest |
| D08 | Breite (User: "warum 80?") | Dynamisch = Terminalbreite, **Cap 110** (Buch-Prinzip), **Floor 80**. | 🟢 Fest |
| D09 | Beschreibung + Tags | **Beide raus** (User: nur Titel; Tags nicht einbeziehen). Kontext steht im bean. | 🟢 Fest |
| D10 | Priority-`normal` | **Ausgeblendet** (Default = kein Rauschen); nur high/critical/low/deferred sichtbar. | 🟢 Fest |
| D11 | Baum-Scope | **Epics-only-Äste** (nur Epics expandieren). Rekursive Äste (Feature-Branches) vertagt. | 🔴 Überholt durch D13 |
| D12 | Loses Leaf ohne Epic | Direkt unter Milestone als `-`, kein Miscellaneous-Bucket. | 🟢 Fest |
| D13 | (PO 2026-07-23) Basis ist fork/main nach Merge von fix/beans-ti53-roadmap-nested-hierarchy | Renderer deckt **4 Ebenen** ab. Layout-Variante β: `titleCol = 17` (nicht 15), Leafs unter Feature echt eingerückt. Ersetzt D11 (war: Epics-only-Äste). | 🟢 Fest |
| D14 | (PO 2026-07-23) Definition-of-Done-Frage | **Der Fork ist das Produkt, nicht der PR.** Definition-of-Done ist das installierte Binary in `/opt/homebrew/bin/beans`, nicht "Tests grün". PR #207 upstream bleibt offen liegen und ist kein Gate. | 🟢 Fest |
| D15 | Priority-Sichtbarkeit auf Feature-Ast-Zeilen | Feature-Ast-Zeilen zeigen Priority (Milestone/Epic nicht). | 🟢 Fest |
| D16 | Breitenrechnung bei Multi-Byte-Glyphen | `utf8.RuneCountInString` für alle Breitenrechnungen (stdlib, D04-konform). | 🟢 Fest |
| D17 | Überlanges Typ-Wort sprengt Titel-Spalte | Typ-Wort nie abschneiden; Präfix ≥ 17 → genau ein Leerzeichen vor Titel. | 🟢 Fest |
| D18 | Darstellung der "No Milestone"-Sektion | Nackte Zeile an Spalte 0, Leerzeile davor. | 🟢 Fest |

## Grund-Mismatch (Rationale D02)

`beans roadmap` ist ein **Markdown-Artefakt-Generator für GitHub/Files**: Image-Badges
(`![task](https://img.shields.io/…)`) und `[id](path)`-Links sind GitHub-Render-Artefakte.
Interaktiv im Terminal kommt der Rohquelltext = "wild". Kein TTY-Check vorhanden
(`roadmap.go:86-88` `fmt.Print(md)` bedingungslos).

## Machbarkeit (grün)

glamour v0.10.0, lipgloss v1.1.1, golang.org/x/term v0.38.0 sind **bereits Deps** in
beans-src. Mode A braucht keine neue Abhängigkeit.
