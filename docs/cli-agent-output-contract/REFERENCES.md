# References — cli-agent-output-contract

Evidence backing every claim in this collection. Anything asserted in DECISIONS.md
or QUESTIONS.md must be traceable to a row here.

## Beans under discussion

| ID | Type | Title |
|---|---|---|
| `beans-ra75` | feature | Runtime errors print the full usage block |
| `beans-13ae` | feature | update --json should return the resulting bean |

## Code sites

| Ref | Site | What it shows |
|---|---|---|
| R01 | `internal/commands/root.go:20` | Root command construction. No `SilenceUsage`, no `SetFlagErrorFunc`. |
| R02 | `internal/commands/root.go:22` | Self-description: `"A file-based issue tracker for AI-first workflows"`. |
| R03 | `go.mod:22` | `github.com/spf13/cobra v1.10.2` — the version whose error-handling semantics apply. |
| R04 | `internal/commands/update.go:109` | `if updateJSON { … return output.Success(b, msg) }` — update already emits the full resulting bean. |
| R05 | `internal/commands/update.go:97` | `b, err = resolver.UpdateBean(ctx, b.ID, input)` — `b` is the post-write bean, not a request echo. |
| R06 | `internal/commands/show.go:52-56` | `output.SuccessSingle` / `output.SuccessMultiple` — bare bean / bare array, no envelope. |
| R07 | `internal/output/output.go` `Response` struct | Envelope fields: `success`, `bean`, `beans`, `count`, `message`, `warnings`, `error`, `code`, `path`. |
| R08 | `internal/output/output.go` `SuccessSingle` | Comment states the intent explicitly: *"This allows intuitive jq usage: beans show --json <id> \| jq '.title'"* — the bare form is a deliberate ergonomic choice, already reasoned about once. |
| R09 | `internal/commands/update.go:54,62,67,84,104` | `cmdError(updateJSON, …)` — update already routes errors through a JSON-aware helper. |
| R10 | `internal/commands/content.go:60` | `func cmdError(jsonMode bool, code string, …)` — the shared error helper. |
| R11 | `internal/output/output.go` `Error` | Prints the JSON error doc to **stdout** *and* returns a non-nil `error` — which cobra then also prints, with usage, to stderr. This is the double-emission mechanism. |
| R12 | `pkg/beancore/links.go:475` | `parent bean not found` — a runtime error, syntactically valid invocation. |
| R13 | `pkg/bean/content.go:17` | `text not found in body` — likewise runtime, not invocation. |

## Measurements

Taken 2026-07-26 against the built binary at `beans-src/beans`, in a throwaway
repo (`git init` + `beans init`, bean `bt-9vx3`). Reproducible.

| Ref | Invocation | Observed |
|---|---|---|
| M01 | `beans update <id> --parent NOPE` | rc=1, **33 lines on stderr** (1 error line + 32 usage) |
| M02 | `beans update <id> --bogus-flag` | rc=1, **33 lines on stderr** — flag errors are indistinguishable in shape from runtime errors |
| M03 | `beans update <id> --parent NOPE --json` | rc=1, clean JSON error on **stdout** (`{"success":false,"error":"parent bean not found: NOPE","code":"VALIDATION_ERROR"}`) **and 33 lines on stderr anyway** |
| M04 | `beans update <id> -s in-progress --json` | `{"success":true,"bean":{id,slug,path,title,status,type,priority,created_at,updated_at,body,etag},"message":"Bean updated"}` |
| M05 | `beans update <id> --body-append "APPENDED-X" --json` | returned `bean.body` ends with `APPENDED-X` — body mods are reflected |
| M06 | `beans update <id> -s todo --body-append "SECOND" --json` | returned bean carries **both** the new status and the appended body |
| M07 | `beans show <id> --json` keys | `body, created_at, etag, id, path, priority, slug, status, title, type, updated_at` — bare, no wrapper |
| M08 | all of the above | empty fields (`parent`, `tags`, `blocked_by`) are **absent**, not `null` — `omitempty` on the bean struct |

