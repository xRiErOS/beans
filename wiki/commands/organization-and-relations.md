# Organization and relations

This page documents the commands that reshape existing beans: editing fields and body text, tagging, ordering siblings, and renaming IDs or slugs. It complements the creation and lifecycle commands documented elsewhere in this wiki.

## `beans update`

`beans update <id>` (alias `u`) changes one or more properties of a single existing bean in one write. It requires at least one change-producing flag; running it with none returns a validation error listing the accepted flags.

Field flags are `--status`, `--type`, `--priority`, `--title`, each validated against the project's configured statuses, types, and priorities before anything is written. `-p`/`--priority` accepts an empty string to clear the priority.

Body flags come in two mutually exclusive groups. The first group replaces the whole body: `-d`/`--body` (use `-` to read from stdin) or `--body-file` to read from a file. The second group edits the existing body in place: `--body-replace-old` together with `--body-replace-new` (both required together) finds and replaces one span of text, and `--body-append` (use `-` for stdin) appends text to the end. `--body-replace-old` and `--body-append` can be combined in the same call and both apply; neither can be combined with `--body`/`--body-file`.

Relationships use paired add/remove flags rather than full replacement: `--parent` sets a parent bean ID and `--remove-parent` clears it (mutually exclusive with each other); `--blocking`/`--remove-blocking` and `--blocked-by`/`--remove-blocked-by` each accept a repeatable bean ID to add or remove one relationship at a time. Tags follow the same pattern: `--tag`/`--remove-tag` (repeatable).

`--set key=value` (repeatable) writes an arbitrary extra front matter key, and `--unset key` (repeatable) removes one; both count as a change even though they bypass the typed fields above. `--if-match <etag>` makes the update conditional: it fails instead of writing if the bean's current etag does not match, which is the command's only optimistic-locking guard — `update` does not otherwise detect concurrent edits.

All of a call's field changes, body changes, relationship changes, and extra-key operations land in a single write under one etag, so a status change and any extra front matter fields required by policy land together rather than being split across two calls. `--json` returns the resulting bean in the same shape as `beans show --json`, not a wrapped success envelope, so a caller can verify the write with one command. Updating a bean whose file is under `archive/` changes it in place; it does not move the file back to the active directory.

```
beans update beans-w1z6 --status in-progress --tag backend
beans update beans-w1z6 --body-replace-old "Initial" --body-replace-new "Updated" --body-append "

More context."
beans update beans-w1z6 --blocked-by beans-xkih --parent beans-e1ta
```

## `beans tag`

`beans tag <id> [id...]` adds and removes tags on one or more beans in a single invocation. `--tag` (repeatable) adds tags, `--remove-tag` (repeatable) removes them; any tag not named by either flag is left untouched, and at least one of the two flags is required. Every given ID is resolved before the first bean is written, so a typo'd or missing ID among several fails the whole call before any file changes — but once writing starts, each bean is written independently, so a failure partway through (for example an etag conflict on a later ID) leaves the beans already written in their new tagged state.

```
beans tag beans-w1z6 beans-e1ta --tag backend --tag urgent
beans tag beans-w1z6 --remove-tag urgent
```

## `beans order`

`beans order <id>` sets a bean's manual order key relative to its siblings under the same parent, using a fractional index. A move only ever writes the one bean file being placed, never the neighboring siblings, and ordering is scoped strictly per parent: `--after`/`--before` must name a sibling that shares the target bean's parent.

Exactly one placement flag is required, and the four are mutually exclusive: `--first` (before all siblings), `--last` (after all siblings), `--after <id>` (immediately after that sibling), or `--before <id>` (immediately before that sibling). `--after`/`--before` require the named sibling to already carry an explicit order value — placing relative to a sibling that has never been ordered fails with a message suggesting `--first`/`--last` on that sibling first. This means a sibling group typically needs one bean placed with `--first` or `--last` before `--after`/`--before` can be used to interleave the rest.

```
beans order beans-w1z6 --last
beans order beans-e1ta --before beans-w1z6
```

## `beans rename`

`beans rename` renames beans in exactly one of three modes, chosen by which flags and positional arguments are given; supplying flags for more than one mode is rejected.

Slug mode changes only the filename's human-readable slug, leaving the bean's ID untouched: `beans rename <id> --slug "new-slug"` sets it explicitly, `beans rename <id> --no-slug` clears it so the filename becomes `<id>.md`, and `beans rename <id> --reslug` regenerates the slug from the bean's current title.

ID mode changes the bean's ID itself and cascades: `beans rename <id> <new-id>` gives the bean a full new ID, and `beans rename <id> --suffix <new-suffix>` keeps the project's configured prefix and only replaces the suffix (it refuses to run against an ID that does not already start with that prefix). Both forms update every reference to the old ID found elsewhere in the store (parent links, blocking/blocked-by lists, and body text) as part of the same rename.

Prefix mode, `beans rename --prefix "new-prefix-"`, rebrands every bean ID in the project to the new prefix in one pass and updates every cross-reference accordingly. It additionally requires `--yes` or an interactive y/N confirmation before applying, and refuses to run while a `beans-serve` process is active or while active git worktrees exist for the project.

`--dry-run` prints the planned changes for any of the three modes without writing anything; combine it with `--json` to get the plan as structured data. Without `--dry-run`, a successful rename reports how many beans changed.

```
beans rename beans-w1z6 --slug "endpoint-handler"
beans rename beans-w1z6 --reslug
beans rename beans-w1z6 bew-1234
beans rename beans-w1z6 --suffix k7x2
beans rename --prefix "bew-" --dry-run
```

## Related documentation

- [Planning and reporting](planning-and-reporting.md)
