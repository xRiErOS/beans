# roadmap TTY-Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `beans roadmap` gibt im Terminal eine gerenderte Plain-Text-Tabelle aus (Milestone/Epic/Feature/Leaf, Titel nie abgeschnitten), bei Pipe/Redirect unverändert GitHub-Markdown — und das Ergebnis landet als installiertes Fork-Binary in Eriks Alltag.

**Architecture:** `buildRoadmap()` bleibt unverändert und liefert weiterhin `*roadmapData`. Neu daneben: ein zweiter Renderer `renderRoadmapPretty(data, width)`, symmetrisch zum bestehenden `renderRoadmapMarkdown(data, links, linkPrefix)` — ein Tree-Walker über dieselbe Datenstruktur, der feste Spaltenbreiten mit stdlib-Mitteln setzt. Die Weiche zwischen beiden sitzt in einer neuen, testbaren Funktion `roadmapOutput(...)`, die `roadmapCmd.RunE` aufruft; `RunE` selbst bleibt dünn und ermittelt nur TTY-Zustand und Breite.

**Tech Stack:** Go (stdlib: `strings`, `unicode/utf8`, `sort`) · `golang.org/x/term` (bereits Dep, Präzedenz `internal/commands/list.go:190`) · Cobra · table-driven Tests (`mise test`)

## Global Constraints

- **Kein neuer Dependency** (D04). Nur stdlib + bereits in `go.mod` vorhandene Module. Insbesondere: **kein** `glamour`, **kein** `lipgloss`, **kein** `text/tabwriter` im Pretty-Pfad.
- **Der Markdown-Pfad bleibt byte-identisch** (Q07/D02). Keine Änderung an `renderRoadmapMarkdown`, `renderBeanRef`, `typeBadge`, `firstParagraph`, `roadmap.tmpl` oder `buildRoadmap`.
- **Basis-Branch ist `fork/main`** (D13), nicht `origin/main`. Fork-Delta gegen `hmans/beans` wird bewusst in Kauf genommen (D01, D14).
- **Titel werden nie abgeschnitten** (D07). Wrap mit Hanging-Indent, Attribute nur auf Zeile 1.
- **Layout-Konstanten (Variante β, D13):** `titleCol = 17`, `prioW = 8`, `statusW = 11`, `idW = 4`, `rightW = prioW + 2 + statusW + 2 + idW = 27`, `titleW = W - titleCol - 2 - rightW = W - 46`.
- **Breite:** `W = clamp(terminalCols, 80, 110)` (D08).
- **Priority `normal` wird nicht angezeigt** (D10). Milestone- und Epic-Zeilen tragen gar keine Priority-Zelle.
- **Commit-Konvention (Repo-CLAUDE.md):** Conventional Commits, Titel ≤ 50 Zeichen, relevante bean-ID als `Refs: <id>` im Body, **kein** `Co-Authored-By`.
- **Testkonvention (Repo-CLAUDE.md):** table-driven Tests. Erwartete Ausgaben als String-Literale im Test, **keine** `testdata/`-Golden-Dateien (das Repo hat kein `internal/commands/testdata/`).

---

## Entscheidungen dieses Plans

| Dxx | Hintergrund | Entscheidung | Status |
|-----|-------------|--------------|--------|
| D13 | `main` == `fork/main` == `origin/main` (0/0 divergent), Nesting-Fix liegt 8 Commits voraus auf `fix/beans-ti53-roadmap-nested-hierarchy`; PR #207 upstream seit 2026-07-17 offen; Eriks installiertes Binary `0.4.2-fork.ti53` hat den Fix bereits | **`fork/main` ist Integrationsbranch**: ti53 zuerst nach `fork/main` mergen (Fast-Forward), TTY-Renderer darauf. Renderer deckt 4 Ebenen ab. Layout-**Variante β**: `titleCol` von 15 auf **17**, Leafs unter Feature echt eingerückt. | 🟢 Fest (PO 2026-07-23) |
| D14 | PO: „es muss sich bei mir im Alltag auswirken. Ich denke nicht, dass beans tatsächlich durch hmans noch vorangetrieben wird." | **Der Fork ist das Produkt, nicht der PR.** Definition-of-Done ist das installierte Binary in `/opt/homebrew/bin/beans`, nicht „Tests grün". Upstream-PR (Q06) bleibt vertagt und ist kein Gate. | 🟢 Fest (PO 2026-07-23) |
| D15 | D10 nimmt Milestone/Epic die Priority-Zelle. Für die neue Feature-Ast-Ebene war nichts entschieden. | **Feature-Ast-Zeilen zeigen Priority** (wie Leafs). Rationale: Milestone/Epic sind reine Planungs-Container, ein Feature ist im beans-Modell eine Arbeitseinheit mit eigener Priorität — `high` auf einem Feature ist echtes Relevanzsignal. Status zeigen alle Ebenen. | 🟢 Fest (kippbar im PO-Review) |
| D16 | Zeichenbreiten-Frage: Titel können Umlaute/Emoji enthalten; `go-runewidth` liegt nur als *indirect* Dep vor | **`utf8.RuneCountInString`** für alle Breitenrechnungen (stdlib, D04-konform). Für Latin inkl. Umlaute korrekt. CJK/Emoji-Titel würden zu früh umbrechen — bekannte, akzeptierte Grenze (siehe Risiken). | 🟢 Fest |
| D17 | Typ-Wörter sind konfigurierbar (`cfg.TypeNames()`), können länger als `feature` (7) sein | **Typ-Wort wird nie abgeschnitten.** Ist das Zeilen-Präfix ≥ `titleCol`, folgt genau **ein** Leerzeichen und der Titel startet dahinter — Raster lokal gebrochen, Ausgabe deterministisch, kein Panic. | 🟢 Fest |
| D18 | `## No Milestone`-Sektion im Markdown-Pfad; im Pretty-Pfad gibt es keine Überschriften-Syntax | Sektionszeile ist der nackte Text **`No Milestone`** an Spalte 0, ohne Glyph, ohne Attribute, mit Leerzeile davor. Kinder darunter auf Milestone-Kind-Einrückung (Epic Indent 2, Feature Indent 2, Orphan-Leaf Indent 2). | 🟢 Fest |

## Offene Punkte (bewusst nicht in diesem Plan)

| Qxx | Frage | Status |
|-----|-------|--------|
| Q02 | Flag-Surface `--format` / `--color` | 🟣 Vertagt — Auto-Detect deckt den Default |
| Q05 | `NO_COLOR`-Convention | 🟣 Vertagt mit Q02 (Pretty-Pfad ist mono, kein Farb-Zweig) |
| Q06 | Upstream-PR an `hmans/beans` | 🟣 Vertagt, kein Gate (D14) |
| Q08 | Farbe im TTY | 🟣 Vertagt — erst mono |
| I01 | `bt-xy2i` ist ein in-progress-Duplikat des completed Epics `beans-f1t4` (gleicher Titel, gleicher Body) im `beans-src`-Repo | 🟣 Aufräumen in Task 6 |

---

## File Structure

| Datei | Verantwortung | Änderung |
|-------|---------------|----------|
| `internal/commands/roadmap.go` | Cobra-Command, `buildRoadmap`, Markdown-Renderer | Modify: `RunE` (post-merge :67-99) + neue Funktion `roadmapOutput`. Die TTY-/Breiten-Ermittlung bleibt bewusst inline in `RunE` — sie ist ein Dreizeiler über `os.Stdout`, die testbare Grenze ist `roadmapOutput`. |
| `internal/commands/roadmap_pretty.go` | **Neu.** Nur der Plain-Text-Renderer: Layout-Konstanten, Zeilen-Primitive, Tree-Walker. Getrennt von `roadmap.go`, weil dort schon 444 Zeilen Datenaufbau + Markdown liegen und beide Renderer unabhängig lesbar bleiben sollen. | Create |
| `internal/commands/roadmap_pretty_test.go` | **Neu.** Tests für Layout-Primitive und Tree-Walker. | Create |
| `internal/commands/roadmap_test.go` | Bestandstests (`TestBuildRoadmap`, `TestFirstParagraph`, `TestRenderBeanRef`, `TestStatusFiltering`) + ti53-Tests | Modify: Markdown-Regressionstest + `roadmapOutput`-Weichen-Test ergänzen |
| `docs/roadmap-tty-output/render-prototype.py` | Verbindliche Layout-Referenz | Modify: auf Variante β + 4 Ebenen |
| `docs/roadmap-tty-output/DESIGN.md` | Format-Spezifikation | Modify: Layout-Regeln auf β + Feature-Ebene |
| `docs/roadmap-tty-output/DECISIONS.md` | Entscheidungs-Register | Modify: D13–D18 nachtragen |

