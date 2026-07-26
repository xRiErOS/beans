# QUESTIONS — beans-rename-command

Alle offenen Fragen aufgelöst (Konvergenz erreicht). Auflösung → DECISIONS.md.

| Qxx | Frage | Auflösung | Status |
|-----|-------|-----------|--------|
| Q01 | Server gestoppt oder Live-Mutation? | Hybrid — siehe D04. | 🟢 |
| Q02 | Wie atomar? | Temp-Staging + Swap (offline) / etag (live) — D06. | 🟢 |
| Q03 | dry-run/Preview Pflicht? | `--dry-run` überall, Rebrand + Confirm — D07. | 🟢 |
| Q04 | Verhalten bei aktiven Worktrees? | Refuse — D05. | 🟢 |
| Q05 | GraphQL vs direkter Disk-Op? | Split nach Blast-Radius — D04. | 🟢 |
| Q06 | Slug-Quelle? | `--slug`/`--no-slug`/`--reslug` — D08. | 🟢 |
| Q07 | Einzel-ID Form? | volle neue ID + `--suffix`, Kollisions-Check — D09. | 🟢 |
| Q08 | git mv oder plain? | Plain FS-Rename, kein auto-commit — D10. | 🟢 |
| Q09 | Externe Refs? | Out of scope, dokumentieren — D11. | 🟢 |
| Q10 | `.beans.yml` mitziehen? | Ja, `prefix:` schreiben — D12. | 🟢 |
