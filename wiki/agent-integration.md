# Agent Integration

This page covers how to wire a coding agent up to `beans` — the generic instruction, the two integrations `beans` currently ships support for, and the safe read/write patterns an agent should follow once it's connected. It supplements the exact CLI syntax documented per-command elsewhere in this wiki.

## The one rule that matters

Never let an agent rely on stale or remembered knowledge of a project's beans setup. Every project can configure its own types, statuses, priorities, required-fields policy, and commit-field behavior in `.beans.yml`, and `beans prime` is the single command that reports what is actually configured in *this* project, right now:

```bash
beans prime
```

`beans prime` renders a full agent-facing usage guide, generated from a template filled in with the running project's real configuration (its types, statuses, priorities, hierarchy ranks, and any `require_fields_on` policy) — it is not static prose. An agent should treat `beans prime`'s output as authoritative over any generic knowledge about beans from training data or a previous session, because the previous project it worked in may have had different types, a different commit-field policy, or no policy at all. When no `.beans.yml` can be found from the current directory, `beans prime` silently exits with no output, which is the intended way to detect "this isn't a beans project" without an error.

The minimal, tool-agnostic way to get an agent to run it is one line in the project's `AGENTS.md`, `CLAUDE.md`, or equivalent instruction file:

```
**IMPORTANT**: before you do anything else, run the `beans prime` command and heed its output.
```

## Claude Code

Claude Code's `SessionStart` and `PreCompact` hooks can invoke `beans prime` automatically so the agent gets current project context at the start of a session and again after context compaction, add this to the project's `.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [
      { "hooks": [{ "type": "command", "command": "beans prime" }] }
    ],
    "PreCompact": [
      { "hooks": [{ "type": "command", "command": "beans prime" }] }
    ]
  }
}
```

A packaged Claude Code plugin for this exists at `extras/claude/plugins/beans-prime/` (the same hook configuration, plus a plugin manifest) so a project doesn't have to hand-edit `.claude/settings.json`. As of this fork's current state that plugin is explicitly a work in progress and not ready for general use — its own README says so — so the manual hook configuration above is the supported path until it lands.

## OpenCode

OpenCode integrates through a plugin that injects `beans prime`'s output into the chat system prompt on session start and again on session compaction. The plugin ships in-repo at `.opencode/plugin/beans-prime.ts`; to use it, copy that file into a project's `.opencode/plugin/` directory (or `~/.opencode/plugin/` for a personal, cross-project install), no build step required. The plugin only injects context when both `beans` is on `PATH` and a `.beans.yml` exists in the target directory — otherwise it silently does nothing, mirroring `beans prime`'s own silent no-op.

## Agents without a supported hook

Any agent that can run a shell command and read `AGENTS.md`/`CLAUDE.md` can use `beans` even without a dedicated hook or plugin: put the one-line instruction from above in that file, and the agent will run `beans prime` itself at the start of its work. This is the same fallback the Claude Code and OpenCode integrations reduce to when the hook or plugin isn't installed — nothing about `beans prime`'s output depends on which integration triggered it.

## Read workflows

Once primed, an agent's normal work loop is: find work, show it, act on it.

```bash
beans next                              # single highest-priority ready bean
beans next --type bug --tag cli         # narrowed the same way list is narrowed
beans list --json --ready               # every ready bean, not just the top one
beans list --json --is-blocked          # beans blocked directly or via a blocked ancestor
beans show --json <id> [id...]          # full details, one or many IDs
beans progress                          # status counts + percent-complete across the workspace
beans progress <id>                     # same, scoped to one bean's descendants
beans milestones                        # milestones with descendant completion counts
```

`beans list -S "<keywords>"` (full-text search over title/slug/body, with Bleve query syntax: fuzzy `~`, wildcard `*`, phrase `"..."`, field-scoped `title:`/`body:`) is the way to check whether a bean for some piece of work already exists before creating a duplicate one. Every read command accepts `--json` for machine-readable output; without it, `beans show` in particular follows stdout — styled on a terminal, raw unpadded/unwrapped Markdown into a pipe or file, so piped `show` output is safe to feed to a parser even without `--json`.

## Write workflows

```bash
beans create --json "Title" -t task -d "Description..." -s todo
beans start <id> [id...]                                   # mark in-progress, show it
beans update --json <id> --blocked-by <other-id>            # THIS bean is blocked by another
beans update --json <id> --body-append "## Notes\n\nmore"   # append to body
beans update --json <id> --body-replace-old "- [ ] x" --body-replace-new "- [x] x"
beans tag <id> [id...] --tag cli --remove-tag stale         # merge tags, one call over n beans
beans complete <id> [id...] --summary "what was done"       # ONLY once every todo is checked off
beans scrap <id> [id...] --reason "why"                     # --reason is required
```

`complete`/`scrap`/`start`/`tag`/`delete` all take multiple IDs in one call; every ID is resolved and policy-checked before the first write, so an unknown ID, a repeated ID, or a policy violation in the batch leaves every bean in it untouched — there is no partial application to clean up after. A batch call that does fail after writes have already started (mid-batch I/O failure, not a validation failure) returns `{"success": false, "beans": [...], "count": n, ...}` naming exactly which beans were written before the failure.

