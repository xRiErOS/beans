# Configuration Reference

This page documents every key `.beans.yml` accepts, in the shape the `beans`, `beans-serve`, and `beans-tui` binaries actually read it. It also covers the type/status/priority override system, custom types, display options, and how a beans store's path is resolved.

## Minimal configuration

A repository does not need a `.beans.yml` at all: `beans init` (with no `--profile`) writes one with only the fields that differ from the built-in defaults.

```yaml
# Beans configuration
# See: https://github.com/hmans/beans
project:
    name: myproject
beans:
    path: .beans
    prefix: myproject-
    id_length: 4
    default_status: todo
    default_type: task
```

Every top-level section (`project`, `beans`, `worktree`, `agent`, `server`, `display`) is optional; an absent section, or an absent field within it, falls back to its documented default.

## File location and store path resolution

`beans init` writes `.beans.yml` at the project root, next to the `.beans` data directory it creates. Every command searches upward from the current directory for the nearest `.beans.yml` unless `--config <path>` names one explicitly.

The beans data directory itself is resolved with the following precedence, from highest to lowest:

1. `--beans-path <path>` on the command line.
2. A `.beans.yml` that was actually found: its `beans.path` value, or the default `.beans`, resolved according to `beans.anchor`.
3. `BEANS_PATH`, but only when no `.beans.yml` was found.
4. The default `.beans` directory for the current working directory when neither a config nor environment override exists.

A found `.beans.yml` outranks `BEANS_PATH` on purpose: the config file is a repository's own declaration of where its store lives, while `BEANS_PATH` is commonly exported by a tool like direnv and inherited into unrelated repositories, where honoring it would silently redirect commands to a foreign store. Run `beans path` in any project to print the path a given invocation actually resolved to.

### Anchoring in git worktrees

`beans.anchor` controls what `beans.path` is relative to:

```yaml
beans:
    path: .beans
    anchor: repo-root
```

- Left empty (the default), `beans.path` is relative to the directory containing `.beans.yml`. In a secondary git worktree this means the worktree gets its own store, separate from the main worktree's.
- Set to `repo-root`, `beans.path` resolves against the main worktree's root instead, so every worktree of one repository shares a single store. From the main worktree, or outside a git repository entirely, `repo-root` has no other worktree to redirect to and behaves the same as leaving `anchor` unset.

Any other value fails config loading immediately with an "unknown beans.anchor" error, rather than silently falling back to the default and resolving a different store than the file asked for.

## `project`

| Key | Type | Default |
| --- | --- | --- |
| `name` | string | the project directory's name, set once by `beans init` |

`project.name` is the human-readable project name shown in the web UI and TUI. It has no effect on bean IDs or file paths.

## `beans`

| Key | Type | Default |
| --- | --- | --- |
| `path` | string | `.beans` |
| `anchor` | string | unset (config-file-relative) |
| `prefix` | string | `""` |
| `id_length` | int | `4` |
| `default_status` | string | `todo` |
| `default_type` | string | `task` when no config file exists; otherwise the first merged type when omitted |
| `require_if_match` | bool | `false` |
| `require_fields_on` | map of status name to list of field names | unset |
| `commit_field` | string | `commit` |

`beans.path` is the directory bean files are stored in, resolved as described above.

`beans.prefix` is prepended to every generated bean ID, for example `myproject-abc1`. `beans init` sets it to the project directory's name followed by a hyphen; it can be changed later with `beans rename --prefix`, which also rewrites every existing ID and cascades the change through cross-references (see the command reference in [wiki/commands/organization-and-relations.md](commands/organization-and-relations.md)).

`beans.id_length` is the number of random characters in the generated ID suffix.

`beans.default_status` and `beans.default_type` are applied to `beans create` when `--status`/`--type` are not passed. `beans check` fails if `default_status` names a status the merged status list does not contain, or if `default_type` names an unknown type.

`beans.require_if_match` requires an `If-Match` header (an ETag) on updates made through the API layer, for optimistic concurrency control.

`beans.require_fields_on` declares a completion policy: it maps a target status name to a list of front matter keys that must carry a non-empty value whenever a bean is written into that status.

```yaml
beans:
    require_fields_on:
        completed:
            - commit
```

With the example above, `beans complete <id>` fails with an error naming the missing field and the flag that supplies it (for example `--commit HEAD`), unless the field is already set or supplied in the same call via `--commit` or `--set key=value`. The write is rejected before anything is saved. Config loading itself rejects a `require_fields_on` entry that names an unknown status, an empty field name, or a field beans already manages natively (`title`, `status`, `type`, `priority`, `tags`, `created_at`, `updated_at`, `order`, `parent`, `blocking`, `blocked_by`) — those are written through their own flags, not through `require_fields_on`.

