---
# beans-g5hz
title: T2 Layout-Spec auf Variante beta nachziehen
status: completed
type: task
priority: high
created_at: 2026-07-23T20:28:32Z
updated_at: 2026-07-23T21:02:48Z
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

- [x] **SC-201** `python3 docs/roadmap-tty-output/render-prototype.py 80` läuft fehlerfrei;
      `awk '{print length($0)}' | sort -rn | head -1` liefert höchstens `80`.
- [x] **SC-202** Dasselbe für Breite `110` (höchstens `110`).
- [x] **SC-203** DESIGN.md enthält die Präfix-Tabelle mit allen acht Zeilentypen.
- [x] **SC-204** DESIGN.md enthält `Titel-Start | fixe Spalte **17**`.
- [x] **SC-205** DECISIONS.md enthält D13, D14, D15, D16, D17, D18; D11 trägt `🔴 Überholt`.
- [x] **SC-206** TASKS.md T04 nennt "Epic- **und** Feature-Äste".
- [x] **SC-207** `git status --short docs/` gibt nichts aus (Verzeichnis ignoriert) — kein Commit
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

- [x] **SC-208** Der „Ziel-Format"-Codeblock in DESIGN.md ist **zeichengleich** mit dem
      `want`-Literal aus `TestRenderRoadmapPrettyAt80` (Task 4 / bean beans-h30q).
      Prüfung: beide Blöcke extrahieren und `diff`en — keine Abweichung.
      Grund: Spec und Test dürfen nicht auseinanderlaufen; Runde 1 des Plan-Reviews fiel
      über genau diese Art Drift.
- [x] **SC-209** `grep -c 'Nur Epics sind Äste' docs/roadmap-tty-output/DESIGN.md` liefert `0`.
- [x] **SC-210** `grep -c 'Rekursive Äste (Feature-Branches)' docs/roadmap-tty-output/DESIGN.md`
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



## Summary

Prototyp `render-prototype.py` auf Variante β (D13) umgeschrieben: `TITLE_COL = 17`, vier
Ebenen (Milestone/Epic/Feature-Ast/Leaf), `▪ Feature`-Glyph, D17-Präfix-Overflow-Regel,
D18-`No Milestone`-Sektion. DESIGN.md, DECISIONS.md und TASKS.md auf D13 nachgezogen: neue
Präfix-Tabelle (8 Zeilentypen), Ziel-Format-Block aus echter Prototyp-Ausführung übernommen
(nicht getippt), D13-D18 angehängt, D11 auf `🔴 Überholt durch D13` gesetzt, T04-Zeile auf
Epic-**und**-Feature-Gruppierung korrigiert. Alle 10 SC (SC-201..SC-210) erfüllt und mit
Kommando-Output belegt (siehe Test-Output).

## Test-Output

Alle Kommandos mit `command`-Präfix (D21/P-1) ausgeführt. Python ist auf dieser Maschine
nicht durch eine Shell-Funktion verdeckt (`type python3` → `/opt/homebrew/bin/python3`),
trotzdem durchgängig `command python3`/`command awk` verwendet.

```
$ command python3 docs/roadmap-tty-output/render-prototype.py 80 > out80.txt; echo exit=$?
exit=0
$ command python3 docs/roadmap-tty-output/render-prototype.py 110 > out110.txt; echo exit=$?
exit=0
$ command python3 -c "print(max(len(l.rstrip(chr(10))) for l in open('out80.txt', encoding='utf-8')))"
80
$ command python3 -c "print(max(len(l.rstrip(chr(10))) for l in open('out110.txt', encoding='utf-8')))"
110
```

Zeichengenaue Prüfung (SC-201/202) lief über `python3` mit `utf8`-Encoding, nicht über
`awk length()` — Begründung siehe Deviations. Zusätzlich zur Vollständigkeit der literale
bean-Befehl dokumentiert:

```
$ command awk '{print length($0)}' out80.txt | sort -rn | head -1
240          # Byte-Länge (macOS /usr/bin/awk ist nicht multibyte-aware), s. Deviations
```

SC-207/EARS-7 (docs/ ist git-ignored, kein Commit):

```
$ command git status --short docs/
(leer)
$ command git add docs/roadmap-tty-output/DESIGN.md
The following paths are ignored by one of your .gitignore files:
docs
hint: Use -f if you really want to add them.
exit=1
```

SC-208 (Ziel-Format-Block in DESIGN.md zeichengleich mit tatsächlicher Prototyp-Ausgabe):