`beans complete` and `beans scrap` are the only commands that both change `status` and append a body section (`## Summary of Changes`, `## Reason for Scrapping`) in one atomic write — do not hand-append those sections and then call `--status completed` separately.

## Concurrency

Update calls should not always assume they hold the only writer. Use etags for optimistic locking when a write follows a read that could have gone stale:

```bash
ETAG=$(beans show <id> --etag-only)
beans update <id> --if-match "$ETAG" --status in-progress
```

A mismatched `--if-match` fails the write and reports the current etag rather than silently overwriting a concurrent change.

## JSON and error handling

Every command failure surfaces in exactly one place, and which place depends on whether that command reaches its JSON output path:

- Without `--json`: two lines to stderr, empty stdout — `Error: <message>` followed by `Run '<command> --help' for usage.`.
- With `--json`: one JSON error document to stdout, empty stderr — `{"success": false, "error": "<message>", "code": "<CODE>"}`. Errors that occur before JSON handling, such as loading an unreadable config or missing store, still surface on stderr.

Successful JSON shapes vary by command. `list`, `show`, `progress`, and `roadmap` return domain data directly; `check` has its own `success` plus diagnostic fields; `update` returns the updated bean; and lifecycle mutations commonly return a success envelope. Agents should branch on process exit status first, parse the command-specific success shape second, and use `error` plus `code` for a JSON failure.

Error codes: `NOT_FOUND`, `NO_BEANS_DIR`, `INVALID_STATUS`, `FILE_ERROR`, `VALIDATION_ERROR`, `CONFLICT`, `POLICY_VIOLATION`.

## GraphQL for bulk/relational reads

`beans graphql` (aliased `beans query`) runs a GraphQL query or mutation against the same data `list`/`show`/`update` operate on, and is the better tool once a task needs several fields across many beans, or needs to traverse relationships in one round trip instead of one `show` per bean:

```bash
# Actionable beans with full detail in one call
beans graphql --json '{ beans(filter: { excludeStatus: ["completed", "scrapped"], isBlocked: false }) { id title status type body } }'

# A bean plus its immediate relationships
beans graphql --json '{ bean(id: "<id>") { title body parent { title } children { id title status } } }'

# A single-field mutation
beans graphql --json 'mutation { updateBean(id: "<id>", input: { status: "completed" }) { id status etag } }'

# Print the full schema before writing a non-trivial query
beans graphql --schema
```

`beans graphql --schema` is always the ground truth for field names and argument shapes; do not assume a field exists (or still has its old name) without checking it there first, since the schema is generated from the same Go types the CLI itself uses, not maintained separately. The query root exposes `bean(id)` and `beans(filter)`; the mutation root exposes `createBean`, `updateBean`, `deleteBean`, `archiveBean`, and the same parent/blocking mutations the CLI flags map to. `BeanFilter` on the query side covers everything `beans list`'s flags cover (status/type/priority include and exclude lists, tag include/exclude, parent/blocking/blocked-by presence and target-ID filters, `isBlocked`/`isExplicitlyBlocked`/`isImplicitlyBlocked`, and `search` using the same Bleve syntax as `--search`).

## Lifecycle safety

- **Never skip `beans complete`'s precondition.** Only run it once every todo item in the bean's body is checked off; if work was deferred, create a follow-up bean for it first rather than completing early.
- **`beans delete` is destructive and cascades.** It removes incoming references (as parent or via blocking) from every bean that named the deleted one, after a confirmation the agent should not bypass with `-f`/`--json` (which implies `-f`) unless the user explicitly asked for an unconditional delete.
- **`beans archive` is not destructive but is one-directional from the CLI's perspective.** It only ever moves `completed`/`scrapped` beans into `<beans-path>/archive/`; run it only when the user asks for it, not as a routine part of completing work, since archived beans stop showing up in a plain directory listing even though every `beans`/GraphQL query still sees them.
- **A required-fields policy blocks the transition, not the bean.** If a project's `.beans.yml` has `beans.require_fields_on`, an unqualified transition into a gated status fails with `POLICY_VIOLATION` naming the missing field(s); `beans prime`'s output for the current project states exactly which fields gate which statuses, including whether the configured commit field is one of them — check there rather than guessing a policy from a different project.
- **Prefer `--blocked-by` when creating new dependent work.** It lets the new bean describe its own dependency without mutating the bean it depends on, and keeps write scope to the bean actually being changed.
- **Include bean files in the same commit as the code change they track.** Both new and modified bean files, including ones marked completed or scrapped as part of the change, belong in the same commit as the code — not a separate housekeeping commit.
- **Any exploratory or example invocation that writes data must target a throwaway store**, e.g. `beans --beans-path /tmp/scratch-beans create ...`, never the project's real `.beans` directory or a shared `BEANS_PATH`.

## Related documentation

- [wiki/data-model.md](data-model.md)
- [wiki/configuration.md](configuration.md)
- [wiki/commands/lifecycle.md](commands/lifecycle.md)
- [wiki/commands/inspection-and-search.md](commands/inspection-and-search.md)
- [wiki/commands/querying-and-automation.md](commands/querying-and-automation.md)
- [wiki/commands/validation-and-maintenance.md](commands/validation-and-maintenance.md)
- [wiki/web-ui-and-api.md](web-ui-and-api.md)