### What M03–M06 falsify

`beans-13ae` asserts that update "prints `Updated <id> <filename>` on success — an
acknowledgement, not a state" and that verification requires "a second process launch
per mutation". For the `--json` path this is false: AC-1 (complete resulting bean) and
AC-3 (read back after the write, reflecting combined changes) are already satisfied
today. Its SC-01 (`… --json | jq -r .parent`) fails for a different reason than the
bean states — the value is at `.bean.parent`, not missing.

## JSON shape inventory (T01)

Every JSON-emitting command, by emitted shape. **Five shapes across nine commands.**

| Shape | Commands | Site | Payload |
|---|---|---|---|
| Bare bean | `show` (single) | `show.go:54` | `{id, title, …}` |
| Bare array | `show` (multi), `list` | `show.go:56`, `list.go:131` | `[{…}, …]` |
| Envelope + bean | `create`, `update`, `delete` | `create.go:109`, `update.go:114`, `delete.go:80` | `{success, bean, message}` |
| Envelope, message only | `archive` | `archive.go:36,58` | `{success, message}` |
| Envelope + path | `init` | `init.go:80` | `{success, message, path}` |
| **Ad-hoc, two documents** | `rename` | `rename.go:77` and `rename.go:214` | bypasses `internal/output` entirely — raw `json.NewEncoder(…).Encode(map[string]any{…})`, emitting a plan document **and** a result document as two separate top-level JSON values on stdout |
| **Own result struct** | `check` | `check.go:20-23` | `{success, config_errors, bean_issues, fixed}` — a locally declared type, also bypassing `internal/output` |
| Raw GraphQL passthrough | `graphql` / `query` | `graphql.go:105` | the GraphQL response as-is; `--json` here means "no colors", not a shape |

`delete.go` also constructs an `output.Response{…}` literal by hand for the
multi-bean case — another variant of the envelope, assembled locally rather than
through a helper.

**Corrected count: ten commands emit JSON in seven distinct shapes.** The sub-agent
inventory missed `rename`, `check` and `graphql`; those three were found by walking
`internal/commands/` directly. Three of the ten (`rename`, `check`, `graphql`) do not
route through `internal/output` at all — so `internal/output` is not currently the
single construction point for the CLI's JSON, which is what makes the drift possible
in the first place (see Q09).

### The `rename` case is already a known, unfiled wart

`beans-src/CLAUDE.md` documents it verbatim: *"`--json` on apply (non-dry-run) writes
two separate JSON documents to stdout, not one … A naive `json.loads(stdout)` in a
scripting consumer parses only the first document and silently drops the result …
(Collapsing this into a single JSON document is a possible future follow-up, **not yet
tracked as a bean**.)"*

The contract problem D05 asks about has therefore already been hit once, written down
as a caveat for agents to work around, and left unfixed. That is evidence the split is
costing something, not a hypothetical.

## Consumers of the mutation envelope (T01)

| Consumer | Site | Reads | Breaks on bare form? |
|---|---|---|---|
| Playwright e2e fixture | `frontend/e2e/fixtures.ts:111` | `(json.bean?.id ?? json.id)` from `beans create --json` | **No** — verified by reading the line. It is already written to accept both shapes. |
| `bnew` shell wrapper | `~/code/dotfiles/shell/.zshrc:559` | `beans list --json \| jq -r '.[].tags[]?'` | No — `list` is already bare. |
| anything else | — | none found across `beans-src`, `~/.claude`, `~/code/dotfiles`, `lean-stack` | — |

**Correction to the sub-agent's report:** it marked the e2e fixture as breaking. The
source shows the opposite — the `?? json.id` fallback makes it shape-tolerant, and its
presence suggests someone already anticipated this change once. The finding was
re-checked at source before being recorded here.

**Consequence for D05:** the "breaking change" framing in `beans-13ae` and in the
original D05 option set is not supported by evidence. Zero consumers break on a move
to the bare form. The cost of unifying is close to nil; the cost of *not* unifying is
already documented in CLAUDE.md as an agent-facing caveat.

