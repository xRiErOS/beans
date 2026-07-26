# DESIGN — roadmap-tty-output

Konvergiertes Ziel-Format + Umsetzungsansatz für `beans roadmap` im Terminal.
Repo: `beans-src` (Fork `xRiErOS/beans`). Ausführbare Spec: `render-prototype.py`
(dieses Verzeichnis) — reproduziert das Format exakt aus echten Demo-Daten.

## Problem (Ist)

`beans roadmap` ist ein Markdown-Artefakt-Generator für GitHub/Files: shields.io-Image-
Badges pro Zeile + `[id](path)`-Links. **Kein TTY-Check** (`roadmap.go:86-88`
`fmt.Print(md)`) — interaktiv im Terminal kommt der Rohquelltext ("wild"), bei
`--beans-path` zusätzlich monströse `../../../../`-Link-Prefixes. Priority/Status
fehlen ganz.

## Lösung: Mode A — TTY-aware dual-mode (D02)

- **stdout ist TTY** → gerenderte Plain-Text-Tabelle (unten). Keine Badges, keine URLs.
- **Pipe/Redirect** → GitHub-Markdown **unverändert** (Regressionsschutz, Skripte bleiben heil).
- Detection: `term.IsTerminal(int(os.Stdout.Fd()))`. `NO_COLOR`/explizite Flags → vertagt (Q02).

## Ziel-Format (eingefroren)

```
Roadmap
════════════════════════════════════════════════════════════════════════════════

■ Milestone      Payment Integration                           todo         fexy
  ▸ Epic         Checkout Flow                                 in-progress  9m0d
    - task       Validate card number (Luhn)                   in-progress  9zpz
    - bug        Total rounds off by one cent        critical  todo         wa9y
    ▪ Feature    Stripe card entry                       high  todo         1vvd
      - task     Refactor payment reconciliation         high  todo         b58r
                 ledger to support multi-currency
                 settlement
  ▪ Feature      Apple Pay express button                high  draft        9bi1
    - task       Wire up sheet                                 todo         lnff
  - task         Update pricing copy                      low  todo         635g

No Milestone

  ▸ Epic         Observability                                 todo         h5km
    - task       Add trace IDs                                 todo         xm6j
  - task         Rotate signing key                            todo         nfun
```

Der Block ist **nicht von Hand getippt**, sondern die tatsächliche stdout-Ausgabe von
`render-prototype.py 80` gegen ein Demo-`.beans`-Verzeichnis mit dieser Hierarchie
(Milestone → Epic → Task/Bug/Feature-Ast → Task, plus ein Orphan-Epic und ein
Orphan-Task ohne Milestone). Jede Attributzeile ist exakt 80 Zeichen lang. Er ist
zeichengleich mit dem `want`-Literal, das Task 4 (`TestRenderRoadmapPrettyAt80`)
verwenden muss — Task 4 übernimmt diesen Block wörtlich, tippt ihn nicht neu.

### Layout-Regeln (exakt, aus Prototyp — Variante β, D13)

| Element | Regel |
|---------|-------|
| Hierarchie-Glyph | `■` Milestone · `▸` Epic · `▪` Feature-Ast · `-` Leaf |
| Typ-Wort-Spalte | jede Zeile trägt Typ als Wort (`Milestone`/`Epic`/`Feature`/`feature`/`task`/`bug`) |
| Titel-Start | fixe Spalte **17** — alle Titel bündig, level-unabhängig |
| Feld-Reihenfolge | Glyph · Typ · **Titel** · Priority · Status · id |
| Right-Block | Breite **27** = priority(8, rechtsbündig) + 2 + status(11, links) + 2 + id(4) |
| Titel-Breite | `W − 17 − 2 − 27 = W − 46` |
| Breite W | `clamp(terminalCols, 80, 110)` — dynamisch, Cap 110, Floor 80 |
| Priority | `normal` **ausgeblendet**; Milestone/Epic ohne Priority-Zelle, Feature-Ast **mit** (D15) |
| Status | linksbündig, alle Ebenen |
| id | 4-Zeichen-Suffix (Prefix `beans-`/`lean-stack-` weg), rechts |
| **Titel nie abschneiden** | Wrap bei Titel-Breite, Hanging-Indent Spalte 17; Attribute nur Zeile 1 |
| Beschreibung / Tags | **raus** |
| Loses Leaf ohne Epic | direkt unter Milestone als `-`, **kein** Miscellaneous-Bucket |
| Überlanges Typ-Wort | Präfix ≥ 17 → genau 1 Space vor Titel, Raster lokal gebrochen (D17) |
| Sektion ohne Milestone | nackte Zeile `No Milestone` an Spalte 0, Leerzeile davor (D18) |

### Zeilen-Präfixe (verbindlich)

| Zeile | Präfix | Länge | Padding auf 17 |
|-------|--------|-------|----------------|
| Milestone | `■ Milestone` | 11 | 6 |
| Epic unter Milestone | `··▸ Epic` | 8 | 9 |
| Leaf unter Epic | `····- <typ>` | 6+len | 11−len |
| Feature-Ast unter Epic | `····▪ Feature` | 13 | 4 |
| Leaf unter Feature (unter Epic) | `······- <typ>` | 8+len | 9−len |
| Feature-Ast direkt unter Milestone | `··▪ Feature` | 11 | 6 |
| Leaf unter Feature (direkt unter MS) | `····- <typ>` | 6+len | 11−len |
| Loses Leaf unter Milestone | `··- <typ>` | 4+len | 13−len |

(`·` = Leerzeichen.) Längster Fall: Leaf unter Feature unter Epic mit Typ `feature` = 8+7 = **15** → 2 Leerzeichen Rest. Deshalb Spalte 17 und nicht 15.

### Gruppierung

- Milestone → Epics und Features (Äste, expandieren offene Kinder) + lose offene Leaf-Kinder als `-`-Zeilen direkt drunter.
- **Epics und Features sind Äste** (D13, seit ti53/PR #207). Ein Feature ohne offene Leaf-Kinder ist kein Container und wird als Blatt gerendert. `featureGroup.Items` ist bereits flach — Leafs unter verschachtelten Features sind hineingeflattet, die Render-Tiefe ist damit fix 4.
- `No Milestone`-Sektion (Unscheduled Epics + Orphans) bleibt, gleich gerendert.

## Umsetzung

- **Kein neuer Dep** (D04): reines stdlib-Rendering, kein glamour/lipgloss.
- `buildRoadmap()` bleibt (liefert `roadmapData`). **Neu:** `renderRoadmapPretty(data, width) string` — Tree-Walker über dieselbe Struktur, symmetrisch zur bestehenden `renderRoadmapMarkdown`.
- `roadmapCmd.RunE`: nach `--json`-Zweig → wenn TTY `renderRoadmapPretty`, sonst `renderRoadmapMarkdown` (Ist-Pfad).
- Wrapping: stdlib (`strings`/eigener Word-Wrap auf Titel-Breite). Kein `text/tabwriter` nötig (feste Spaltenbreiten reichen; tabwriter kann Wrap nicht).
- Prototyp `render-prototype.py` ist die verbindliche Layout-Referenz (liest `beans list --json --full`).

## Bewusst ausgeklammert (vertagt, nicht vergessen)

- **Q02** Flag-Surface (`--format`/`--color`) — Auto-Detect deckt Default; Overrides später.
- **Q08** Farbe im TTY (stdlib-ANSI) — erst mono, Farbe optional als Politur.
- **Q06** Upstream-PR an `hmans/beans` statt Fork-Delta.
- Priority `normal` sichtbar machen (aktuell versteckt).