---

## Task 1: `fork/main` als Integrationsbasis herstellen

**Files:**
- Modify: keine Quelldatei — reine Branch-Operation im Repo `/Users/erik/Obsidian/tools/lean-stack/beans-src`

**Interfaces:**
- Consumes: nichts
- Produces: Ein `main`, auf dem `featureGroup` existiert. Alle folgenden Tasks setzen voraus, dass in `internal/commands/roadmap.go` diese Typen deklariert sind:
  ```go
  type roadmapData struct {
      Milestones  []milestoneGroup  `json:"milestones"`
      Unscheduled *unscheduledGroup `json:"unscheduled,omitempty"`
  }
  type unscheduledGroup struct {
      Epics    []epicGroup    `json:"epics,omitempty"`
      Features []featureGroup `json:"features,omitempty"`
      Other    []*bean.Bean   `json:"other,omitempty"`
  }
  type milestoneGroup struct {
      Milestone *bean.Bean     `json:"milestone"`
      Epics     []epicGroup    `json:"epics,omitempty"`
      Features  []featureGroup `json:"features,omitempty"`
      Other     []*bean.Bean   `json:"other,omitempty"`
  }
  type epicGroup struct {
      Epic     *bean.Bean     `json:"epic"`
      Items    []*bean.Bean   `json:"items,omitempty"`
      Features []featureGroup `json:"features,omitempty"`
  }
  type featureGroup struct {
      Feature *bean.Bean   `json:"feature"`
      Items   []*bean.Bean `json:"items,omitempty"`
  }
  ```
  `featureGroup.Items` ist **flach**: Leafs unterhalb verschachtelter Features sind hineingeflattet. Die maximale Render-Tiefe ist damit fix 4 (Milestone → Epic → Feature → Leaf).

- [ ] **Step 1: Ausgangslage verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git branch --show-current
git rev-list --left-right --count main...fix/beans-ti53-roadmap-nested-hierarchy
```

Erwartet:
```
main
0	8
```

Steht links etwas anderes als `0`, ist `main` divergiert — dann **stoppen** und den PO fragen, statt zu mergen.

- [ ] **Step 2: Merge durchführen (Fast-Forward)**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git merge --ff-only fix/beans-ti53-roadmap-nested-hierarchy
```

Erwartet: `Fast-forward` und eine Diffstat-Zeile mit `internal/commands/roadmap.go`, `roadmap.tmpl`, `roadmap_test.go`.