## The okf-tools incident, reconstructed (Q07)

Both beans open with an Origin section blaming an incident on 2026-07-26 in
`~/dev/okf-tools` (epic `okf-cli-5uog`). Forensics on the session
transcript, the hook, and the git history show **two independent causes, and the one
that lost the updates is not a beans defect at all.**

### Cause 1 — the two lost updates never ran

At `13:04:28Z` the agent issued **one compound Bash call**: four `beans update …`
statements, each suffixed `>/dev/null`, followed by `git add && git commit -q -m
"chore(beans): operationalize capability surface plan"` — a 52-character title.

`~/.claude/hooks/git-enforce.py` is a `PreToolUse(Bash)` hook that returns
`permissionDecision: "deny"`. **A deny rejects the entire tool call before a shell is
spawned.** The four beans mutations never executed. The error the agent received was a
single line:

```
E1: Commit-Title >50 Zeichen (52). Kuerzen.
```

Three seconds later the agent shortened the title and re-ran **only the git portion**.
The four mutations were dropped permanently.

| Ref | Artifact | Shows |
|---|---|---|
| I01 | session jsonl @ `13:04:28.045Z` | the compound call, 4× `beans update … >/dev/null` + `git commit` (52 chars) |
| I02 | same @ `13:04:28.117Z`, `is_error=true` | the whole tool result is the one E1 line — the beans statements are not mentioned |
| I03 | `~/.claude/hooks/git-enforce.py:2,25,30-34,89` | `PreToolUse(Bash)`, `TITLE_MAX = 50`, `permissionDecision: "deny"`; the reason string names only the violated rule. **Verified independently at source.** |
| I04 | `~/.claude/settings.json` → `PreToolUse`, matcher `Bash` | wiring confirmed; nothing in the block ran |
| I05 | same @ `13:04:31.648Z` | the retry is git-only; the four updates are never reissued |
| I06 | `git show e61de25:.beans/okf-cli-a4as--….md` | committed with `updated_at: 13:04:17Z` and **no `parent:` key** — the file was never touched |
| I07 | same @ `13:10:03.879Z` | the identical command later succeeds verbatim, rc=0 — the invocation was always valid |

**This is a hook-ergonomics defect, not a CLI defect.** A `PreToolUse` deny is
all-or-nothing over a compound command, but its reason describes a single statement,
so the blast radius is invisible to the caller. Neither `beans-ra75` nor `beans-13ae`
addresses it, and neither would have prevented it.

### Cause 2 — the duplicate orphans, and this one *is* the envelope

The same session produced three duplicate orphan beans (`wafj`, `ku01`, `6361`, later
scrapped). Mechanism:

```
beans create --json … | python3 … json.load(sys.stdin).get('id','')
```

`id` is **not** at the top level — it is nested under `bean` (M04). The extractor
printed an empty string, the agent concluded the creates had failed, and re-created
them. The beans had in fact been created at `13:02:30Z`.

| Ref | Artifact | Shows |
|---|---|---|
| I08 | session jsonl @ `13:02:30.115Z` | `Bash completed with no output` — the extractor returned empty while the creates succeeded |
| I09 | disk: `wafj`/`ku01`/`6361` — `parent: -NONE-`, `scrapped`, commit `596d07f` | the duplicates, orphaned and later cleaned up |

**Consequence for D05.** The envelope inconsistency is no longer an ergonomics
argument. It has already caused an agent to misread a successful mutation as a failure
and corrupt the tree in response. `frontend/e2e/fixtures.ts:111` reads
`(json.bean?.id ?? json.id)` precisely because someone hit this before and defended
against it locally instead of fixing the shape. D05 has an incident behind it.

### Consequence for the two beans' Origin sections

Both beans present the incident as evidence for the change they propose. For `ra75`
that link is now false — the usage noise did not cause the loss; a hook deny did.
`ra75` remains a legitimate improvement on its own merits (33 lines of manual per
error, in a tool that calls itself AI-first), but its Origin narrative must be
corrected or it will keep justifying itself with an incident it did not cause.

