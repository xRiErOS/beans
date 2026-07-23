---
# beans-1ec3
title: 'roadmap: TTY-aware Ausgabe (Plain-Text statt Roh-Markdown)'
status: todo
type: epic
priority: normal
created_at: 2026-07-23T20:26:08Z
updated_at: 2026-07-23T20:45:05Z
---

`beans roadmap` ist ein Markdown-Artefakt-Generator für GitHub/Files: shields.io-Image-Badges
pro Zeile plus `[id](path)`-Links. Es gibt keinen TTY-Check — interaktiv im Terminal kommt der
Rohquelltext. Ziel dieses Epos: TTY-aware dual-mode nach dem gh/bat/glow-Idiom.

**Plan (verbindlich, local-only):** `docs/roadmap-tty-output/PLAN.md`
**Format-Spec:** `docs/roadmap-tty-output/DESIGN.md` · **Layout-Referenz:** `render-prototype.py`
Der Plan ist `ce-plan-reviewer`-grün (2 Runden) und PO-freigegeben (2026-07-23).

## Kontext für alle Kinder (DRY — nicht in den Blättern wiederholt)

**Repo:** `/Users/erik/Obsidian/tools/lean-stack/beans-src` (Fork `xRiErOS/beans`, Upstream `hmans/beans`).
Remotes: `fork` = xRiErOS, `origin` = hmans. **Push nur nach `fork`, nie nach `origin`.**

**Architektur:** `buildRoadmap()` bleibt unverändert und liefert `*roadmapData`. Neu daneben
`renderRoadmapPretty(data, width)` — Tree-Walker über dieselbe Struktur, symmetrisch zu
`renderRoadmapMarkdown`. Die Weiche sitzt in `roadmapOutput(...)`, das `RunE` aufruft.

## Entscheidungen (stabile Kennungen, im Plan ausführlich)

- **D01** Fix im beans-src CLI-Command, nicht in beans-tui.
- **D02** TTY-aware dual-mode: stdout ist TTY → gerendert; Pipe/Redirect → Markdown wie bisher.
- **D04** stdlib-Plain-Rendering, **kein neuer Dependency** (kein glamour, kein lipgloss, kein tabwriter).
- **D07** Titel werden **nie** abgeschnitten — Wrap mit Hanging-Indent, Attribute nur auf Zeile 1.
- **D08** Breite `W = clamp(terminalCols, 80, 110)`.
- **D10** Priority `normal` ausgeblendet; Milestone/Epic ohne Priority-Zelle.
- **D12** Loses Leaf ohne Epic direkt unter Milestone, kein Miscellaneous-Bucket.
- **D13** (PO 2026-07-23) Basis ist `fork/main` **nach** Merge von `fix/beans-ti53-roadmap-nested-hierarchy`.
  Renderer deckt **4 Ebenen** ab. Layout-Variante β: `titleCol = 17` (nicht 15), Leafs unter
  Feature echt eingerückt. Ersetzt D11 (war: Epics-only-Äste).
- **D14** (PO 2026-07-23) **Der Fork ist das Produkt, nicht der PR.** Definition-of-Done ist das
  installierte Binary in `/opt/homebrew/bin/beans`, nicht "Tests grün". PR #207 upstream bleibt
  offen liegen und ist kein Gate.
- **D15** Feature-Ast-Zeilen zeigen Priority (Milestone/Epic nicht).
- **D16** `utf8.RuneCountInString` für alle Breitenrechnungen (stdlib, D04-konform).
- **D17** Typ-Wort nie abschneiden; Präfix >= 17 → genau ein Leerzeichen vor Titel.
- **D18** `No Milestone` als nackte Zeile an Spalte 0, Leerzeile davor.

## Layout-Konstanten (Variante β)

`titleCol = 17` · `prioW = 8` · `statusW = 11` · `idW = 4` · `rightW = 27` · `titleW = W - 46`

## Globale Constraints