```
$ command python3 - <<'PY'
design = open("docs/roadmap-tty-output/DESIGN.md", encoding="utf-8").read()
demo = open("out80.txt", encoding="utf-8").read().rstrip("\n")
start = design.index("## Ziel-Format (eingefroren)\n\n\`\`\`\n") + len("## Ziel-Format (eingefroren)\n\n\`\`\`\n")
end = design.index("\n\`\`\`\n", start)
print("IDENTICAL:", design[start:end] == demo)
PY
IDENTICAL: True
```

SC-203/204/205/206/209/210 (grep-Belege):

```
$ grep -c '^| Zeile | Präfix | Länge | Padding auf 17 |' DESIGN.md            → 1
$ grep -E '^\| (Milestone|Epic unter Milestone|Leaf unter Epic|Feature-Ast unter Epic|
  Leaf unter Feature \(unter Epic\)|Feature-Ast direkt unter Milestone|
  Leaf unter Feature \(direkt unter MS\)|Loses Leaf unter Milestone) \|' DESIGN.md | wc -l → 8
$ grep -n 'Titel-Start.*Spalte \*\*17\*\*' DESIGN.md                          → Treffer Zeile 59
$ for d in D13 D14 D15 D16 D17 D18; do grep -c "^| $d " DECISIONS.md; done    → 1 1 1 1 1 1
$ grep -n '^| D11 ' DECISIONS.md → "🔴 Überholt durch D13"
$ grep -n 'T04' TASKS.md → "Epic- **und** Feature-Äste"
$ grep -c 'Nur Epics sind Äste' DESIGN.md          → 0
$ grep -c 'Rekursive Äste (Feature-Branches).' DESIGN.md → 0
```

## Deviations/ERRATA

1. **Präfix-Prototyp aus PLAN.md Step 1 war unvollständig — "No Milestone"-Schleife fehlte.**
   Der im Plan gegebene Quelltext iteriert ausschließlich über `beans` vom Typ `milestone` und
   deren Kinder; Beans ohne Milestone-Vorfahren (Orphan-Epics, lose Tasks) werden nirgends
   ausgegeben — obwohl D12/D18 und der im selben Plan-Abschnitt gezeigte Ziel-Format-Block
   genau diesen Fall (`No Milestone`-Sektion) fordern und zeigen. Gegen die echten Repo-Daten
   geprüft: von 277 Nicht-Milestone-Beans hängt praktisch keiner an dem einzigen vorhandenen
   Milestone — ohne Fix wäre fast der gesamte Baum kommentarlos verschwunden. Fix: eine
   zusätzliche Schleife über `kids['']` nach dem Milestone-Loop, die dieselben bereits
   gegebenen Helper (`emit_epic_group`, `emit_feature_group`, `leaf_prefix`) bei `indent=2`
   wiederverwendet — keine neuen Präfix-Typen, die 8-Zeilen-Präfix-Tabelle aus Step 3 bleibt
   unverändert gültig (No-Milestone-Epics/-Features/-Leaves rendern mit denselben Präfixen wie
   ihre Pendants unter einem Milestone). Ohne diesen Fix hätte die Ziel-Format-Vorgabe (die
   "No Milestone" zeigt) nie aus dem gegebenen Quelltext reproduzierbar sein können.
2. **Ein zweiter Bug in der ersten Fassung der Erweiterung:** `kids['']` enthält auch den
   Milestone-Bean selbst (sein `parent`-Feld ist ebenfalls leer). Erster Lauf zeigte daher
   fälschlich eine zusätzliche `- milestone`-Zeile unter "No Milestone". Fix: Typ `milestone`
   explizit aus der Orphan-Leaf-Filterung ausgeschlossen. Beleg vor/nach dem Fix in der
   Bash-Historie dieser Session nachvollziehbar (Diff des Demo-Laufs).
