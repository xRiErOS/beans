# Querying and Automation

This page documents `beans graphql`, the CLI's structured-query interface, and the JSON output/error envelope that every scriptable `beans` command shares. Read it before wiring `beans` into a script, an agent tool call, or a CI step.

## `beans graphql`

`beans graphql <query>` (alias: `beans query`) executes a GraphQL query or mutation directly against the local bean store, without starting the `beans-serve` web server. It runs the same resolver and schema that `beans-serve` exposes over HTTP, so a query written against one works unmodified against the other for read-only bean data (server-only fields such as worktrees or agent sessions are not backed by anything when there is no server process).

The query is the command's single positional argument. When no argument is given, `beans graphql` reads the query from stdin instead, which avoids shell-quoting problems for multi-line documents:

```
echo '{ beans { id title } }' | beans graphql
cat query.graphql | beans graphql
```

Flags:

| Flag | Description |
|---|---|
| `-v, --variables string` | Query variables as a JSON string |
| `-o, --operation string` | Operation name, for a document with more than one named operation |
| `--json` | Output plain JSON without ANSI colors (for piping into `jq` or a file) |
| `--schema` | Print the full GraphQL schema (SDL) and exit, ignoring the query argument |

Output is always the `data` portion of the GraphQL response, pretty-printed; `--json` strips the color codes that the default terminal-facing mode adds. A query or execution error is reported on stderr and the command exits non-zero — no partial `data` is printed on error.

Examples, verified against a throwaway store:

```
beans graphql '{ beans { id title status } }'
beans graphql '{ bean(id: "abc") { title status body } }'
beans graphql '{ beans(filter: { status: ["todo", "in-progress"] }) { id title } }'
beans graphql '{ beans { id title blockedBy { id title } children { id title } } }'
beans graphql -v '{"id": "abc"}' 'query GetBean($id: ID!) { bean(id: $id) { title } }'
beans graphql --schema
```

A malformed query fails before any resolver runs:

```
$ beans graphql '{ nope }'
Error: graphql: Cannot query field "nope" on type "Query".
Run 'beans graphql --help' for usage.
```

The schema itself — every `Query`, `Mutation`, and `Subscription` field, with its own doc comments — is the authoritative reference for what can be queried; `beans graphql --schema` always reflects the binary actually running. The subset of the schema backed by `beans` alone (no server process) covers bean data: `bean(id)` and `beans(filter)`, plus their relationship fields (`blockedBy`, `blocking`, `children`, `parent`, and similar). Fields describing worktrees, agent sessions, terminals, and file diffs are resolved by [`beans-serve`](../web-ui-and-api.md) and return empty or zero values when queried through the plain `beans` binary.

## JSON output contract

Successful JSON payloads are command-specific. `list`, `show`, `progress`, and `roadmap` return their domain model directly; `check` returns its own `success` plus diagnostic fields; `update --json` returns the updated bean directly; and rename dry runs return a rename plan. Mutating lifecycle commands such as `create`, `start`, `complete`, `scrap`, `archive`, `delete`, `tag`, and `order` use a success envelope with command-appropriate `bean`, `beans`, `message`, or `warnings` fields.

For example, a lifecycle mutation can return:

```json
{
  "success": true,
  "bean": { "...": "the affected bean" }
}
```

A failed result:

```json
{
  "success": false,
  "error": "human-readable message",
  "code": "MACHINE_READABLE_CODE"
}
```

`code` is one of a fixed set: `NOT_FOUND`, `NO_BEANS_DIR`, `INVALID_STATUS`, `FILE_ERROR`, `VALIDATION_ERROR`, `CONFLICT`, `POLICY_VIOLATION`. Verified example, a lookup against an ID that does not exist:

```
$ beans show doesnotexist --json
{
  "success": false,
  "error": "bean not found: doesnotexist",
  "code": "NOT_FOUND"
}
```

## Error reporting outside `--json`

Without `--json`, a failing command prints exactly two lines to stderr — the error, and a pointer to `--help` — and exits with status 1:

```
$ beans show doesnotexist
Error: bean not found: doesnotexist
Run 'beans show --help' for usage.
```

This replaces cobra's default behavior of printing the error followed by the entire flag usage block; the two-line shape is identical for a runtime failure (a missing bean) and an invocation failure (an unparsable flag), so scripts and human readers see one consistent failure format regardless of cause.

The two output modes never overlap: when a command has already written a JSON error document to stdout, the two-line stderr report is suppressed, so a `--json` caller gets exactly one artifact. An error raised before a command reaches its JSON-emitting code path — an unreadable config file, a missing `.beans` directory — still reaches the user on stderr even when `--json` was requested, because no document was ever emitted to suppress it.

## Related documentation

- [Validation and Maintenance](validation-and-maintenance.md)
- [Web UI and API](../web-ui-and-api.md)
- [Separate Binaries](separate-binaries.md)
- [Configuration Reference](../configuration.md)
