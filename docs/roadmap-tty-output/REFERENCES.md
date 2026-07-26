# REFERENCES — roadmap-tty-output

## Code (beans-src, Fork xRiErOS/beans)

- `internal/commands/roadmap.go:58-90` — `roadmapCmd`, RunE. `fmt.Print(md)` @86-88 = **kein TTY-Check**.
- `internal/commands/roadmap.go:373-391` — `typeBadge()`: emittiert `![type](https://img.shields.io/badge/…)`. Farb-Map @379 (bug/feature/task/epic/milestone → hex).
- `internal/commands/roadmap.go:359-371` — `renderBeanRef()`: `[id](path)`-Links.
- `internal/commands/roadmap.go:339-356` — `renderRoadmapMarkdown()`: text/template mit FuncMap.
- `internal/commands/roadmap.tmpl` — Struktur: `# Roadmap` / `## Milestone` / `### Epic` / `### Miscellaneous` / `## No Milestone`. `beanLine` = Badge + Title + Ref.
- `internal/commands/roadmap.go:93-214` — `buildRoadmap()`: Gruppierung/Sortierung (Milestone→Epic→Items, Unscheduled, Orphans).
- `internal/commands/roadmap_test.go` — Bestandstests.

## Deps (schon vorhanden — keine neue Abhängigkeit nötig)

- `github.com/charmbracelet/glamour v0.10.0` — Markdown → ANSI.
- `github.com/charmbracelet/lipgloss v1.1.1-…` — Styling/Tree.
- `golang.org/x/term v0.38.0` — `term.IsTerminal`.
- `github.com/mattn/go-isatty v0.0.20` (indirect).

## Idiom-Vorbilder

- `gh`, `bat`, `glow` — TTY-Detect: Terminal → gerendert, Pipe → roh. `--color=auto|always|never`.

## Repro

`beans roadmap` im `lean-stack`-Repo → großer Markdown-Dump mit shields.io-Badges pro Zeile.
Gepiped byte-für-byte identisch (kein TTY-Zweig).

## Prototyp + Demo-Daten

- `render-prototype.py` (dieses Verzeichnis) — ausführbare Layout-Referenz, liest
  `beans list --json --full`. Aufruf: `python3 render-prototype.py [breite]` (bare = Terminalbreite, Cap 110, Floor 80). Erwartet Pfad in `/tmp/roadmap_bp.txt`.
- Demo-Beans-Repo (scratchpad, ephemer): 2 Milestones (Payment Integration, Observability) mit Epics + Leaves, variiert in type/priority/status, inkl. einem absichtlich langen Titel für den Wrap-Fall.
