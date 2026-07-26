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

`delete.go` also constructs an `output.Response{…}` literal by hand for the
multi-bean case — a sixth variant of the same envelope, assembled locally.

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