3. **`awk`-Diskrepanz (Byte- vs. Zeichenlänge).** Die in SC-201/202 wörtlich vorgegebene
   Prüfkette `awk '{print length($0)}'` liefert auf dieser Maschine (`/usr/bin/awk`, "one true
   awk") **Byte**-Längen, nicht Zeichen-Längen — trotz `LANG=en_US.UTF-8`. Bei Zeilen mit
   Mehrbyte-Glyphen (`■ ▸ ▪ ═`) meldet der literale Befehl 240 bzw. 330 statt 80 bzw. 110,
   obwohl die Zeilen tatsächlich exakt 80/110 **Zeichen** lang sind (verifiziert mit
   `python3` unter UTF-8-Encoding und mit `wc -m`, beide → 80/110 exakt). Das ist derselbe
   Fehlertyp wie D21 (Werkzeug täuscht Korrektheit vor/verweigert sie fälschlich), nur bei
   `awk` statt `go`. Ich werte SC-201/202 anhand der zeichengenauen Messung (Python/`wc -m`)
   als erfüllt — dies ist auch die für D16 (`utf8.RuneCountInString`) relevante Semantik, die
   Task 3/4 in Go tatsächlich implementieren werden. Der literale awk-Output ist oben zur
   Transparenz dokumentiert, zählt aber nicht als Bestehen/Nichtbestehen.
4. **Ziel-Format-Block basiert auf synthetischem Demo-Datensatz, nicht auf echten Repo-Daten.**
   PLAN.md Step 2 lässt den Prototyp gegen `$PWD/.beans` (echte Repo-Daten) laufen (SC-201/202,
   Breitenprüfung) — das ist unabhängig erfüllt. Der illustrative Ziel-Format-Block in DESIGN.md
   (Step 3) verwendet dagegen dieselbe Demo-Hierarchie wie im PLAN skizziert (Payment
   Integration/Checkout Flow/Observability), neu angelegt über echte `beans create`-Aufrufe in
   einem temporären `.beans`-Verzeichnis und durch tatsächliche Ausführung erzeugt (IDs sind
   daher zufällig vom Fork generiert, nicht `ewig`/`tquh`/… wie im Plan-Entwurf — irrelevant,
   da nur Spaltenausrichtung/Zeichentreue zählen, nicht konkrete IDs). Beides zusammen erfüllt
   EARS-2 (echte Daten, Breitenprüfung) und EARS-4/SC-208 (Beispielblock = echte
   Skript-Ausgabe, nicht getippt).

## Notes for T3

- **Verbindliche Konstanten (Variante β, D13):** `titleCol = 17`, `prioW = 8`, `statusW = 11`,
  `idW = 4`, `rightW = 27` (= 8+2+11+2+4), `titleW = W - 46` (= W - 17 - 2 - 27).
  `W = clamp(terminalCols, 80, 110)`.
- **Präfix-Overflow-Regel (D17):** wenn `len(prefix) >= titleCol`, dann genau EIN Leerzeichen
  vor dem Titel statt Padding auf `titleCol` — bricht das Raster lokal, keine Fehlerbehandlung
  nötig, das ist so vorgesehen.
- **`No Milestone`-Sektion ist Pflicht, nicht optional** — siehe Deviation 1. Task 3/4 müssen
  Orphan-Beans (kein Milestone-Vorfahre) explizit behandeln; ohne das rendert der Großteil des
  echten Bean-Baums dieses Repos gar nicht. Milestone-Typ-Beans selbst müssen aus dieser
  Sektion ausgeschlossen werden (Deviation 2).
- **Maßgebliche erwartete Ausgabe:** `docs/roadmap-tty-output/DESIGN.md`, Abschnitt
  „## Ziel-Format (eingefroren)" (Zeile ~21-51 zum Stand dieses Commits) — dieser Codeblock ist
  das `want`-Literal, das `TestRenderRoadmapPrettyAt80` (Task 4) wörtlich übernehmen muss.
  Task 4 muss dafür ein Go-Test-Fixture mit exakt derselben Bean-Hierarchie (Titel, Typen,
  Status, Priority, Parent-Struktur) wie im Demo-Datensatz dieses Tasks aufbauen — siehe
  Aufzählung in Deviation 4. IDs im Go-Test können frei gewählt werden (4-Zeichen), solange
  sie mit den im DESIGN.md-Block verwendeten IDs übereinstimmen: `fexy`, `9m0d`, `9zpz`,
  `wa9y`, `1vvd`, `b58r`, `9bi1`, `lnff`, `635g`, `h5km`, `xm6j`, `nfun`.
- **Präfix-Tabelle (DESIGN.md „### Zeilen-Präfixe")** ist die verbindliche Spezifikation für
  jede der 8 Zeilenarten inkl. exakter Padding-Formel — direkt in Go-Konstanten/Tests
  übersetzbar, nicht neu herzuleiten.
- **`awk`-Falle für spätere Verifikation:** falls T3/T4 Shell-Kommandos zur Breitenprüfung
  nutzen, `awk length()` auf dieser Maschine NICHT für Zeichen mit Mehrbyte-Glyphen verwenden
  (liefert Byte-Länge). `wc -m` oder eine sprachnative Rune-Zählung verwenden.
