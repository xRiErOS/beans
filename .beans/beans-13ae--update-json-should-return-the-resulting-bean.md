---
# beans-13ae
title: update --json should return the resulting bean
status: todo
type: feature
priority: normal
created_at: 2026-07-26T13:20:13Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-xej5
---

## Origin

Same session as `beans-ra75` (2026-07-26, ~30 `beans update` calls while operationalizing a plan in `okf-tools`). Two updates did not take effect; the tree defect that caused was found two verification rounds later, not at the moment of failure.

`beans update` reported both correctly — see `beans-ra75` for the proof that there is no silent-failure defect. The gap is not in the reporting, it is that **a mutation's outcome cannot be re-read without a second command**. An agent that suppresses stdout (routine when 30 confirmations would otherwise flood its context) keeps the exit code and nothing else, and an exit code does not say *what the bean now looks like*.

## The improvement

`beans update` prints `Updated <id> <filename>` on success — an acknowledgement, not a state. To verify the mutation actually landed as intended, the caller must issue `beans show <id>` and parse it, which is a second process launch per mutation and, at 30 mutations, exactly the cost that makes agents skip verification.

`--json` already exists on `update`. What it returns should be the bean's resulting state, so that one call both mutates and reports — the read-after-write that makes a suppressed-output workflow safe.

### Requirement 1: A mutation can be verified from its own output

**Objective:** As an agent applying many updates in sequence, I want each update to hand back the resulting bean, so that I can confirm the change landed without a second command per mutation.

#### Acceptance Criteria

1. WHEN `beans update` is called with `--json` and succeeds, THE CLI SHALL emit the complete resulting bean — at minimum id, title, type, status, priority, parent, blocked_by, tags and body — in the same schema `beans show --json` returns.
2. THE non-`--json` output SHALL remain unchanged, so existing human-facing and script-facing behaviour is untouched.
3. WHERE a single invocation applies several changes (status plus parent plus body edit), THE emitted bean SHALL reflect all of them, read back after the write rather than assembled from the request.

#### Success Criteria

- SC-01: `beans update <id> --parent <p> --json | jq -r .parent` prints `<p>` without any further command.
- SC-02: `beans update <id> --body-append "X" --json | jq -r .body` ends with `X`.
- SC-03: `beans update <id> -s completed --json` returns a payload that validates against the same schema as `beans show --json <id>`, field for field.
- SC-04: `beans update <id> -s completed` without `--json` prints exactly what it prints today.

## Why this pairs with beans-ra75

`beans-ra75` reduces what a failure prints; this reduces what a success costs to confirm. Together they make the batched-mutation workflow — the one an AI-first tracker is built for — verifiable at one command per change instead of two.

## Deliberately not proposed here

A `--quiet` flag, or suppressing the success line by default. The confirmation line is useful to humans and cheap; the problem is not that it prints, it is that it does not carry the state. Adding a flag would give callers one more thing to get wrong.