## Serialisation facts (D08)

All commands serialise the **same** `bean.Bean` struct (`pkg/bean/bean.go:138-166`),
so `omitempty` is a global property of the type, not a per-command choice.

| Field | JSON tag | Effect when empty |
|---|---|---|
| `id`, `path`, `title`, `status` | no `omitempty` | always present |
| `type`, `priority`, `slug`, `body`, `order` | `omitempty` | key absent |
| `parent`, `tags`, `blocked_by`, `blocking` | `omitempty` | key absent |

Two consequences for D08:
- The absent/present split is already **inconsistent** — `status` always appears,
  `type` does not, for no stated reason.
- Emitting `null` rather than omitting is not a tag change: `Parent` is a `string`, so
  dropping `omitempty` yields `"parent": ""`, not `"parent": null`. Real nullability
  needs `*string`, which touches YAML round-tripping of the frontmatter as well. D08-B
  is a type change, not a serialisation tweak.

## Build path (Q06)

`go build -o /tmp/beans-cli ./cmd/beans` succeeds in ~2s, rc=0. The `//go:embed dist/*`
lives in `internal/web/embed.go`, reached by `beans-serve`, and does not gate a CLI
build. The failing `mise build` (frontend/pnpm) therefore does **not** block work on
this contract — CLI changes can be built and tested Go-only.

## Cobra semantics (v1.10.2) — verified against source

Source: `~/go/pkg/mod/github.com/spf13/cobra@v1.10.2/command.go`, `ExecuteC`,
lines 1147–1168. Read directly; no longer inferred from documentation.

```go
err = cmd.execute(flags)
if err != nil {
    if errors.Is(err, flag.ErrHelp) { cmd.HelpFunc()(cmd, args); return cmd, nil }
    if !cmd.SilenceErrors && !c.SilenceErrors {      // c == root
        c.PrintErrln(cmd.ErrPrefix(), err.Error())
    }
    if !cmd.SilenceUsage && !c.SilenceUsage {        // c == root
        c.Println(cmd.UsageString())
    }
}
```

| Ref | Fact | Consequence |
|---|---|---|
| R14 | The usage check is evaluated **after** `cmd.execute()` returns (line 1163–1166). | A `RunE`, `Args` validator or `FlagErrorFunc` can mutate the silence flags during the call and the change is honoured. Confirmed. |
| R15 | The condition is a conjunction over **both** the executed command and the root: usage prints only if *neither* has `SilenceUsage`. | `SilenceUsage: true` on the root silences usage for **every** error class — runtime and flag-parse alike. |
| **R16** | **Because it is a conjunction, setting `cmd.SilenceUsage = false` in the invocation-error path does NOT restore the usage block** — the root's `true` still short-circuits it. Restoring requires `cmd.Root().SilenceUsage = false`, or printing `cmd.UsageString()` explicitly. | **The implementation note in `beans-ra75` is wrong as written.** It proposes exactly the insufficient `cmd.SilenceUsage = false`. An implementer following it would ship AC-1 working and AC-2 silently broken, with SC-03 catching it only if the regression test is written first. |
| R17 | A **separate, earlier** error path exists for `Find`/`Traverse` failures (unknown *command*), lines 1124–1133. It never prints usage. It prints the error line plus `Run '<command path> --help' for usage.` and returns. | Cobra already ships the exact shape ra75 is asking for — one error line plus a one-line pointer to the manual. It is applied to unknown commands but not to anything else. This is a fourth option for D07 and arguably the most idiomatic one, since it copies upstream rather than inventing a policy. |
| R18 | `errors.Is(err, flag.ErrHelp)` is handled before both branches. | `--help` is unaffected by any silencing decision. No risk to the help path. |

Not yet checked: whether `PersistentPreRunE` failures (root.go config/load errors)
travel the same path — they are returned from `cmd.execute()`, so they should, but
the `beans init`/`prime`/`version` early-return branch was not probed.