`beans.commit_field` renames the front matter key `require_fields_on` and `beans complete --commit` use to record a git commit SHA. Only this key is git-verified (the value must resolve to a real commit) when written or checked; every other field named in `require_fields_on` only needs to be non-empty.

## `worktree`

| Key | Type | Default |
| --- | --- | --- |
| `base_ref` | string | `main` |
| `path` | string | `~/.beans/worktrees/<project-name>/` |
| `setup` | string | `""` |
| `run` | string | `""` |
| `integrate` | string (`local` or `pr`) | `local` |
| `fetch_timeout` | int (seconds) | `10` |

These settings govern the git worktree management surfaced by the web UI (`beans-serve`). `base_ref` is the git ref new worktree branches start from. `path` is where worktrees are created; it supports a leading `~` for the home directory. `setup` is a shell command run inside a fresh worktree (for example `pnpm install`). `run` is a shell command that, when set, adds a "Run" button to the workspace toolbar. `integrate` chooses the worktree integration strategy: `local` squash-merges locally and hides the PR-related buttons, `pr` pushes the branch and opens a pull request instead. `fetch_timeout` bounds the `git fetch` that refreshes `base_ref` before a new worktree is created; set it to `0` to skip that fetch entirely, for example in an airgapped environment.

## `agent`

| Key | Type | Default |
| --- | --- | --- |
| `enabled` | bool | `true` |
| `default_mode` | string (`act` or `plan`) | `act` |
| `default_effort` | string (`low`, `medium`, `high`, `max`) | unset |

`agent.enabled` toggles whether the web UI exposes agent chats, status panes, and worktree features at all. `agent.default_mode` sets the default permission mode for new agent sessions: `act` runs autonomously, `plan` is read-only. `yolo` is still accepted as a backwards-compatible alias for `act`. `agent.default_effort`, when set, is the thinking-effort level new agent sessions start with; left unset, a new session starts with no effort override and uses the invoking tool's own default. See [wiki/agent-integration.md](agent-integration.md) for the session lifecycle these settings apply to.

## `server`

| Key | Type | Default |
| --- | --- | --- |
| `port` | int | `8080` |
| `cors_origins` | list of strings | `["http://localhost:*", "http://127.0.0.1:*"]` |

Both keys apply to `beans-serve`. `cors_origins` accepts exact origins and port wildcards (`http://localhost:*`); `"*"` allows every origin and is not recommended outside local development. See [wiki/web-ui-and-api.md](web-ui-and-api.md) for the API this server exposes.

## `display`

| Key | Type | Default |
| --- | --- | --- |
| `theme` | string | `mocha` |
| `max_width` | int | `110` |

```yaml
display:
    theme: latte
    max_width: 100
```

`display.theme` selects one of the four bundled Catppuccin flavors: `latte`, `frappe`, `macchiato`, or `mocha`. It governs terminal rendering in the `beans` CLI and `beans-tui`; an unrecognized name is ignored and the default applies (`beans check` flags it as invalid). `display.max_width` caps the rendered width in terminal cells; `0` (unset) yields the default of `110`, and `-1` disables the cap entirely so output uses the full terminal width.

Every `color` field described below (on statuses, types, and priorities) must name one of the Catppuccin tones the active theme defines: `rosewater`, `flamingo`, `pink`, `mauve`, `red`, `maroon`, `peach`, `yellow`, `green`, `teal`, `sky`, `sapphire`, `blue`, `lavender`, `text`, `subtext1`, `subtext0`, `overlay2`, `overlay1`, `overlay0`, `surface2`, `surface1`, `surface0`, or `base`. An empty `color` field is valid and renders with no explicit color.

## Statuses, types, and priorities

Beans ships built-in tables for statuses, types, and priorities. `.beans.yml` can override entries in these tables field by field, or add new entries, through the top-level `statuses`, `types`, and `priorities` lists. A list entry is matched to a built-in entry by `name`; an entry naming an unrecognized name is appended as a new one instead.

### Built-in statuses

| Name | Color | Archive | Short | Description |
| --- | --- | --- | --- | --- |
| `in-progress` | `peach` | no | `I` | Currently being worked on |
| `todo` | `green` | no | `T` | Ready to be worked on |
| `draft` | `overlay2` | no | `D` | Needs refinement before it can be worked on |
| `completed` | `overlay1` | yes | `C` | Finished successfully |
| `scrapped` | `surface2` | yes | `S` | Will not be done |

`short` is the single-character code narrow terminal views and legends render; a status without one renders `?`. An entry in `statuses` overrides `color`, `description`, `archive`, and `short` on the named status, or defines a new status if the name is not one of the five above:

