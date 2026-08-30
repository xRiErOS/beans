# Planning and reporting

This page documents the read-only commands that summarize a project's structure and status: roadmap views, milestone rollups, aggregate progress, and relationship graphs.

## `beans roadmap`

`beans roadmap [id]` displays a roadmap of milestones, epics, and their child items, walking the parent/child hierarchy rather than any single flat list. With no ID argument it renders the entire roadmap, rooted at every top-rank container (e.g. milestones). With an ID argument naming a milestone, epic, or feature, it scopes the output to that item's subtree only; `--status` and `--no-status` cannot be combined with an ID argument, since scoping to one subtree and filtering the top-level milestone list are mutually exclusive concerns.

`--depth <n>` limits how many levels below the roadmap's root are rendered, following the `tree -L n` convention where the root itself never counts. Without an ID argument the root is the roadmap as a whole, so `--depth 1` lists milestones only; with an ID argument the root is that item, so `--depth 1` lists its direct children. `--status <name>` (repeatable) and `--no-status <name>` (repeatable) filter which milestones appear by status, and `--include-done` includes completed items that are otherwise omitted.

`--view` chooses the layout used for terminal output: `tree` (default) nests items under their containers, `table` lists them flat and sortable. The `--view` choice only affects terminal rendering — Markdown output always uses the fixed Markdown template regardless of `--view`.

`--format` chooses the output mode explicitly: `tty` for the colored terminal tree/table, or `markdown` for a plain Markdown document with headings per milestone/epic and a list per leaf item. Left unset, the format is auto-detected from whether stdout is a terminal. In Markdown mode, `--no-links` renders bean IDs as plain text instead of Markdown links, `--link-prefix <url>` sets their URL prefix, and `--tags` includes tags. `--max-width <n>` caps terminal output. `--json` instead returns the roadmap domain model as structured data and bypasses terminal/Markdown rendering.

```
beans roadmap
beans roadmap --view table --format tty
beans roadmap beans-xkih --depth 1
beans roadmap --format markdown --tags --no-links > ROADMAP.md
beans roadmap --status in-progress --no-status scrapped
```

## `beans milestones`

`beans milestones` lists every bean on the top container rank (e.g. milestones), each annotated with how many of its descendants — through any number of parent levels, such as an epic's tasks — are completed. Completed and scrapped milestones are hidden by default; `--all` includes them. Descendants hidden by a status/archive policy do not contribute to the completed/total counts, so a hidden subtree under a visible milestone cannot inflate or deflate its progress figure.

`--view` chooses the arrangement: `table` (default) lists milestones flat and sortable, `tree` nests them. `--tags` renders each milestone's tags. `--max-width <n>` caps the rendered width (`0` disables the cap; otherwise falls back to the `display.max_width` config value, or 110).

```
beans milestones
beans milestones --all --view tree --tags
```

## `beans progress`

`beans progress` shows counts by status across every configured status, plus a percent-complete figure computed as `completed / (total - scrapped)`, truncated toward zero. `--parent <id>` scopes the counts to a single bean's descendants (for example a milestone or epic) instead of the whole workspace. `--json` returns the per-status counts plus the derived `completed`, `total`, and `percent` fields; the plain-text form additionally renders a fixed-width bar under the counts.

```
beans progress
beans progress --parent beans-xkih
beans progress --json
```


## `beans graph`

`beans graph [id]` prints parent and blocking relationships. The default Graphviz DOT output can be piped into tools such as `dot -Tpng`; `--format ascii` prints a terminal edge list, and `--format json` returns `nodes` and `edges`.

Without an ID the command includes the complete store. Naming one bean scopes the graph to its neighborhood: `--depth 1` includes its direct relationships, larger values widen the traversal by hops, and `--depth 0` walks its whole connected component. `--relation parent` and `--relation blocks` can be repeated to restrict edge kinds. Broken links and self-links are omitted here and reported by `beans check`.

```
beans graph
beans graph beans-xkih --format ascii
beans graph beans-xkih --depth 2 --relation parent
beans graph --format json
```

## Related documentation

- [Organization and relations](organization-and-relations.md)
- [Querying and automation](querying-and-automation.md)
