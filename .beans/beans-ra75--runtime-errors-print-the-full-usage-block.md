---
# beans-ra75
title: Runtime errors print the full usage block
status: todo
type: feature
created_at: 2026-07-26T13:19:49Z
updated_at: 2026-07-26T13:19:49Z
---

## Origin

Found while operationalizing a plan in `~/Obsidian/tools/okf-tools` on 2026-07-26 (epic `okf-cli-5uog`, ~30 `beans update` calls in one session). Two of those updates did not take effect and the failure went unnoticed until a verification agent caught the resulting tree defect two rounds later.

**The first suspicion was wrong and is recorded here so nobody re-files it:** `beans update` does *not* fail silently. Probed directly — a bad parent, a missing bean and an unmatched `--body-replace-old` each print a clear error on stderr and exit 1. Multi-line `--body-append` via command substitution, bodies starting with `##`, and sequences with stdout suppressed all work correctly (verified on a throwaway bean, then removed). The CLI reports its errors properly.

**What actually hurt** is the shape of that report.

## The defect

Every returned error is followed by the command's **full usage block**. Measured: `beans update --help` is 33 lines; a one-line runtime error therefore ships with ~33 lines of flag documentation behind it. Observed verbatim:

```
$ beans update okf-cli-t7jz --parent okf-cli-NOPE
Error: parent bean not found: okf-cli-NOPE
Usage:
  beans update <id> [flags]

Aliases:
  update, u

Flags:
      --blocked-by stringArray          ID of bean that blocks this one (can be repeated)
      ... 30 more lines ...
```

Cause: no `SilenceUsage` is set anywhere in the repository (`grep -rn SilenceUsage .` → no matches), so cobra falls back to printing usage for every non-nil error returned from `RunE`, whether the error is about how the command was *called* or about what happened while it *ran*. The two error classes shown above are the second kind: `pkg/beancore/links.go:475` (`parent bean not found`) and `pkg/bean/content.go:17` (`text not found in body`) are runtime outcomes — the invocation was syntactically fine and the usage text tells the reader nothing they got wrong.

## Why this matters more for an agent than for a human

A human sees the error at the top of a short scroll. An agent batches several commands into one call, and its harness returns a bounded slice of the combined output. A 1:33 signal-to-usage ratio per error is what pushes real output out of that window — in the incident above, a failing pre-commit hook plus one usage block was enough to hide an earlier command's outcome entirely. The project describes itself as "A file-based issue tracker for AI-first workflows" (`internal/commands/root.go:22`); the error path is the one place that claim is currently not honoured.

### Requirement 1: A runtime error reports the error, not the manual

**Objective:** As an agent parsing command output, I want a failed operation to print only what failed, so that one error does not displace the rest of a batch's output.

#### Acceptance Criteria

1. WHEN a command returns an error that is not about invocation syntax, THE CLI SHALL print the error message and SHALL NOT print the usage block.
2. WHEN a command is called with unknown flags, missing required arguments or an unparsable value, THE CLI SHALL still print the usage block — that is the case usage was written for.
3. THE CLI SHALL keep exit code 1 for both classes; this changes what is printed, not what is signalled.
4. THE error message SHALL remain on stderr, so that `--json` consumers reading stdout are unaffected.

#### Success Criteria

- SC-01: `beans update <id> --parent <nonexistent>` prints exactly one line, `Error: parent bean not found: <id>`, and exits 1. No `Usage:` in the output.
- SC-02: `beans update <id> --body-replace-old <absent> --body-replace-new x` prints exactly one error line and exits 1.
- SC-03: `beans update --nonexistent-flag` still prints the usage block and exits 1.
- SC-04: `beans update <id> -s completed` (the success path) prints its confirmation unchanged.
- SC-05: the existing test suite stays green; a regression test asserts the absence of `Usage:` for SC-01/SC-02 and its presence for SC-03.

## Implementation note

The usual cobra idiom is `SilenceUsage: true` on the root command plus explicit usage printing (or `cmd.SilenceUsage = false`) in the argument-validation path — not per-subcommand flags, which drift. Root command is at `internal/commands/root.go:20`; all three binaries (`cmd/beans`, `cmd/beans-serve`, `cmd/beans-tui`) route through `commands.Execute(root)`, so one setting covers them.

Consider `SilenceErrors` separately: it is a different switch (suppressing cobra's own error printing in favour of the caller's) and is NOT wanted here — the error line must stay.