```yaml
statuses:
    - name: completed
      color: green
    - name: blocked
      color: red
      description: Waiting on an external dependency
```

`archive` is only touched by an explicit `archive: true` or `archive: false`; a color- or description-only override on `completed` cannot accidentally flip it back to non-archiving. Archived statuses are excluded from the default working views and are what `beans archive` moves into cold storage.

### Built-in types

| Name | Rank | Short | Emphasis | Description |
| --- | --- | --- | --- | --- |
| `milestone` | 1 | `M` | yes | A target release or checkpoint; group work that should ship together |
| `epic` | 2 | `E` | yes | A thematic container for related work; should have child beans, not be worked on directly |
| `feature` | 3 | `F` | no | A user-facing capability or enhancement |
| `bug` | 4 | `B` | no | Something that is broken and needs fixing |
| `task` | 4 | `T` | no | A concrete piece of work to complete (e.g. a chore, or a sub-task for a feature) |

`rank` carries the parent/child hierarchy: a bean can be the parent of a child bean only when the child's rank is strictly greater than the parent's. Ranks 1 through 3 are container ranks; rank 4 is the leaf rank, and any type without an explicit `rank` (including a newly appended one) lands there. `short` is the single-character code the narrow list view renders; when empty it defaults to the type name's first letter, upper-cased. `emphasis` renders the type bold across the type word, ID, and title in list views — the mechanism that visually distinguishes a container type from a leaf type, since the Catppuccin palette is uniformly pastel and color alone would not separate them. `roadmap`, when explicitly set to `false`, hides a type from its own section in `beans roadmap` and from `beans milestones`; on a container rank this also hides its whole subtree. Leaving `roadmap` unset keeps the type visible.

```yaml
types:
    - name: bug
      color: maroon
      short: B
    - name: chore
      rank: 4
      short: C
      description: Internal work without customer benefit
```

`types_exclusive: true` switches off the merge with the built-in table entirely: `types` then becomes the complete type table on its own, with nothing from the built-in defaults layered underneath it. This is what `beans init --profile <name>` sets, since a profile is meant to give a project exactly its own types. A hand-written `.beans.yml` that never sets `types_exclusive` keeps the default entry-by-entry override behavior, where an override list naming only `bug` leaves the other four built-in types untouched.

```yaml
types_exclusive: true
types:
    - name: task
      short: T
      description: A concrete piece of work to complete
```

See [wiki/project-profiles.md](project-profiles.md) for the bundled profiles (`classic`, `todo`, `simple`, `complex`) `beans init --profile` can expand into a `types_exclusive` type table like this one.

### Built-in priorities

| Name | Color | Symbol | Description |
| --- | --- | --- | --- |
| `critical` | `red` | `‼` | Urgent, blocking work. When possible, address immediately |
| `high` | `yellow` | `!` | Important, should be done before normal work |
| `normal` | `""` | none | Standard priority |
| `low` | `overlay0` | `↓` | Less important, can be delayed |
| `deferred` | `overlay0` | `→` | Explicitly pushed back, avoid doing unless necessary |

Priorities are ordered from highest to lowest urgency, and `symbol` is the compact glyph narrow views and legends render for a priority (a priority without one renders no glyph). A bean file may omit `priority`; the loader then materializes `normal` in memory without rewriting the file. `priorities` overrides follow the same name-matched, field-by-field rule as `statuses` and `types`, including `symbol`.

```yaml
priorities:
    - name: critical
      color: pink
```

## Validation

`beans check` validates a project's configuration alongside its bean files: that the merged status and type tables are well-formed, that `default_status` and `default_type` name entries that actually exist, that every bean carries a known type, that every configured `color` names a valid theme tone, that `beans.prefix` matches the IDs actually on disk, and that any `require_fields_on` policy is satisfiable. A misconfigured `.beans.yml` — an unknown `beans.anchor`, an invalid `require_fields_on` entry, or a config file that fails to parse — is rejected at load time, before any command runs, rather than surfacing as a downstream validation warning. See [wiki/commands/validation-and-maintenance.md](commands/validation-and-maintenance.md) for the full `beans check` output.

## Related documentation

- [wiki/project-profiles.md](project-profiles.md)
- [wiki/commands/project-setup.md](commands/project-setup.md)
- [wiki/commands/organization-and-relations.md](commands/organization-and-relations.md)
- [wiki/commands/validation-and-maintenance.md](commands/validation-and-maintenance.md)
- [wiki/agent-integration.md](agent-integration.md)
- [wiki/web-ui-and-api.md](web-ui-and-api.md)
- [wiki/commands/separate-binaries.md](commands/separate-binaries.md)
- [wiki/tui-companion.md](tui-companion.md)
- [wiki/fork-lineage.md](fork-lineage.md)