- **Der Markdown-Pfad muss byte-identisch bleiben.** Keine Änderung an `renderRoadmapMarkdown`,
  `renderBeanRef`, `typeBadge`, `firstParagraph`, `roadmap.tmpl`, `buildRoadmap`.
- Conventional Commits, Titel <= 50 Zeichen, `Refs: <bean-id>` im Body, **kein** `Co-Authored-By`.
- Table-driven Tests, erwartete Ausgaben als String-Literale — es gibt kein `internal/commands/testdata/`.
- **Hand-getippte Layout-Literale sind verboten.** Erwartete Render-Ausgaben immer aus dem
  Prototyp bzw. dem tatsächlichen Algorithmus erzeugen. Runde 1 des Plan-Reviews fiel genau
  hierüber (3 Blocker, Literale 2-5 Zeichen zu kurz).
- **`docs/` ist per `.git/info/exclude` von git ausgeschlossen** — Doku-Änderungen bleiben lokal,
  `git add docs/...` schlägt fehl. Das ist gewollt.

## Risiken

- **R01** Glyphen `■`/`▸`/`▪` sind East-Asian-Ambiguous — bei doppelter Breite verschieben sich
  Spalten um 1. Am echten Terminal prüfen (T6).
- **R02** `utf8.RuneCountInString` zählt CJK/Emoji als eine Zelle — bekannte, akzeptierte Grenze.
- **R04** `brew install`/`upgrade` überschreibt das Fork-Binary — dann T6 Build+Install wiederholen.

## Definition of Done

- `main` enthält ti53-Merge **und** TTY-Renderer, `fork/main` gepusht (nicht `origin`).
- `mise test` grün.
- `beans roadmap` gepiped byte-identisch zum Stand vorher.
- `beans roadmap` am Terminal: vier Ebenen, bündige Titel, keine Badges/Links, bei 80 Spalten
  kein Umbruch.
- `/opt/homebrew/bin/beans` meldet `0.4.2-fork.tty` und wirkt in `beans-tui` und `lean-stack`.


## Nachtrag 2026-07-23 (Gate-B-Verifikation, F03)

Risikoregister vervollständigt — R05 fehlte:

- **R05** Das Fork-Delta gegen `hmans/beans` wächst um einen zweiten Commit-Strang (Nesting-Fix
  plus TTY-Renderer). Ein späterer Upstream-Merge wird dadurch aufwendiger.
  **Umgang:** bewusst akzeptiert (D01/D14) — PR #207 bleibt offen liegen, der Fork ist das
  Produkt. Kein Aktionsbedarf, nur Registrierung.

## Nachtrag 2026-07-23 (PO, Vorflug-Check Realisierung) — D19 Test-Gate praezisiert

**Befund:** `mise test` ist `depends = ["codegen", "test:e2e"]` + `run = "go test ./..."`.
Der e2e-Teil ist lokal rot, aber nicht wegen Code:
`browserType.launch: Executable doesn't exist at .../ms-playwright/chromium_headless_shell-1208/...`
— das Playwright-Browser-Binary fehlt auf dieser Maschine. Alle Specs failen in 0-1 ms
(Setup-Fail, kein Assertion-Fail). `go test ./...` allein: **EXIT=0, gruen**.

**D19 (PO 2026-07-23):** Das Test-Gate dieses Epos ist **`go test ./...` gruen**, nicht
`mise test`. Der e2e-Pfad ist **explizit ausgeklammert**: dieser Epos beruehrt ausschliesslich
`internal/commands/roadmap*.go`, kein Frontend, kein GraphQL-Schema, kein Codegen-Input.
Das e2e-Rot ist preexistierend und umgebungsbedingt. Kein Playwright-Browser-Download
als Vorbedingung.

**Ersetzt** in der Definition of Done die Zeile "mise test gruen" durch:
- `go test ./...` gruen (EXIT=0).
- `beans roadmap` gepiped byte-identisch zum Stand vorher.
- Terminal-Smoke am echten TTY (T6).

Die uebrigen DoD-Punkte bleiben unveraendert.
