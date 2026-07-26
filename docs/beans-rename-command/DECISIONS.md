# DECISIONS — beans-rename-command

Think-Phase (TPIC). Codes fix, scope-lokal. Status: 🟣 Offen · 🔵 In Arbeit · 🟢 Done · 🔴 Abgelehnt · 🟡 Unklar.

| Dxx | Hintergrund | Entscheidung | Status |
|-----|-------------|--------------|--------|
| D01 | `rename` könnte nur Slug ODER nur ID meinen — technisch grundverschieden (Slug frei, ID kaskadiert). | Command deckt **beides** ab: Slug-Rename UND ID/Prefix-Rename. | 🟢 |
| D02 | ID-Rename kann einzelne bean oder projekt-weiten Prefix meinen. | **Beide Ebenen**: `--prefix <neu>` (projekt-weit, re-IDt alle) UND `<id> <neu-id>` (Einzel-bean). | 🟢 |
| D03 | Kaskade-Umfang bei ID-Änderung. | Beide ID-Modi kaskadieren `parent`/`blocking`/`blocked_by` aller referenzierenden beans + Dateiname + `# <id>`-Kommentarzeile. | 🟢 |
| D04 | Exec-Arch: Server-State/Watcher vs. Massen-Migration vs. "CLI-via-GraphQL"-Konvention. | **Hybrid**: Slug- + Einzel-ID-Rename = GraphQL-Mutation über laufenden Server (klein/live/konventionskonform). Prefix-Rebrand = OFFLINE-Batch-Command, verweigert bei laufendem Server, schreibt atomar. Trennung nach Blast-Radius. (löst Q01/Q05) | 🟢 |
| D05 | Aktive Worktrees haben eigene `.beans/` mit alten IDs → Divergenz beim Merge. | Prefix-Rebrand **verweigert bei aktiven Worktrees**. Nutzer muss erst mergen/aufräumen. Guard prüft `~/.beans/worktrees/<proj>/`. (löst Q04) | 🟢 |
| D06 | Atomarität Massen-Rebrand (Q02). | Offline-Rebrand: alle beans in Memory laden, vollständige new-ID-Map + Ref-Rewrite + neue Filenames berechnen, in **Temp-Staging** schreiben, dann atomarer Swap (Backup alt). Fehler vor Swap → Abbruch, nichts berührt. Live-Mutation: bestehender transaktionaler Schreibpfad + etag. | 🟢 |
| D07 | Blast-Radius-Schutz (Q03). | `--dry-run` auf allen Modi. Prefix-Rebrand: Summary + Bestätigung (`--yes`) Pflicht vor Ausführung. | 🟢 |
| D08 | Slug-Quelle (Q06). | `--slug <x>` explizit · `--no-slug` leert · `--reslug` regeneriert aus Title (`Slugify`). | 🟢 |
| D09 | Einzel-ID-Form (Q07). | `beans rename <id> <neu-id>` volle neue ID, Kollisions-Check; `--suffix <x>` behält Prefix. | 🟢 |
| D10 | git-Handling (Q08). | Plain FS-Rename, KEIN auto `git mv`/commit (beans committet nie selbst; Nutzer staged bean-Files). git erkennt Renames per Content. | 🟢 |
| D11 | Externe Referenzen (Q09). | IDs in Commit-Messages/docs/SSTD out of scope — nicht rewriten, nur dokumentieren. | 🟢 |
| D12 | Config-Konsistenz (Q10). | Prefix-Rebrand schreibt neuen `prefix:` in `.beans.yml`. | 🟢 |

## Kontext (Codebase-Grounding)

- ID = `config.Beans.Prefix` + nanoid(`id_length`, default 4, Alphabet `0-9a-z`). `pkg/bean/id.go:49` `NewID`.
- ID wird bei jedem Laden **aus Dateiname rekonstruiert** (Teil vor `--`). `pkg/beancore/core.go:203` `loadBean`, `pkg/bean/id.go:67` `ParseFilename`.
- Dateiname = `BuildFilename(id, slug)` → `id--slug.md`. `pkg/bean/id.go:92`.
- Slug: separates Feld `Bean.Slug`, `yaml:"-"`, aus `Slugify(title)` bei Create. NIE in Cross-Refs. `pkg/bean/id.go:100`.
- Cross-Refs speichern **volle ID**: `parent`/`blocking`/`blocked_by`. `pkg/bean/bean.go:159-166`.
- `# <id>`-Zeile in Frontmatter = Kommentar, Parse ignoriert. `pkg/bean/bean.go:249`.
- Kein bestehender Rename-Mechanismus (CLI/Mutation/Func).
- Config: `BeansConfig.Prefix` yaml `prefix`, `IDLength` yaml `id_length`. `pkg/config/config.go:177`.
- Referenz-Projekt `bew_BeWiki-Python-Download`: 233 beans, 216 mit Cross-Refs, Prefix `bew_BeWiki-Python-Download-`.
