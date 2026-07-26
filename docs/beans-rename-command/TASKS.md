# TASKS — beans-rename-command

Abgeleitete/manifeste Aufgaben (Think-Phase — noch keine Realisierung). Feinschliff in Plan-Phase.

| Txx | Prio | Aufgabe | Status |
|-----|------|---------|--------|
| T01 | high | Slug-Rename-Pfad: Dateiname-Teil nach `--` ändern/entfernen, `Bean.Slug` setzen. | 🟣 |
| T02 | high | Einzel-ID-Rename: neue ID validieren (Kollision), Datei umbenennen, `# id`-Kommentar, alle Refs kaskadieren. | 🟣 |
| T03 | high | Prefix-Rebrand: über alle beans Prefix tauschen (Suffix behalten), Dateien + Refs + `.beans.yml`. | 🟣 |
| T04 | high | Atomaritäts-Mechanismus (Q02) implementieren. | 🟣 |
| T05 | med | `--dry-run`/Preview-Ausgabe (Q03). | 🟣 |
| T06 | med | Server-/Worktree-Guard (Q01/Q04) — refuse oder koordiniert. | 🟣 |
| T07 | med | CLI-Oberfläche `internal/commands/rename.go` (Cobra), Flags. | 🟣 |
| T08 | med | GraphQL-Mutation(en) falls Q05 → server-route. `internal/graph/schema.graphqls` + codegen. | 🟣 |
| T09 | high | Tests: table-driven Unit (id/slug/prefix), Kaskade-Integrität, dry-run. E2E falls UI berührt. | 🟣 |
| T10 | low | Doku: CLAUDE.md/README Rename-Konvention, LESSONS falls Fallstrick. | 🟣 |
