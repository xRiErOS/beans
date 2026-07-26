# QUESTIONS — roadmap-tty-output

| Qxx | Hintergrund | Relevanz | Frage | Status |
|-----|-------------|----------|-------|--------|
| Q01 | Pretty-Pfad braucht Render-Engine | Crux Mode A | glamour vs lipgloss vs stdlib? | 🟢 Gelöst → **D04** (stdlib, keine Lib) |
| Q02 | Auto-Default braucht manuelle Overrides | Nutzbarkeit | Flag-Surface `--format`/`--color`? | 🟣 Vertagt (User: deferred). Auto-Detect deckt Default. |
| Q03 | Image-Badges wertlos im Terminal | Symptom #1 | Badges droppen/Chips/plain? | 🟢 Gelöst → **D05** (Typ-Wort-Spalte) |
| Q04 | `[id](path)`-Links nutzlos | Lesbarkeit | ID plain / OSC-8? | 🟢 Gelöst → **D06** (4-Zeichen-ID, keine URL) |
| Q05 | `NO_COLOR`/`--color` Convention | Korrektheit | respektieren? | 🟣 Vertagt mit Q02 (erst mono, kein Farb-Zweig) |
| Q06 | Fix = Fork-Delta gegen `hmans/beans` | Wartung | Upstream-PR? | 🟣 Vertagt (nach erstem Wurf bewerten) |
| Q07 | Markdown-Pfad muss byte-identisch bleiben | Regression | Golden-Tests? | 🟢 Bestätigt → **T05** (Pipe-Pfad unangetastet + Golden) |
| Q08 | Farbe im TTY (stdlib-ANSI) | Politur | mono oder Farbe? | 🟣 Vertagt (erst mono; Farbe optional später) |
