# DESIGN — the CLI's output contract for agents

Convergence artifact of the think session of 2026-07-26. Scaffolding, not a
specification: it fixes *what* the contract is and *why*, and hands the acceptance
work to the beans. No EARS here — that is written when the work is operationalised.

Decisions: `DECISIONS.md` (D01–D13, all settled). Evidence: `REFERENCES.md`.

## The problem, as it actually is

`beans` calls itself "A file-based issue tracker for AI-first workflows"
(`internal/commands/root.go:22`). Two places do not honour that, and neither is the
place the originating beans pointed at.

**On success, the shape is not a contract.** Ten commands emit JSON in seven shapes.
`show` returns a bare bean, `update` wraps it under `bean`, `archive` returns only a
message, `rename` emits two top-level documents on one stream, `check` declares its
own struct. Three of the ten never touch `internal/output`, so there is no single
place where the shape is decided — which is how the drift happened and how it would
happen again.

This has already cost something. In `okf-tools` on 2026-07-26 an agent extracted an
ID from `beans create --json` with `.get('id','')`, read nothing because `id` sits
under `bean`, concluded the creates had failed, and re-created three beans as
orphans. `frontend/e2e/fixtures.ts:111` reads `(json.bean?.id ?? json.id)` — someone
met the same trap earlier and defended against it locally instead of removing it.

**On failure, the signal is buried.** Cobra prints the full usage block after every
returned error, because no `SilenceUsage` is set anywhere. Measured: 33 lines of
stderr for a one-line error, identical for a runtime failure and a typo'd flag. With
`--json` the error is emitted twice — a correct machine-readable document on stdout
*and* the 33 lines on stderr. The consumer doing it right pays the full price.

## The contract

### Success: shape carries meaning, and it is uniform

| Result | Shape |
|---|---|
| One bean | the bare bean object |
| Many beans | a bare array of bean objects |
| No bean (e.g. `archive`, `init`) | an object describing what happened |

`create`, `update` and `delete` drop their envelope. `show` and `list` are already
right and do not change. `rename` collapses to one document per invocation. `check`
keeps its own result type but constructs it through `internal/output`.
`graphql`/`query` is a raw GraphQL passthrough and stands outside this contract.

`jq -r .parent` means the same thing at every call site. That is the whole point.

Empty fields stay absent rather than `null` (D08): `Parent` is a `string`, so removing
`omitempty` would yield `""`, not `null`, and real nullability would require `*string`
and drag YAML frontmatter round-tripping along. Disproportionate; the absence is the
contract.

### Failure: exactly one machine-readable artifact

| Invocation | stdout | stderr | rc |
|---|---|---|---|
| `--json` | the error document `{success:false, error, code}` | *empty* | 1 |
| without `--json` | empty | `Error: <what failed>` + `Run '<cmd> --help' for usage.` | 1 |

An error has no bean to be, so it keeps an envelope — the deliberate counterpart to a
bare success, and unambiguous against it (a bean has no `error` key). Under `--json`,
cobra's own error print is silenced (`SilenceErrors`) so the document is the only
output. Without `--json`, the human gets two lines instead of thirty-three.

The two-line form is not invented here: cobra already emits exactly this for unknown
commands (`command.go:1124-1133`). We adopt its own idiom rather than writing a policy,
and apply it to **every** error class — including flag errors. This supersedes
`beans-ra75`'s AC-2, which wanted flag errors to keep the block; the mechanism that
bean proposed cannot deliver that split anyway (`SilenceUsage` is checked as a
conjunction over command and root, so `cmd.SilenceUsage = false` cannot restore it).

### The seam that keeps it true

`internal/output` becomes the only place response shapes are constructed, with a test
asserting that no command package encodes JSON directly. Without an enforced seam the
contract decays at the next new command — the drift documented above is what an
unenforced convention looks like after a while.

## What this deliberately does not do

- **No `--quiet` flag, no new flags at all.** The success line is cheap and useful to
  humans; the problem was never that it prints.
- **No change to exit codes.** Both error classes stay rc=1. This changes what is
  printed, not what is signalled.
- **No auto-migration for consumers.** The inventory found none that break; a
  compatibility shim would be dead code on arrival.
- **`graphql`/`query` stays out.** It is a passthrough, and pretending otherwise would
  mean reshaping someone else's protocol.

## Carry-over into operationalisation

The two originating beans need correcting before they are worked, not after:

| Bean | Correction |
|---|---|
| `beans-ra75` | Its Origin blames the incident on usage noise. False — a hook deny caused it. Keep the bean on its own merits, fix the narrative. Rewrite AC-2/SC-03 to D07. Re-type to `bug` (D09). Its implementation note prescribes a mechanism that cannot work (R16) and must be replaced. |
| `beans-13ae` | Its premise is falsified: `update --json` already returns the full post-write bean. Reframe to the envelope contract, add the Q07 cause-2 evidence, correct AC-1's field list to D08 and SC-01 to the bare accessor. |

Open work is enumerated in `TASKS.md` (T03, T04, T05, T07, T10–T16). Two items are
already done and out: the hook defect (D13, commit `fad35712`) and `docs/` versioning
(D10).
