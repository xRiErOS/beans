# Questions — cli-agent-output-contract

Loose ends. A question here is not yet a decision — it either becomes one (moves to
DECISIONS.md) or gets deliberately deferred with a reason.

**All questions resolved 2026-07-26.** Kept in full because several of them were
resolved by *falsifying* the assumption behind them, and that record is the point.

| ID | Question | Resolution | Status |
|---|---|---|---|
| Q01 | Does cobra v1.10.2 evaluate `SilenceUsage` after `RunE` returns, and how does it combine command and root? | `cobra@v1.10.2/command.go:1163`. Evaluated after `cmd.execute()` returns, but as a **conjunction** over command *and* root. The mechanism `beans-ra75` prescribes (`cmd.SilenceUsage = false`) therefore cannot restore usage, and would ship AC-2 silently broken. Superseded by D07. | 🟢 |
| Q02 | Who parses `beans update --json` today? | Two consumers, **neither breaks** on a bare shape. `bnew` reads the already-bare `list`; `frontend/e2e/fixtures.ts:111` reads `(json.bean?.id ?? json.id)` — already shape-tolerant, and in hindsight a local defence against the very trap that later caused the duplicate orphans. | 🟢 |
| Q03 | How far does the envelope question reach? | Ten commands, seven shapes. Three (`rename`, `check`, `graphql`) bypass `internal/output` entirely; `delete` hand-rolls a `Response` literal. → D05, D12. | 🟢 |
| Q04 | Should the JSON error document go to stdout or stderr? | stdout. It is the protocol channel and already carries the document; moving it would make `beans update --json \| jq` yield nothing on failure. → D06. | 🟢 |
| Q05 | Is `beans query`/`graphql` in scope? | No — `--json` there means "no colours" and the payload is a raw GraphQL response. Out of scope. `check` **is** in scope and was missed by the first inventory. | 🟢 |
| Q06 | Does the failing `mise build` gate this work? | No. `go build ./cmd/beans` succeeds, rc=0, ~2s; the frontend embed is only reached by `beans-serve`. | 🟢 |
| Q07 | What actually made the two updates not take effect? | **Two causes, neither the one the beans assumed.** (1) The updates never ran: a `PreToolUse(Bash)` deny from `git-enforce.py` rejected a compound call over a 52-char commit title while naming only the git rule, so the agent re-ran the git half and dropped four mutations. (2) The duplicate orphans *were* the envelope: `.get('id','')` read nothing because `id` sits under `bean`. See REFERENCES → okf-tools incident. | 🟢 |
| Q08 | Does `rename --json`'s two-document emission get fixed or ratified? | Fixed under D05; the `CLAUDE.md` caveat comes out with it. → D11. | 🟢 |
| Q09 | Should `internal/output` be the only construction point, enforced by a test? | Yes. Without an enforced seam the contract decays at the next new command. → D12. | 🟢 |
| Q10 | Does the error document keep `{success:false,…}` once success is bare? | Yes — an error has no bean to be, so its envelope is the deliberate counterpart to a bare success, and the shapes are unambiguous (a bean has no `error` key). → D06. | 🟢 |
| Q11 | Suppress cobra's stderr print when `--json` is active? | Yes, `SilenceErrors` under `--json`. Exactly one machine-readable artifact, no duplication. → D06. | 🟢 |
| Q12 | Where does the hook defect belong, and what is the fix? | Fixed in place, `~/.claude/hooks/git-enforce.py`, commit `fad35712` — `deny()` now enumerates every statement in the call and states none ran. → D13. | 🟢 |