- [ ] **Step 3: Verifizieren, dass das Datenmodell da ist**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
grep -n 'type featureGroup' internal/commands/roadmap.go
```

Erwartet: eine Trefferzeile — `62:type featureGroup struct {`

- [ ] **Step 4: Tests laufen lassen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/
```

Erwartet: `ok  	github.com/hmans/beans/internal/commands`

Schlägt etwas fehl, ist das ein Blocker für alle Folge-Tasks — nicht weiterarbeiten, sondern melden.

- [ ] **Step 5: `fork/main` aktualisieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git push fork main
```

Erwartet: Push nach `https://github.com/xRiErOS/beans.git`, Branch `main`.

> Kein Push nach `origin` (= `hmans/beans`). Der Fork ist das Produkt (D14).

---

## Task 2: Layout-Spezifikation auf Variante β nachziehen

Der Prototyp ist laut DESIGN.md die *verbindliche* Layout-Referenz. Er kennt bisher nur 3 Ebenen und `titleCol = 15`. Bevor Go-Code entsteht, muss die Referenz stimmen — sonst implementiert Task 4 gegen eine veraltete Spec. Deliverable dieses Tasks ist eine Referenz-Ausgabe, die der PO abnehmen kann, bevor eine Zeile Go geschrieben ist.

**Files:**
- Modify: `docs/roadmap-tty-output/render-prototype.py`
- Modify: `docs/roadmap-tty-output/DESIGN.md:38-61`
- Modify: `docs/roadmap-tty-output/DECISIONS.md`

**Interfaces:**
- Consumes: nichts aus Task 1 (reines Doku-Artefakt)
- Produces: die verbindlichen Layout-Konstanten und die Präfix-Tabelle, gegen die Task 3 und Task 4 implementieren.

- [ ] **Step 1: Prototyp auf Variante β + 4 Ebenen umschreiben**

Ersetze den kompletten Inhalt von `docs/roadmap-tty-output/render-prototype.py` durch:

```python
#!/usr/bin/env python3
"""Verbindliche Layout-Referenz für `beans roadmap` im TTY (Variante β, D13).

Aufruf:  python3 render-prototype.py [breite]
         Ohne Argument: echte Terminalbreite, Cap 110, Floor 80.
         Erwartet den Pfad zum Demo-.beans-Verzeichnis in /tmp/roadmap_bp.txt.
"""
import sys, json, subprocess, textwrap, shutil

BP = open('/tmp/roadmap_bp.txt').read().strip()
if len(sys.argv) > 1:
    W = int(sys.argv[1])
else:
    W = max(80, min(shutil.get_terminal_size((110, 24)).columns, 110))

raw = subprocess.run(['beans', 'list', '--beans-path', BP, '--json', '--full'],
                     capture_output=True, text=True).stdout
beans = json.loads(raw)
kids = {}
for b in beans:
    kids.setdefault(b.get('parent') or '', []).append(b)

ARCHIVE = {'completed', 'scrapped'}
def open_(b): return b['status'] not in ARCHIVE

TITLE_COL = 17                              # Variante β (D13), war 15
PRIO_W, STAT_W, ID_W = 8, 11, 4
RIGHT_W = PRIO_W + 2 + STAT_W + 2 + ID_W    # = 27
TITLE_W = W - TITLE_COL - 2 - RIGHT_W       # = W - 46

def sid(b): return b['id'].split('-')[-1]

def right_block(b, show_prio):
    prio = (b.get('priority') or '') if show_prio else ''
    if prio == 'normal':                    # D10: Default nicht zeigen
        prio = ''
    stat = b.get('status') or ''
    return f"{prio:>{PRIO_W}}  {stat:<{STAT_W}}  {sid(b):<{ID_W}}"

def emit(prefix, b, show_prio):
    """Eine Bean-Zeile. Titel wrappt mit Hanging-Indent auf TITLE_COL (D07)."""
    lines = textwrap.wrap(b['title'], TITLE_W) or ['']
    if len(prefix) >= TITLE_COL:            # D17: Präfix zu lang -> 1 Space
        first = prefix + ' ' + lines[0]
    else:
        first = f"{prefix:<{TITLE_COL}}{lines[0]}"
    rb = right_block(b, show_prio)
    pad = max(2, W - RIGHT_W - len(first))
    print(first + ' ' * pad + rb)
    for cont in lines[1:]:
        print(' ' * TITLE_COL + cont)

def leaf_prefix(indent, b):
    return ' ' * indent + '- ' + b['type']

def emit_feature_group(f, indent):
    """Feature-Ast + seine Leafs. D15: Feature-Zeile zeigt Priority."""
    emit(' ' * indent + '▪ Feature', f, show_prio=True)
    for it in [c for c in kids.get(f['id'], []) if open_(c)]:
        emit(leaf_prefix(indent + 2, it), it, show_prio=True)

def emit_epic_group(e, indent):
    items = [c for c in kids.get(e['id'], []) if open_(c) and c['type'] != 'feature']
    feats = [c for c in kids.get(e['id'], []) if open_(c) and c['type'] == 'feature']
    if not items and not feats:
        return
    emit(' ' * indent + '▸ Epic', e, show_prio=False)
    for it in items:                                    # Leafs zuerst ...
        emit(leaf_prefix(indent + 2, it), it, show_prio=True)
    for f in feats:                                     # ... dann Feature-Äste
        emit_feature_group(f, indent + 2)

print("Roadmap")
print("═" * W)

for m in [b for b in beans if b['type'] == 'milestone']:
    print()
    emit("■ Milestone", m, show_prio=False)
    mkids = kids.get(m['id'], [])
    for e in [c for c in mkids if c['type'] == 'epic']:
        emit_epic_group(e, 2)
    for f in [c for c in mkids if c['type'] == 'feature' and open_(c)]:
        emit_feature_group(f, 2)
    for it in [c for c in mkids
               if c['type'] not in ('epic', 'feature') and open_(c)]:   # D12
        emit(leaf_prefix(2, it), it, show_prio=True)
```

- [ ] **Step 2: Prototyp gegen echte Daten laufen lassen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
echo "$PWD/.beans" > /tmp/roadmap_bp.txt
python3 docs/roadmap-tty-output/render-prototype.py 80 | head -20
python3 docs/roadmap-tty-output/render-prototype.py 110 | head -20
```

Erwartet: In beiden Läufen beginnen alle Titel an derselben Spalte (Zeichen 18, 0-indiziert 17), und der Right-Block endet bündig bei Spalte 80 bzw. 110. Verifiziere mit:

```bash
python3 docs/roadmap-tty-output/render-prototype.py 80 | awk 'NR>2 && length($0)>0 {print length($0)}' | sort -u
```

Erwartet: nur Werte `≤ 80`, und für Zeilen mit Right-Block exakt `80`.

- [ ] **Step 3: Präfix-Tabelle in DESIGN.md eintragen**

Ersetze in `docs/roadmap-tty-output/DESIGN.md` den Abschnitt „### Layout-Regeln (exakt, aus Prototyp)" (aktuell Zeile 38-54) durch:

```markdown
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
```

Ersetze außerdem in DESIGN.md den Code-Block im Abschnitt „## Ziel-Format (eingefroren)". Der folgende Block ist **nicht von Hand getippt**, sondern der exakte Renderer-Output bei W=80 — jede Attributzeile ist exakt 80 Zeichen lang:

```
Roadmap
════════════════════════════════════════════════════════════════════════════════

■ Milestone      Payment Integration                           todo         ewig
  ▸ Epic         Checkout Flow                                 in-progress  tquh
    - task       Validate card number (Luhn)                   in-progress  mf38
    - bug        Total rounds off by one cent        critical  todo         dg21
    ▪ Feature    Stripe card entry                       high  todo         sq2h
      - task     Refactor payment reconciliation         high  todo         uswm
                 ledger to support multi-currency
                 settlement
  ▪ Feature      Apple Pay express button                high  draft        k04r
    - task       Wire up sheet                                 todo         pp01
  - task         Update pricing copy                      low  todo         lo0s

No Milestone

  ▸ Epic         Observability                                 todo         un01
    - task       Add trace IDs                                 todo         un02
  - task         Rotate signing key                            todo         or01
```

Er ist zeichengleich mit dem `want`-Literal in `TestRenderRoadmapPrettyAt80` (Task 4 Step 5) — Spec und Test dürfen nicht auseinanderlaufen.

Ersetze im Abschnitt „### Gruppierung" den Satz „**Nur Epics sind Äste.** Features/Tasks bleiben Blätter …" durch:

```markdown
- **Epics und Features sind Äste** (D13, seit ti53/PR #207). Ein Feature ohne offene Leaf-Kinder ist kein Container und wird als Blatt gerendert. `featureGroup.Items` ist bereits flach — Leafs unter verschachtelten Features sind hineingeflattet, die Render-Tiefe ist damit fix 4.
```

Und streiche in „## Bewusst ausgeklammert" die Zeile „- Rekursive Äste (Feature-Branches)." — sie ist durch D13 erledigt.

- [ ] **Step 4: D13–D18 in DECISIONS.md nachtragen**

Hänge die sechs Zeilen aus dem Abschnitt „Entscheidungen dieses Plans" (oben) an die Tabelle in `docs/roadmap-tty-output/DECISIONS.md` an und setze D11 auf `🔴 Überholt durch D13`.

- [ ] **Step 5: TASKS.md mit D13 in Einklang bringen**

`docs/roadmap-tty-output/TASKS.md` Zeile 10 (T04) fordert noch „Epics-only-Gruppierung aus buildRoadmap ableiten" — das widerspricht D13 (vier Ebenen). Ersetze die T04-Zeile durch:

```markdown
| T04 | mittel | Kurz-ID-Helper (Prefix strippen), Priority-`normal`-Filter, Gruppierung über Epic- **und** Feature-Äste aus buildRoadmap ableiten (D13, war Epics-only) | 🟣 Offen |
```

Sonst bleibt die Think-Sammlung nach Planausführung intern widersprüchlich.

- [ ] **Step 6: Commit**

> `docs/` ist in diesem Repo per `.git/info/exclude` von git ausgeschlossen (bewusst: die Denk-Kette soll nicht in Upstream-PR-Diffs landen — dieselbe Praxis wie beim ti53-Plan). Ein `git add` schlägt hier fehl. Dieser Task hat daher **keinen** Commit; die Doku-Änderungen bleiben lokal. Prüfe stattdessen, dass die Dateien geschrieben sind:

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git status --short docs/ 2>/dev/null
grep -c 'titleCol\|Spalte \*\*17\*\*' docs/roadmap-tty-output/DESIGN.md
grep -c 'D13' docs/roadmap-tty-output/DECISIONS.md
```

Erwartet: `git status` gibt für `docs/` nichts aus (ignoriert), beide `grep -c` liefern einen Wert ≥ 1.

---

## Task 3: Layout-Primitive in Go

Vier kleine, unabhängig testbare Funktionen. Sie kennen die Baumstruktur nicht — nur einzelne Zeilen.

**Files:**
- Create: `internal/commands/roadmap_pretty.go`
- Test: `internal/commands/roadmap_pretty_test.go`

**Interfaces:**
- Consumes: `*bean.Bean` (Felder `ID`, `Title`, `Type`, `Status`, `Priority`) aus `github.com/hmans/beans/pkg/bean`.
- Produces — diese Signaturen nutzt Task 4:
  ```go
  const (
      roadmapTitleCol = 17
      roadmapPrioW    = 8
      roadmapStatusW  = 11
      roadmapIDW      = 4
      roadmapRightW   = roadmapPrioW + 2 + roadmapStatusW + 2 + roadmapIDW // 27
  )
  func roadmapShortID(id string) string
  func roadmapRightBlock(b *bean.Bean, showPrio bool) string
  func roadmapWrapTitle(title string, width int) []string
  func roadmapLine(prefix string, b *bean.Bean, showPrio bool, width int) string
  ```
  `roadmapLine` liefert eine **mehrzeilige** Zeichenkette (Wrap-Folgezeilen inklusive), jede Zeile ohne abschließendes `\n`; die Zeilen sind mit `\n` verbunden, ohne Trailing-Newline.

- [ ] **Step 1: Failing test für `roadmapShortID` und `roadmapRightBlock` schreiben**

Erstelle `internal/commands/roadmap_pretty_test.go`:

```go
package commands

import (
	"testing"

	"github.com/hmans/beans/pkg/bean"
)

func TestRoadmapShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"prefixed", "beans-tquh", "tquh"},
		{"multi hyphen prefix", "lean-stack-ewig", "ewig"},
		{"bare", "mf38", "mf38"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roadmapShortID(tt.id); got != tt.want {
				t.Errorf("roadmapShortID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestRoadmapRightBlock(t *testing.T) {
	tests := []struct {
		name     string
		b        *bean.Bean
		showPrio bool
		want     string
	}{
		{
			name:     "high priority shown",
			b:        &bean.Bean{ID: "beans-dg21", Status: "todo", Priority: "high"},
			showPrio: true,
			want:     "    high  todo         dg21",
		},
		{
			name:     "normal priority hidden",
			b:        &bean.Bean{ID: "beans-mf38", Status: "in-progress", Priority: "normal"},
			showPrio: true,
			want:     "          in-progress  mf38",
		},
		{
			name:     "priority suppressed for containers",
			b:        &bean.Bean{ID: "beans-ewig", Status: "todo", Priority: "critical"},
			showPrio: false,
			want:     "          todo         ewig",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roadmapRightBlock(tt.b, tt.showPrio)
			if got != tt.want {
				t.Errorf("roadmapRightBlock() =\n%q\nwant\n%q", got, tt.want)
			}
			if len(got) != roadmapRightW {
				t.Errorf("roadmapRightBlock() width = %d, want %d", len(got), roadmapRightW)
			}
		})
	}
}
```

- [ ] **Step 2: Test laufen lassen, Fehlschlag verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run 'TestRoadmapShortID|TestRoadmapRightBlock'
```

Erwartet: Compile-Fehler `undefined: roadmapShortID`, `undefined: roadmapRightBlock`, `undefined: roadmapRightW`.

- [ ] **Step 3: Konstanten + beide Funktionen implementieren**

Erstelle `internal/commands/roadmap_pretty.go`:

```go
package commands

import (
	"fmt"
	"strings"

	"github.com/hmans/beans/pkg/bean"
)

// Layout constants for the TTY-rendered roadmap (variant beta, D13).
// See docs/roadmap-tty-output/DESIGN.md for the authoritative spec.
const (
	roadmapTitleCol = 17 // column where every title starts
	roadmapPrioW    = 8  // priority cell, right-aligned
	roadmapStatusW  = 11 // status cell, left-aligned
	roadmapIDW      = 4  // short ID cell, left-aligned
	roadmapRightW   = roadmapPrioW + 2 + roadmapStatusW + 2 + roadmapIDW // 27

	roadmapMinWidth = 80
	roadmapMaxWidth = 110
)

// roadmapShortID strips the repo prefix and returns the 4-character suffix.
// "beans-tquh" -> "tquh", "lean-stack-ewig" -> "ewig".
func roadmapShortID(id string) string {
	if i := strings.LastIndex(id, "-"); i >= 0 {
		return id[i+1:]
	}
	return id
}

// roadmapRightBlock renders the fixed-width attribute block: priority, status,
// short ID. Priority "normal" is never shown (D10); showPrio is false for
// container rows (milestone, epic).
func roadmapRightBlock(b *bean.Bean, showPrio bool) string {
	prio := ""
	if showPrio && b.Priority != "normal" {
		prio = b.Priority
	}
	return fmt.Sprintf("%*s  %-*s  %-*s",
		roadmapPrioW, prio,
		roadmapStatusW, b.Status,
		roadmapIDW, roadmapShortID(b.ID))
}
```

Der Import-Block enthält bewusst noch **kein** `"unicode/utf8"` — das wird erst in Step 7 gebraucht und wäre hier ein „imported and not used"-Compile-Fehler.

- [ ] **Step 4: Test laufen lassen, grün verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run 'TestRoadmapShortID|TestRoadmapRightBlock' -v
```

Erwartet: `--- PASS` für alle sieben Subtests, am Ende `ok`.

- [ ] **Step 5: Failing test für `roadmapWrapTitle` und `roadmapLine` schreiben**

Erweitere zuerst den Import-Block von `internal/commands/roadmap_pretty_test.go` auf:

```go
import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/hmans/beans/pkg/bean"
)
```

(In Step 1 fehlten beide bewusst, weil sie dort noch ungenutzt gewesen wären — Go lehnt ungenutzte Imports ab.)

Hänge dann an `internal/commands/roadmap_pretty_test.go` an:

```go
func TestRoadmapWrapTitle(t *testing.T) {
	tests := []struct {
		name  string
		title string
		width int
		want  []string
	}{
		{"fits", "Checkout Flow", 34, []string{"Checkout Flow"}},
		{"empty stays one line", "", 34, []string{""}},
		{
			name:  "wraps on word boundary",
			title: "Refactor payment reconciliation ledger to support multi-currency settlement",
			width: 34,
			want: []string{
				"Refactor payment reconciliation",
				"ledger to support multi-currency",
				"settlement",
			},
		},
		{
			name:  "hard-breaks an overlong word",
			title: "Supercalifragilisticexpialidocious",
			width: 10,
			want:  []string{"Supercalif", "ragilistic", "expialidoc", "ious"},
		},
		{"umlauts count as one cell", "Prüfung ändern", 8, []string{"Prüfung", "ändern"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roadmapWrapTitle(tt.title, tt.width)
			if len(got) != len(tt.want) {
				t.Fatalf("roadmapWrapTitle() = %q (%d lines), want %q (%d lines)",
					got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("line %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRoadmapLine(t *testing.T) {
	epic := &bean.Bean{ID: "beans-tquh", Title: "Checkout Flow", Type: "epic",
		Status: "in-progress", Priority: "normal"}

	got := roadmapLine("  ▸ Epic", epic, false, 80)
	// Prefix "  ▸ Epic" is 8 runes, so 9 spaces pad it to column 17.
	want := "  ▸ Epic         Checkout Flow" +
		strings.Repeat(" ", 23) +
		"          in-progress  tquh"
	if got != want {
		t.Errorf("roadmapLine() =\n%q\nwant\n%q", got, want)
	}
	if utf8.RuneCountInString(got) != 80 {
		t.Errorf("line width = %d, want 80", utf8.RuneCountInString(got))
	}
	// Title must start at column 17 (rune index), glyph counts as one cell.
	runes := []rune(got)
	if string(runes[roadmapTitleCol:roadmapTitleCol+len("Checkout")]) != "Checkout" {
		t.Errorf("title does not start at column %d: %q", roadmapTitleCol, got)
	}
}

func TestRoadmapLineWrapsWithHangingIndent(t *testing.T) {
	long := &bean.Bean{
		ID:       "beans-uswm",
		Title:    "Refactor payment reconciliation ledger to support multi-currency settlement",
		Type:     "task",
		Status:   "todo",
		Priority: "high",
	}
	got := roadmapLine("    - task", long, true, 80)
	lines := strings.Split(got, "\n")
	if len(lines) != 3 {
		t.Fatalf("want 3 lines, got %d:\n%s", len(lines), got)
	}
	// Attributes only on line 1 (D07).
	if !strings.HasSuffix(lines[0], "    high  todo         uswm") {
		t.Errorf("line 1 missing right block: %q", lines[0])
	}
	// Continuation lines: hanging indent, no attributes.
	for i, l := range lines[1:] {
		if !strings.HasPrefix(l, strings.Repeat(" ", roadmapTitleCol)) {
			t.Errorf("continuation %d not indented to %d: %q", i, roadmapTitleCol, l)
		}
		if strings.Contains(l, "uswm") {
			t.Errorf("continuation %d carries attributes: %q", i, l)
		}
	}
}

func TestRoadmapLineOverlongPrefix(t *testing.T) {
	// D17: a custom type longer than the raster gets exactly one space.
	b := &bean.Bean{ID: "beans-zz99", Title: "Titel", Type: "verylongcustomtype",
		Status: "todo", Priority: "normal"}
	got := roadmapLine("      - verylongcustomtype", b, true, 80)
	if !strings.HasPrefix(got, "      - verylongcustomtype Titel") {
		t.Errorf("overlong prefix not followed by single space + title: %q", got)
	}
}
```

- [ ] **Step 6: Test laufen lassen, Fehlschlag verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run 'TestRoadmapWrapTitle|TestRoadmapLine'
```

Erwartet: `undefined: roadmapWrapTitle`, `undefined: roadmapLine`.

- [ ] **Step 7: `roadmapWrapTitle` und `roadmapLine` implementieren**

Ergänze in `internal/commands/roadmap_pretty.go` den Import `"unicode/utf8"` und hänge an:

```go
// roadmapWrapTitle word-wraps a title to the given cell width. Words longer
// than the width are hard-broken. Never returns an empty slice — an empty
// title yields one empty line. Widths are counted in runes (D16): correct for
// Latin incl. umlauts; CJK/emoji titles wrap early.
func roadmapWrapTitle(title string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(title)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	cur := ""
	flush := func() {
		lines = append(lines, cur)
		cur = ""
	}
	for _, w := range words {
		// Hard-break words that cannot fit on a line of their own.
		for utf8.RuneCountInString(w) > width {
			if cur != "" {
				flush()
			}
			r := []rune(w)
			lines = append(lines, string(r[:width]))
			w = string(r[width:])
		}
		switch {
		case cur == "":
			cur = w
		case utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) <= width:
			cur += " " + w
		default:
			flush()
			cur = w
		}
	}
	if cur != "" {
		flush()
	}
	return lines
}

// roadmapLine renders one bean as one or more physical lines. The first line
// carries prefix, title and the right-hand attribute block; continuation lines
// carry only the wrapped title at the hanging indent (D07). The returned
// string has no trailing newline.
func roadmapLine(prefix string, b *bean.Bean, showPrio bool, width int) string {
	titleW := width - roadmapTitleCol - 2 - roadmapRightW
	if titleW < 1 {
		titleW = 1
	}
	parts := roadmapWrapTitle(b.Title, titleW)

	prefixW := utf8.RuneCountInString(prefix)
	var first string
	if prefixW >= roadmapTitleCol {
		// D17: raster locally broken, keep exactly one separating space.
		first = prefix + " " + parts[0]
	} else {
		first = prefix + strings.Repeat(" ", roadmapTitleCol-prefixW) + parts[0]
	}

	pad := width - roadmapRightW - utf8.RuneCountInString(first)
	if pad < 2 {
		pad = 2
	}

	var sb strings.Builder
	sb.WriteString(first)
	sb.WriteString(strings.Repeat(" ", pad))
	sb.WriteString(roadmapRightBlock(b, showPrio))
	for _, cont := range parts[1:] {
		sb.WriteString("\n")
		sb.WriteString(strings.Repeat(" ", roadmapTitleCol))
		sb.WriteString(cont)
	}
	return sb.String()
}
```

- [ ] **Step 8: Test laufen lassen, grün verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run 'TestRoadmap' -v
```

Erwartet: alle Subtests `PASS`, am Ende `ok`.

- [ ] **Step 9: Commit**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git add internal/commands/roadmap_pretty.go internal/commands/roadmap_pretty_test.go
git commit -m "feat(roadmap): layout primitives for tty output"
```

---

## Task 4: Tree-Walker `renderRoadmapPretty`

**Files:**
- Modify: `internal/commands/roadmap_pretty.go`
- Test: `internal/commands/roadmap_pretty_test.go`

**Interfaces:**
- Consumes: `roadmapLine`, `roadmapTitleCol`, `roadmapRightW`, `roadmapMinWidth`, `roadmapMaxWidth` aus Task 3; `roadmapData`, `milestoneGroup`, `epicGroup`, `featureGroup`, `unscheduledGroup` aus Task 1.
- Produces — Task 5 nutzt:
  ```go
  func renderRoadmapPretty(data *roadmapData, width int) string
  func roadmapClampWidth(cols int) int
  ```
  `renderRoadmapPretty` liefert die vollständige Ausgabe **mit** abschließendem Newline, symmetrisch zu `renderRoadmapMarkdown`.

**Reihenfolge-Vertrag:** Der Walker sortiert nichts selbst. Er läuft die Slices exakt in der Reihenfolge ab, die `buildRoadmap` liefert (dort bereits nach `sortByTypeThenStatus` / Titel sortiert). Innerhalb einer Gruppe zuerst `.Items`, dann `.Features` — identisch zur Reihenfolge in `roadmap.tmpl`, damit Pretty- und Markdown-Pfad dieselbe Abfolge zeigen.

- [ ] **Step 1: Failing test für `roadmapClampWidth` schreiben**

Hänge an `internal/commands/roadmap_pretty_test.go` an:

```go
func TestRoadmapClampWidth(t *testing.T) {
	tests := []struct {
		name string
		cols int
		want int
	}{
		{"below floor", 40, 80},
		{"zero (not a terminal)", 0, 80},
		{"at floor", 80, 80},
		{"between", 96, 96},
		{"at cap", 110, 110},
		{"above cap", 200, 110},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roadmapClampWidth(tt.cols); got != tt.want {
				t.Errorf("roadmapClampWidth(%d) = %d, want %d", tt.cols, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Test laufen lassen, Fehlschlag verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run TestRoadmapClampWidth
```

Erwartet: `undefined: roadmapClampWidth`.

- [ ] **Step 3: `roadmapClampWidth` implementieren**

Hänge an `internal/commands/roadmap_pretty.go` an:

```go
// roadmapClampWidth constrains the terminal width to the readable band
// [80, 110] (D08). A non-positive value (stdout is not a terminal) yields the
// floor.
func roadmapClampWidth(cols int) int {
	if cols < roadmapMinWidth {
		return roadmapMinWidth
	}
	if cols > roadmapMaxWidth {
		return roadmapMaxWidth
	}
	return cols
}
```

- [ ] **Step 4: Test laufen lassen, grün verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run TestRoadmapClampWidth -v
```

Erwartet: sechs `PASS`-Subtests.

- [ ] **Step 5: Failing test für den vollen Walker schreiben**

Hänge an `internal/commands/roadmap_pretty_test.go` an:

```go
// prettyFixture builds a roadmapData covering every rendering path:
// milestone, epic with leafs, feature branch under an epic, feature branch
// directly under the milestone, loose leaf under the milestone (D12), an
// overlong title (D07) and a No-Milestone section (D18).
func prettyFixture() *roadmapData {
	return &roadmapData{
		Milestones: []milestoneGroup{{
			Milestone: &bean.Bean{ID: "beans-ewig", Title: "Payment Integration",
				Type: "milestone", Status: "todo", Priority: "normal"},
			Epics: []epicGroup{{
				Epic: &bean.Bean{ID: "beans-tquh", Title: "Checkout Flow",
					Type: "epic", Status: "in-progress", Priority: "normal"},
				Items: []*bean.Bean{
					{ID: "beans-mf38", Title: "Validate card number (Luhn)",
						Type: "task", Status: "in-progress", Priority: "normal"},
					{ID: "beans-dg21", Title: "Total rounds off by one cent",
						Type: "bug", Status: "todo", Priority: "critical"},
				},
				Features: []featureGroup{{
					Feature: &bean.Bean{ID: "beans-sq2h", Title: "Stripe card entry",
						Type: "feature", Status: "todo", Priority: "high"},
					Items: []*bean.Bean{
						{ID: "beans-uswm",
							Title:  "Refactor payment reconciliation ledger to support multi-currency settlement",
							Type:   "task", Status: "todo", Priority: "high"},
					},
				}},
			}},
			Features: []featureGroup{{
				Feature: &bean.Bean{ID: "beans-k04r", Title: "Apple Pay express button",
					Type: "feature", Status: "draft", Priority: "high"},
				Items: []*bean.Bean{
					{ID: "beans-pp01", Title: "Wire up sheet", Type: "task",
						Status: "todo", Priority: "normal"},
				},
			}},
			Other: []*bean.Bean{
				{ID: "beans-lo0s", Title: "Update pricing copy", Type: "task",
					Status: "todo", Priority: "low"},
			},
		}},
		Unscheduled: &unscheduledGroup{
			Epics: []epicGroup{{
				Epic: &bean.Bean{ID: "beans-un01", Title: "Observability",
					Type: "epic", Status: "todo", Priority: "normal"},
				Items: []*bean.Bean{
					{ID: "beans-un02", Title: "Add trace IDs", Type: "task",
						Status: "todo", Priority: "normal"},
				},
			}},
			Other: []*bean.Bean{
				{ID: "beans-or01", Title: "Rotate signing key", Type: "task",
					Status: "todo", Priority: "normal"},
			},
		},
	}
}

func TestRenderRoadmapPrettyAt80(t *testing.T) {
	want := `Roadmap
════════════════════════════════════════════════════════════════════════════════

■ Milestone      Payment Integration                           todo         ewig
  ▸ Epic         Checkout Flow                                 in-progress  tquh
    - task       Validate card number (Luhn)                   in-progress  mf38
    - bug        Total rounds off by one cent        critical  todo         dg21
    ▪ Feature    Stripe card entry                       high  todo         sq2h
      - task     Refactor payment reconciliation         high  todo         uswm
                 ledger to support multi-currency
                 settlement
  ▪ Feature      Apple Pay express button                high  draft        k04r
    - task       Wire up sheet                                 todo         pp01
  - task         Update pricing copy                      low  todo         lo0s

No Milestone

  ▸ Epic         Observability                                 todo         un01
    - task       Add trace IDs                                 todo         un02
  - task         Rotate signing key                            todo         or01
`
	got := renderRoadmapPretty(prettyFixture(), 80)
	if got != want {
		t.Errorf("renderRoadmapPretty(80) mismatch.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderRoadmapPrettyLineWidths(t *testing.T) {
	for _, w := range []int{80, 96, 110} {
		out := renderRoadmapPretty(prettyFixture(), w)
		for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
			if n := utf8.RuneCountInString(line); n > w {
				t.Errorf("width %d: line %d is %d runes: %q", w, i, n, line)
			}
		}
	}
}

func TestRenderRoadmapPrettyTitleColumn(t *testing.T) {
	out := renderRoadmapPretty(prettyFixture(), 110)
	for _, line := range strings.Split(out, "\n") {
		// Every bean row: milestone, epic, feature branch, and leafs at any
		// indent (2, 4 or 6 spaces). Continuation lines carry no "- "/glyph.
		isBeanRow := strings.HasPrefix(line, "■") ||
			strings.Contains(line, "▸ Epic") ||
			strings.Contains(line, "▪ Feature") ||
			strings.Contains(line, "- ")
		if !isBeanRow {
			continue
		}
		runes := []rune(line)
		if len(runes) <= roadmapTitleCol {
			continue
		}
		if runes[roadmapTitleCol] == ' ' {
			t.Errorf("title does not start at column %d: %q", roadmapTitleCol, line)
		}
	}
}

func TestRenderRoadmapPrettyEmpty(t *testing.T) {
	got := renderRoadmapPretty(&roadmapData{}, 80)
	want := "Roadmap\n" + strings.Repeat("═", 80) + "\n"
	if got != want {
		t.Errorf("empty roadmap = %q, want %q", got, want)
	}
}
```

Der Import-Block (`"strings"`, `"testing"`, `"unicode/utf8"`, `pkg/bean`) deckt diese Tests bereits ab — Task 3 Step 5 hat ihn vollständig gesetzt, hier ist nichts zu ergänzen.

> **Hinweis an den Implementierer:** Der erwartete String in `TestRenderRoadmapPrettyAt80` ist aus der Spec abgeleitet und im Whitespace fehleranfällig. Wenn er in Step 7 abweicht, prüfe zuerst mit `python3 docs/roadmap-tty-output/render-prototype.py 80` gegen den Prototyp (Task 2), welcher der beiden recht hat — der Prototyp ist die Referenz. Passe dann den Test-Literal an, **nicht** die Layout-Konstanten.

- [ ] **Step 6: Test laufen lassen, Fehlschlag verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run TestRenderRoadmapPretty
```

Erwartet: `undefined: renderRoadmapPretty`.

- [ ] **Step 7: Walker implementieren**

Hänge an `internal/commands/roadmap_pretty.go` an:

```go
// renderRoadmapPretty renders the roadmap as a plain-text table for terminal
// display (D02/D05). Symmetric to renderRoadmapMarkdown: same data, different
// surface. No colors, no links, no badges. Output ends with a newline.
//
// The walker performs no sorting of its own — it follows the slice order
// produced by buildRoadmap, and within a group renders Items before Features,
// matching roadmap.tmpl.
func renderRoadmapPretty(data *roadmapData, width int) string {
	var sb strings.Builder

	sb.WriteString("Roadmap\n")
	sb.WriteString(strings.Repeat("═", width))
	sb.WriteString("\n")

	emit := func(prefix string, b *bean.Bean, showPrio bool) {
		sb.WriteString(roadmapLine(prefix, b, showPrio, width))
		sb.WriteString("\n")
	}

	// leafPrefix builds "<indent>- <type>" for a leaf row.
	leafPrefix := func(indent int, b *bean.Bean) string {
		return strings.Repeat(" ", indent) + "- " + b.Type
	}

	var writeFeature func(fg featureGroup, indent int)
	writeFeature = func(fg featureGroup, indent int) {
		// D15: feature rows do show priority.
		emit(strings.Repeat(" ", indent)+"▪ Feature", fg.Feature, true)
		for _, it := range fg.Items {
			emit(leafPrefix(indent+2, it), it, true)
		}
	}

	writeEpic := func(eg epicGroup, indent int) {
		emit(strings.Repeat(" ", indent)+"▸ Epic", eg.Epic, false)
		for _, it := range eg.Items {
			emit(leafPrefix(indent+2, it), it, true)
		}
		for _, fg := range eg.Features {
			writeFeature(fg, indent+2)
		}
	}

	for _, mg := range data.Milestones {
		sb.WriteString("\n")
		emit("■ Milestone", mg.Milestone, false)
		for _, eg := range mg.Epics {
			writeEpic(eg, 2)
		}
		for _, fg := range mg.Features {
			writeFeature(fg, 2)
		}
		for _, it := range mg.Other { // D12: loose leaf, no Miscellaneous bucket
			emit(leafPrefix(2, it), it, true)
		}
	}

	if u := data.Unscheduled; u != nil {
		sb.WriteString("\nNo Milestone\n\n") // D18
		for _, eg := range u.Epics {
			writeEpic(eg, 2)
		}
		for _, fg := range u.Features {
			writeFeature(fg, 2)
		}
		for _, it := range u.Other {
			emit(leafPrefix(2, it), it, true)
		}
	}

	return sb.String()
}
```

- [ ] **Step 8: Test laufen lassen, grün verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run TestRenderRoadmapPretty -v
```

Erwartet: vier `PASS`-Testfunktionen. Bei Whitespace-Abweichung in `TestRenderRoadmapPrettyAt80`: Prototyp-Vergleich wie im Hinweis zu Step 5.

- [ ] **Step 9: Commit**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git add internal/commands/roadmap_pretty.go internal/commands/roadmap_pretty_test.go
git commit -m "feat(roadmap): pretty tree walker for tty"
```

---

## Task 5: TTY-Weiche verdrahten, Markdown-Pfad absichern

**Files:**
- Modify: `internal/commands/roadmap.go:67-99` (`roadmapCmd.RunE`)
- Modify: `internal/commands/roadmap.go` (neue Funktion `roadmapOutput`)
- Test: `internal/commands/roadmap_test.go`

> **Alle Zeilenangaben in diesem Task sind Post-Merge** (Stand nach Task 1, `roadmap.go` hat dann 604 statt 444 Zeilen). Die Angaben in `REFERENCES.md` beziehen sich dagegen auf den Pre-Merge-Stand und sind hier nicht anwendbar.

**Interfaces:**
- Consumes: `renderRoadmapPretty(data, width)`, `roadmapClampWidth(cols)` aus Task 4; `renderRoadmapMarkdown(data, links, linkPrefix)` (Bestand, `roadmap.go:499` post-merge).
- Produces:
  ```go
  func roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string) string
  ```
  Die Weiche liegt bewusst **nicht** in `RunE`: `RunE` liest `os.Stdout`, was im Test nicht sauber austauschbar ist. `roadmapOutput` bekommt den TTY-Zustand als Parameter und ist damit direkt table-driven testbar.

- [ ] **Step 1: Failing test für die Weiche schreiben**

Hänge an `internal/commands/roadmap_test.go` an:

```go
func TestRoadmapOutputSwitchesOnTTY(t *testing.T) {
	data := &roadmapData{
		Milestones: []milestoneGroup{{
			Milestone: &bean.Bean{ID: "beans-ewig", Title: "Payment Integration",
				Type: "milestone", Status: "todo", Priority: "normal"},
			Other: []*bean.Bean{
				{ID: "beans-lo0s", Title: "Update pricing copy", Type: "task",
					Status: "todo", Priority: "normal"},
			},
		}},
	}

	pipe := roadmapOutput(data, false, 0, true, ".beans")
	if !strings.Contains(pipe, "img.shields.io") {
		t.Errorf("non-TTY output must stay GitHub markdown, got:\n%s", pipe)
	}
	if !strings.Contains(pipe, "# Roadmap") {
		t.Errorf("non-TTY output missing markdown heading:\n%s", pipe)
	}

	tty := roadmapOutput(data, true, 100, true, ".beans")
	if strings.Contains(tty, "img.shields.io") {
		t.Errorf("TTY output must not contain badges:\n%s", tty)
	}
	if strings.Contains(tty, "](") {
		t.Errorf("TTY output must not contain markdown links:\n%s", tty)
	}
	if !strings.HasPrefix(tty, "Roadmap\n") {
		t.Errorf("TTY output missing plain heading:\n%s", tty)
	}
	if !strings.Contains(tty, "■ Milestone") {
		t.Errorf("TTY output missing milestone glyph:\n%s", tty)
	}
}

// TestRoadmapMarkdownByteIdentical is the regression guard for Q07/D02: the
// piped path must not change when the TTY renderer is added.
func TestRoadmapMarkdownByteIdentical(t *testing.T) {
	data := &roadmapData{
		Milestones: []milestoneGroup{{
			Milestone: &bean.Bean{ID: "beans-ewig", Title: "Payment Integration",
				Type: "milestone", Status: "todo"},
			Epics: []epicGroup{{
				Epic: &bean.Bean{ID: "beans-tquh", Title: "Checkout Flow",
					Type: "epic", Status: "in-progress"},
				Items: []*bean.Bean{
					{ID: "beans-mf38", Title: "Validate card number", Type: "task",
						Status: "todo", Path: "beans-mf38--validate.md"},
				},
			}},
		}},
	}

	direct := renderRoadmapMarkdown(data, true, ".beans")
	viaSwitch := roadmapOutput(data, false, 0, true, ".beans")
	if direct != viaSwitch {
		t.Errorf("piped path diverged from renderRoadmapMarkdown.\n--- direct ---\n%s\n--- via switch ---\n%s",
			direct, viaSwitch)
	}
}
```

`internal/commands/roadmap_test.go` importiert bisher nur `"testing"`, `"time"`, `pkg/bean` und `pkg/config` (Zeile 3-9). Ergänze **`"strings"`** im Import-Block — sonst `undefined: strings`.

- [ ] **Step 2: Test laufen lassen, Fehlschlag verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run 'TestRoadmapOutputSwitchesOnTTY|TestRoadmapMarkdownByteIdentical'
```

Erwartet: `undefined: roadmapOutput`.

- [ ] **Step 3: `roadmapOutput` implementieren**

Füge in `internal/commands/roadmap.go` direkt **vor** `buildRoadmap` ein (also nach der schließenden Klammer von `roadmapCmd`, post-merge Zeile 99):

```go
// roadmapOutput picks the rendering surface. Terminal gets the plain-text
// table, everything else keeps the GitHub markdown byte-for-byte (D02, Q07) —
// the gh/bat/glow idiom. cols is the raw terminal width and is only consulted
// on the TTY path.
func roadmapOutput(data *roadmapData, isTTY bool, cols int, links bool, linkPrefix string) string {
	if isTTY {
		return renderRoadmapPretty(data, roadmapClampWidth(cols))
	}
	return renderRoadmapMarkdown(data, links, linkPrefix)
}
```

- [ ] **Step 4: Test laufen lassen, grün verifizieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -run 'TestRoadmapOutputSwitchesOnTTY|TestRoadmapMarkdownByteIdentical' -v
```

Erwartet: beide Testfunktionen `PASS`.

- [ ] **Step 5: `RunE` auf die Weiche umstellen**

Ersetze in `internal/commands/roadmap.go` den Block ab dem Kommentar `// Markdown output` **bis einschließlich der schließenden Klammern von `RunE` und des `cobra.Command`-Literals** (post-merge Zeile 88-99, also inklusive `\t},` und `}`) durch:

```go
		// Rendered output: pretty table on a terminal, markdown when piped.
		links := !roadmapNoLinks
		linkPrefix := roadmapLinkPrefix
		if links && linkPrefix == "" {
			// Default: relative path from cwd to .beans directory
			linkPrefix = defaultLinkPrefix()
		}

		fd := int(os.Stdout.Fd())
		isTTY := term.IsTerminal(fd)
		cols := 0
		if isTTY {
			if w, _, err := term.GetSize(fd); err == nil {
				cols = w
			}
		}

		fmt.Print(roadmapOutput(data, isTTY, cols, links, linkPrefix))
		return nil
	},
}
```

Ergänze im Import-Block von `internal/commands/roadmap.go` (Zeile 3-18, durch den Merge unverändert) die Zeile:

```go
	"golang.org/x/term"
```

(nach `"github.com/spf13/cobra"`, damit die Gruppierung der Fremd-Imports erhalten bleibt; `gofmt` sortiert korrekt). `os` ist bereits importiert (Zeile 8).

- [ ] **Step 6: Bauen und formatieren**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command gofmt -l internal/commands/
command go build ./internal/commands/
```

Erwartet: `gofmt -l` gibt **nichts** aus, `go build` ist still.

- [ ] **Step 7: Volle Test-Suite des Pakets**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go test ./internal/commands/ -v 2>&1 | tail -30
```

Erwartet: `ok`, keine `FAIL`-Zeile. Insbesondere müssen die ti53-Bestandstests weiterhin grün sein.

- [ ] **Step 8: Manuelle Ende-zu-Ende-Probe (beide Pfade)**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
command go run ./cmd/beans roadmap | head -5          # gepiped -> Markdown
command go run ./cmd/beans roadmap --json | head -3   # unverändert JSON
```

Erwartet: erster Aufruf beginnt mit `# Roadmap` und enthält `img.shields.io`; zweiter Aufruf liefert JSON. Der TTY-Pfad lässt sich hier nicht prüfen — das passiert in Task 6 mit dem installierten Binary.

- [ ] **Step 9: Commit**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git add internal/commands/roadmap.go internal/commands/roadmap_test.go
git commit -m "feat(roadmap): render plain table on tty"
```

---

## Task 6: Binary bauen, installieren, im Alltag verifizieren

Ohne diesen Task wirkt nichts (D14). Die Prozedur ist aus dem abgeschlossenen Epic `beans-f1t4` und seinen Tasks `beans-lk3p` / `beans-d7y9` übernommen.

**Files:**
- Keine Quelldatei. Build- und Installationsschritte + beans-Pflege.

**Interfaces:**
- Consumes: der auf `main` gemergte Stand aus Tasks 1–5.
- Produces: `/opt/homebrew/bin/beans` mit Version `0.4.2-fork.tty`.

- [ ] **Step 1: Volle Test-Suite**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
mise test 2>&1 | tail -20
```

Erwartet: kein `FAIL`. Schlägt etwas außerhalb von `internal/commands` fehl, prüfe zuerst per `git stash`, ob der Fehler schon vorher bestand — nur eigene Regressionen sind ein Blocker.

- [ ] **Step 2: Version-gestempeltes Binary bauen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
SHA=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
command go build -ldflags "\
-X github.com/hmans/beans/internal/version.Version=0.4.2-fork.tty \
-X github.com/hmans/beans/internal/version.Commit=$SHA \
-X github.com/hmans/beans/internal/version.Date=$DATE" \
-o /tmp/beans-fork ./cmd/beans
/tmp/beans-fork version
```

Erwartet: `beans 0.4.2-fork.tty (<sha>) built <datum>`.

> Build-Target ist `./cmd/beans`, **nicht** das Repo-Root (aus `beans-lk3p`). Der eingebettete Frontend-Stub unter `internal/web/dist/` muss vorhanden sein, damit das Backend kompiliert — er ist es bereits (`index.html`, `_app/`).

- [ ] **Step 3: TTY-Ausgabe am echten Terminal prüfen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
/tmp/beans-fork roadmap
```

Erwartet: gerenderte Tabelle mit `Roadmap`-Kopf, `═`-Linie, Glyphen `■ ▸ ▪ -`, keine `img.shields.io`-URLs, keine `](`-Links, alle Titel bündig.

Dann die Grenzbreite prüfen — bei 80 Spalten sitzt der Right-Block auf Kante:

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
tmux new-session -d -s roadmapsmoke -x 80 -y 40 \
  "/tmp/beans-fork roadmap | head -40 > /tmp/roadmap80.txt"
sleep 2
tmux kill-session -t roadmapsmoke 2>/dev/null
awk '{print length($0)}' /tmp/roadmap80.txt | sort -rn | head -3
```

Erwartet: kein Wert über `80`. Zeilen, die umbrechen würden, sind ein Blocker.

- [ ] **Step 4: Pipe-Pfad gegen das alte Binary diffen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
beans roadmap > /tmp/roadmap-old.md          # aktuell installiertes 0.4.2-fork.ti53
/tmp/beans-fork roadmap > /tmp/roadmap-new.md
diff /tmp/roadmap-old.md /tmp/roadmap-new.md && echo "IDENTISCH"
```

Erwartet: `IDENTISCH`. Jede Abweichung verletzt D02/Q07 — dann **nicht** installieren, sondern melden.

- [ ] **Step 5: Installieren**

```bash
cp /tmp/beans-fork /opt/homebrew/bin/beans
chmod +x /opt/homebrew/bin/beans
which beans
beans version
```

Erwartet: `/opt/homebrew/bin/beans` und `beans 0.4.2-fork.tty (<sha>)`.

> Als echte Datei kopieren, nicht symlinken (aus `beans-d7y9`). Achtung: ein späteres `brew install`/`brew upgrade` überschreibt das Fork-Binary — dann Steps 2 und 5 wiederholen.

- [ ] **Step 6: Cross-Repo-Wirksamkeit prüfen**

```bash
cd /Users/erik/Obsidian/tools/beans-tui/beans-tui-repository
beans roadmap
cd /Users/erik/Obsidian/tools/lean-stack
beans roadmap
```

Erwartet: in beiden fremden Repos die gerenderte Tabelle, kein Fehler, keine Markdown-Reste. Das ist der Alltags-Nachweis aus D14.

- [ ] **Step 7: beans pflegen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
beans list --json | python3 -c "import json,sys; [print(b['id'], b['status'], b['title']) for b in json.load(sys.stdin) if b['id']=='bt-xy2i']"
```

Setze das Duplikat-Epic `bt-xy2i` (I01 — inhaltsgleich mit dem completed Epic `beans-f1t4`) auf `scrapped`:

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
beans update bt-xy2i --status scrapped
```

Ergänze im Body die Begründung: `Duplikat von beans-f1t4 (completed 2026-07-21), im falschen ID-Namensraum angelegt.`

- [ ] **Step 8: Commit**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git status --short
git add .beans/bt-xy2i--fork-beans-lokal-live-setzen-roadmap-feature-nesti.md
git commit -m "chore(beans): bt-xy2i scrapped (dup of beans-f1t4)"
```

> `.beans/` **nur mit expliziten Einzelpfaden** stagen, nie per Glob — das Repo trägt fremde uncommittete bean-Änderungen (siehe `git status`).

- [ ] **Step 9: Fork pushen**

```bash
cd /Users/erik/Obsidian/tools/lean-stack/beans-src
git push fork main
```

Erwartet: Push nach `xRiErOS/beans`. **Kein** Push nach `origin`.

---

## Risiken

| Rxx | Risiko | Auswirkung | Umgang |
|-----|--------|------------|--------|
| R01 | Die Glyphen `■` (U+25A0), `▸` (U+25B8), `▪` (U+25AA) sind East-Asian-**Ambiguous** — Terminals mit `EastAsianWidth`-Einstellung rendern sie doppelt breit | Alle Spalten der betroffenen Zeile verschieben sich um 1 | Erik hat `■`/`▸` bereits an seinem echten Terminal verifiziert; `▪` ist derselben Klasse. Task 6 Step 3 prüft es am realen Terminal. Fallback wäre ASCII (`#`/`>`/`*`) — nur ziehen, wenn Step 3 es zeigt. |
| R02 | `utf8.RuneCountInString` zählt CJK/Emoji als eine Zelle (D16) | Titel mit CJK/Emoji brechen zu früh um, Right-Block rutscht | Bewusst akzeptiert. Eriks Titel sind Latin. Falls es je stört: `mattn/go-runewidth` liegt bereits als indirect Dep im Modulgraph — Umstellung ist ein Einzeiler in `roadmapWrapTitle`/`roadmapLine`. |
| R03 | Der erwartete String in `TestRenderRoadmapPrettyAt80` ist von Hand aus der Spec abgeleitet | Test schlägt beim ersten Lauf mit Whitespace-Diff fehl und kostet Zeit | Task 4 Step 5 enthält die Anweisung, gegen den Prototyp zu vergleichen und **den Test**, nicht die Konstanten anzupassen. |
| R04 | `brew install`/`brew upgrade` überschreibt `/opt/homebrew/bin/beans` | Fork-Binary weg, alter Zustand zurück | Bekannt (bean `beans-znpz`, vom PO bewusst nicht als Doku-Task getrackt). Wiederherstellung = Task 6 Steps 2+5. |
| R05 | Fork-Delta gegen `hmans/beans` wächst um einen zweiten Commit-Strang | Ein späterer Upstream-Merge wird aufwendiger | Bewusst (D01/D14). PR #207 bleibt offen liegen; der Fork ist das Produkt. |

## Definition of Done

- [ ] `main` in `beans-src` enthält den ti53-Merge **und** den TTY-Renderer; `fork/main` gepusht (nicht `origin`).
- [ ] `mise test` grün.
- [ ] `beans roadmap` gepiped ist byte-identisch zum Stand vor der Änderung (Task 6 Step 4: `IDENTISCH`).
- [ ] `beans roadmap` am Terminal zeigt die Tabelle mit vier Ebenen, bündigen Titeln, ohne Badges/Links.
- [ ] Bei 80 Spalten bricht keine Zeile um (Task 6 Step 3).
- [ ] `/opt/homebrew/bin/beans` meldet `0.4.2-fork.tty` und wirkt in `beans-tui` und `lean-stack` (D14).
- [ ] `render-prototype.py`, `DESIGN.md`, `DECISIONS.md`, `TASKS.md` spiegeln Variante β (D11 auf überholt, T04 auf vier Ebenen).
- [ ] `bt-xy2i` scrapped.
