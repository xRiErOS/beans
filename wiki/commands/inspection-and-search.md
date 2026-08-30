# Inspection and Search

This page covers the commands that read beans back out of the store: `list`, `show`, and `next`. Use it to find, filter, and display existing beans without changing them.

## `beans list` (alias `ls`)

`beans list` lists every bean in the `.beans` directory as a table, sorted by status, priority, type, and title unless `--sort` says otherwise. Pass `--view tree` to render the same beans nested under their parents instead of flat, `--json` to return matching bean metadata as a JSON array, `--full` to include each body in that JSON output, and `--quiet`/`-q` to print just one ID per line for piping into other commands.

Filtering flags narrow the result set and combine with AND logic across flag kinds: `-s`/`--status`, `-t`/`--type`, `-p`/`--priority`, and `--tag` each accept repeated values with OR logic within the same flag; `--no-status`, `--no-type`, `--no-priority`, and `--no-tag` exclude by the same criteria; `--parent string` filters by parent ID; `--has-parent`/`--no-parent` filter on whether a parent is set at all; `--has-blocking`/`--no-blocking` filter on whether the bean blocks others; `--is-blocked`/`--unblocked` filter on whether the bean itself is blocked; `--where stringArray` filters on arbitrary extra front matter `key=value` pairs with AND logic; `-S`/`--search string` runs a full-text query against title and body.

`--ready` filters to beans available to start: not blocked, and excluding beans that are already in-progress, completed, scrapped, or draft. This is the same predicate `beans next` uses to pick a single bean, so `beans list --ready` is the way to see the whole queue rather than just its head.

The `-S`/`--search` flag uses Bleve query string syntax: `login` is an exact term match, `login~` is a fuzzy match at one edit distance, `login~2` widens that to two edit distances, `log*` is a wildcard prefix match, `"user login"` is an exact phrase match, `user AND login` and `user OR login` combine terms, and `slug:auth`, `title:login`, `body:auth` restrict the match to a single field.

Two more flags control terminal rendering rather than filtering: `--tags` renders each bean's tags in the table, and `--max-width int` caps the rendered width (0 disables the cap; the default comes from `display.max_width` in config, else 110).

```
beans list --ready --type bug --tag cli
```

```
beans list --json --status in-progress --sort updated --desc
```

## `beans show`

`beans show <id> [id...]` displays the full contents of one or more beans, including front matter and body, and accepts multiple IDs in a single call. Output follows stdout: on a terminal it is styled and the body is rendered as markdown, while piped or redirected output falls back to the raw markdown of the source file, unpadded and unwrapped, so downstream parsers get exactly the file content.

Flags: `--raw` forces raw Markdown output even on a terminal; `--json` returns the bean's front matter and body as JSON instead of Markdown; `--body-only` prints only the body content; `--etag-only` prints only the etag, which is useful for detecting whether a bean changed between two reads. These four output-mode flags are mutually exclusive.

```
beans show beans-vvat beans-gng9
```

```
beans show --etag-only beans-vvat
```

## `beans next`

`beans next` finds the single highest-priority bean available to start — not blocked, and excluding in-progress, completed, scrapped, and draft beans — and displays it the same way `beans show` would. If nothing qualifies, it reports that no ready beans were found rather than printing an empty bean.

`--type`, `--tag`, `--parent`, and `--sort` mean the same as in `beans list`, so a narrowed query moves between the two commands unchanged; `--desc` reverses the sort order, and `--json` returns the bean as JSON instead of the rendered view. Unlike `list`, `next` has no `--status`, `--priority`, or search flags, because its readiness predicate already fixes which statuses are eligible.

```
beans next --type bug --tag cli
```

```
beans next --parent beans-vvat --sort order
```

## Related documentation

- [Project Setup and Introspection](project-setup.md)
- [Lifecycle](lifecycle.md)
- [Organization and Relations](organization-and-relations.md)
- [Planning and Reporting](planning-and-reporting.md)
- [Querying and Automation](querying-and-automation.md)
- [Data Model](../data-model.md)
- [Configuration](../configuration.md)
