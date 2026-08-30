# Bean Lifecycle

This page covers the commands that create beans and move them through their statuses: `create`, `start`, `complete`, `scrap`, `archive`, and `delete`. Read it alongside the destructive-versus-non-destructive distinction called out at the end, since `delete` is the only command here that removes data permanently.

## `beans create` (aliases `c`, `new`)

`beans create [title]` creates a new bean with a generated ID and an optional title. Flags: `-t`/`--type string` sets the bean type (`milestone`, `epic`, `feature`, `bug`, `task` in the default profile; the actual set depends on the project's configured type profile, see `../project-profiles.md`); `-s`/`--status string` sets the initial status (`in-progress`, `todo`, `draft`, `completed`, `scrapped`); `-p`/`--priority string` sets the priority (`critical`, `high`, `normal`, `low`, `deferred`); `-d`/`--body string` sets body content (`-` reads it from stdin), or `--body-file` reads it from a file; `--parent string` sets a parent bean ID; `--blocked-by`/`--blocking stringArray` record blocking relationships and can be repeated; `--tag stringArray` adds tags and can be repeated; `--order string` sets an explicit fractional-index order value; `--prefix string` overrides the configured ID prefix for this bean; `--set`/`--unset stringArray` add or remove extra front matter `key=value` pairs and can be repeated; `--json` returns the created bean as JSON.

A bean created without `-s` gets whatever status the project's profile defines as the default, which is not necessarily `todo` — check `beans list` after creating one if you expect it to show up under `beans next`.

```
beans --beans-path ./demo/.beans create "Fix login bug" -t bug -p high -s todo
```

```
beans --beans-path ./demo/.beans create "Design auth flow" -t feature --parent beans-vvat --tag auth
```

## `beans start`

`beans start <id> [id...]` marks one or more existing beans as in-progress. A single bean is displayed in full, the same as `beans show` would; several beans each get one confirmation line instead, since printing many full bodies on one screen helps nobody. Every ID in the call is resolved before the first bean is written, so an unknown ID leaves the whole batch untouched. Flags: `--json` returns the result as JSON.

```
beans --beans-path ./demo/.beans start beans-vvat
```

## `beans complete`

`beans complete <id> [id...]` marks one or more existing beans as completed. `--summary`, `--commit`, and `--set` apply to every bean named in the call. Flags: `--summary string` appends an optional summary of changes to each bean's body; `--commit string` records a git ref (`HEAD`, a branch, a tag, or a SHA) in the configured commit field of each bean; `--set stringArray` sets an extra front matter `key=value` on every bean in the call and can be repeated; `--json` returns the result as JSON. Every ID is resolved and checked against the status policy before the first bean is written, so an unknown ID or a policy violation leaves the whole batch alone.

```
beans --beans-path ./demo/.beans complete beans-vvat --summary "Fixed login" --commit HEAD
```

## `beans scrap`

`beans scrap <id> [id...]` marks one or more existing beans as scrapped instead of completed, for work that will not be finished. `--reason` applies to every bean named in the call and is required. Flags: `--reason string` records why the bean was scrapped (required); `--json` returns the result as JSON. Every ID is resolved before the first bean is written, so an unknown ID leaves the whole batch alone.

```
beans --beans-path ./demo/.beans scrap beans-gng9 --reason "Out of scope"
```

## `beans archive`

`beans archive` moves every bean whose status is configured for archiving (`completed` and `scrapped` by default) into `.beans/archive/`. Archived beans keep their relationships intact and remain visible in `beans list` and other queries; the command relocates files to keep the active directory tidy without deleting project history. It takes no ID arguments and processes every qualifying bean. `--json` returns the result as JSON.

```
beans --beans-path ./demo/.beans archive
```

## `beans delete` (alias `rm`)

`beans delete <id> [id...]` permanently removes one or more beans from the store, unlike `archive`, `complete`, and `scrap`, which all preserve the bean file. By default it asks for confirmation; if other beans reference the target(s) as a parent or via blocking, it also warns about those references and removes them after confirmation. Flags: `-f`/`--force` skips both the confirmation prompt and the reference warnings; `--json` outputs the result as JSON and implies `--force`, so a scripted `--json` call never blocks on a prompt.

Because `--json` implies `--force`, a `beans delete --json <id>` call in a script will silently drop any references to that bean from other beans' front matter — confirm the ID is the one you mean before scripting this call.

```
beans --beans-path ./demo/.beans delete beans-u76f --json
```

## Related documentation

- [Project Setup and Introspection](project-setup.md)
- [Inspection and Search](inspection-and-search.md)
- [Organization and Relations](organization-and-relations.md)
- [Planning and Reporting](planning-and-reporting.md)
- [Validation and Maintenance](validation-and-maintenance.md)
- [Data Model](../data-model.md)
- [Project Profiles](../project-profiles.md)
