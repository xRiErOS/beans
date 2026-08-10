---
# beans-3o3r
title: Follow-ups from beans-2ark Opus spec review (B02, B03, I02, I03)
status: draft
type: task
priority: normal
created_at: 2026-08-10T11:19:22Z
updated_at: 2026-08-10T11:19:40Z
parent: beans-2ark
---

Deferred, non-blocking findings from the independent Opus spec review of `beans-2ark` (all seven leaves). None of these block the epic's PO gate per the reviewer's own verdict — parked here so they are not lost.

### B02 (low)

`beans list --where noequals` reports the wrong flag name: `Error: invalid --set value "noequals": expected key=value`. `validateWhereKeys` (`internal/commands/list.go:299`) reuses `parseSetPair`, whose error message (`internal/commands/content.go:101`) is hardcoded to `--set`. Contradicts the intent behind `beans-re1p` AC4 (errors should name the actual flag used).

Fix: give `parseSetPair` a flag-name parameter, or have the caller reformat the message.

### B03 (low)

An empty extra-front-matter key is silently accepted: `beans create "e" -t task --set "=value"` exits 0 and writes `"": value` into the front matter. `parseSetPair` (`content.go:99`) lets `=value` through (key `""`), and `checkReservedKey("")` doesn't catch it. No AC covers this case, but the resulting key is unaddressable by any other command.

Fix: reject an empty key in `validateExtraKeys` as a usage error.

### I02 (test-coverage)

`TestParseRenderRoundtripWithExtraKeys` (`pkg/bean/bean_test.go:1906`) is evergreen against the feature it claims to protect: it starts from a `Bean` struct (not a parsed file), so removing the extra-key branch from `Render` loses the extras on both comparison sides equally and the test stays green. It also doesn't literally cover `beans-54rb` AC2/SC-01 ("a bean file is parsed and rendered ... byte-identical to the input") since it never starts from a file.

Fix: rewrite from a literal file-content constant so a removed `Render` branch actually turns it red.

### I03 (test-coverage)

`TestQueryBeanExtra` and `TestMutationUpdateBeanPreservesExtra` (`internal/graph/schema.resolvers_test.go:3255`, `:3283`) assert on the returned Go struct. `TestQueryBeanExtra` would pass even if `extra` weren't in the GraphQL schema at all (Go-level autobind already carries it). `TestMutationUpdateBeanPreservesExtra` checks the returned object, while `beans-slvx` SC-01 says the pair must remain "in the file" after a mutation.

Fix: add a file-content assertion to the mutation test. A true wire-level GraphQL-string test is optional — this repo has none anywhere, so matching that convention is defensible, but the mutation test should still check the file.

_Source: Opus subagent review of beans-2ark, session 2026-08-10._
