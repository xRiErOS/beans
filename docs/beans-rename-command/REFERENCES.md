# REFERENCES — beans-rename-command

Code-Grounding (file:line) und externe Bezüge.

## ID / Slug / Filename
- `pkg/bean/id.go:49` `NewID(prefix, length)` — Generator, `prefix + nanoid`.
- `pkg/bean/id.go:100` `Slugify(title)` — lowercase, `-`-Norm, 50-Zeichen-Trunc.
- `pkg/bean/id.go:92` `BuildFilename(id, slug)` → `id--slug.md` (bzw. `id.md` ohne Slug).
- `pkg/bean/id.go:67` `ParseFilename` — inverse; Formate `id--slug`, `id.slug`, `id-slug` (legacy), `id`.
- `pkg/bean/id.go:17` `containsBlockedWord` / Blocklist.

## Modell / Storage
- `pkg/bean/bean.go:137` `Bean.ID`/`Bean.Slug` `yaml:"-"` (nicht echte Frontmatter-Keys).
- `pkg/bean/bean.go:159-166` `Parent`/`Blocking`/`BlockedBy` — volle ID-Strings.
- `pkg/bean/bean.go:168-192` `frontMatter`/`renderFrontMatter` — yaml `parent`/`blocking`/`blocked_by`.
- `pkg/bean/bean.go:249` `Render()` schreibt `# <id>`-Kommentarzeile.
- `pkg/bean/bean.go:183` `Parse()` — ID aus Frontmatter NICHT gelesen (nur Filename).

## Core / IO
- `pkg/beancore/core.go:499` `Core.Create` — `NewID` mit Config-Prefix.
- `pkg/beancore/core.go:203` `loadBean` — `ParseFilename`.
- `pkg/beancore/core.go:628` `saveToDisk` — nutzt `b.Path` wenn gesetzt (recomputet Filename NICHT für Bestand).
- `pkg/beancore/core.go:333` `NormalizeID` — expandiert Kurz-ID via Prefix, kennt keinen Slug.
- `pkg/beancore/watcher.go:227` / `worktree_watcher.go:208` — Rename-Events des FS-Watchers.

## GraphQL
- `internal/graph/schema.graphqls:206-352` — Mutationsliste (kein rename/slug).
- `pkg/beangraph/mutations.go:15` `CreateBean` (`Slugify` bei Create), `:96` `UpdateBean` (fasst id/slug nicht an).

## Config
- `pkg/config/config.go:177` `BeansConfig.Prefix` (yaml `prefix`), `:178` `IDLength` (yaml `id_length`).
- `pkg/config/config.go:396` HeadComment "Prefix for bean IDs".

## Worktree-Architektur (CLAUDE.md)
- Worktrees außerhalb Repo in `~/.beans/worktrees/<project>/`; eigene lokale `.beans/`.
- `beans serve` = autoritativer Runtime-State, merged Worktree-Änderungen als "dirty".

## Referenz-Projekt (Motivation)
- `~/dev/bew_BeWiki-Python-Download/` — Prefix `bew_BeWiki-Python-Download-`, 233 beans, 216 Cross-Refs. Motiviert Prefix-Kürzung → `bew-`.
