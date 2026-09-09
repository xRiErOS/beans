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

The styled header carries the whole front matter, not a selection of it: type and id, title, status and priority (plus an inherited status, if one applies), tags, `parent`, `blocking` and `blocked by`, any unknown ("extra") keys such as `branch` or `release` in alphabetical order, and finally the created/updated timestamps and `order`. Tags sit in the header rather than after the body, where a long bean pushed them off the first screen.

Flags: `--raw` forces raw Markdown output even on a terminal; `--json` returns the bean's front matter and body as JSON instead of Markdown; `--body-only` prints only the body content; `--meta` prints only the front matter without the body — the styled header on a terminal, and the source YAML block off one, which still parses as a bean file with an empty body; `--etag-only` prints only the etag, which is useful for detecting whether a bean changed between two reads. `--json`, `--raw`, `--body-only` and `--etag-only` each replace the whole representation, so they exclude one another and they exclude `--meta` and `--table`; `--meta --table` is the one combination that is allowed, because the two describe different things — how much of the bean, and in what arrangement.

`--table` arranges the same front matter as a ruled grid instead of the flowing header. A band across the top carries what the bean is — `id`, `type` and `status` packed left, `priority` against the right edge — and a band across the bottom carries the managed stamps, `created` and `updated` left with `order` on the right. Between them sits one horizontally ruled row per front matter entry, in a fixed order: `title`, `tags`, `parent`, `blocked by`, `blocking`, then any unknown ("extra") keys alphabetically. The bands are what separate identity and bookkeeping from content; in a flat label column the id sat in the same shape as the title.

The column geometry comes from the label and config vocabulary rather than from the bean at hand, so beans rendered in separate calls line up and a reader scans down a column instead of reading every line. In a relation row the related bean's type and title fill the value cell, and its id sits in the label column on the row beneath the label — the label column is where a reader looks for what a row is, and keeping the id out of the value column leaves type and title the full width. An id that cannot be resolved becomes the value itself rather than being dropped. Long free text wraps inside its own cell. Unlike the default view, `--table` forces the grid into a pipe as well — the way `--raw` forces raw markdown onto a terminal — and it combines with `--meta` to give the grid alone.

`--max-width <n>` caps the rendered width at `n` cells, in the grid as well as in the default styled view, and follows the same policy as `beans list`: the flag outranks `display.max_width` in the config, which outranks the built-in default of 110, and `0` disables the cap and renders at the terminal's own width. Two limits apply on top of the number: the rendered width never exceeds the detected terminal width, and it never falls below 80 cells, so a value below 80 renders at 80. Below that floor no layout stays readable, and the floor is shared with `beans list` rather than being a rule of its own here.

```
beans show beans-vvat beans-gng9
```

```
beans show --meta beans-vvat
```

```
beans show --meta --table --max-width 100 beans-vvat beans-gng9
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
