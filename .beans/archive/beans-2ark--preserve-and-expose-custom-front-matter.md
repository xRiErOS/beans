---
# beans-2ark
title: Preserve and expose custom front matter
status: completed
type: epic
priority: high
tags:
    - accepted
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T12:21:48Z
parent: beans-xej5
---

Package 1 of `docs/beans-planning-primitives/BRIEFING.md`. Requirements R-01 to R-08.

`Parse` (`pkg/bean/bean.go:185`) reads through `github.com/adrg/frontmatter` into the `frontMatter` struct (`pkg/bean/bean.go:170`), and `Render` (`pkg/bean/bean.go:227`) writes back out of `renderFrontMatter` (`pkg/bean/bean.go:212`). Whatever is in neither struct stops existing after the first write. Measured on 2026-08-10: `beans update <id> --priority high` removed three hand-added keys without a trace.

This epic goes first and alone — it changes the data format, and everything else builds on it.


## Epic Summary (to-review)

All seven leaves completed:
- beans-n3hl: Bean.Extra parses unknown front matter keys
- beans-54rb: Render writes Extra back sorted, stable, SC-02 reverse-mutation verified
- beans-re1p: --set/--unset in create/update, reserved-key rejection
- beans-k9cc: beans version reports the custom-front-matter capability
- beans-7ohz: extra surfaces in list/show --json (already worked via MarshalJSON; locked with regression tests)
- beans-slvx: extra exposed through the GraphQL schema (Map scalar), codegen committed
- beans-usk9: --where key=value filters on extra keys

Every leaf independently verified (go build/go test rerun outside the implementer's own report) before being accepted here. Full `go build ./...` clean at HEAD of this epic's work; `go test ./...` (excluding pkg/bean transiently, see beans-y2a2's in-flight note if still open at review time) green across all touched packages.

Ready for PO review.


## Review Round 1 (Opus, independent verification)

Opus subagent review of all seven leaves against their acceptance criteria, running its own tests/smoke checks (not trusting bean self-reports): six passed independently; `beans-re1p` failed on B01 (ETag concurrency bypassed by the extra-key second write, two reproduced failure modes) plus I01 (no drift-guard between the three parallel known-key lists in `pkg/bean/bean.go`).

Both fixed:
- B01: `internal/commands/create.go`/`update.go` now pass a correct `ifMatch` (the bean's own etag, or the caller's `--if-match`) to the second write instead of `nil`. RED→GREEN reproduced independently, see `beans-re1p`'s Fix Round 1 section.
- I01: `TestKnownKeyMapsMatchFrontMatterTags` (`pkg/bean/bean_test.go`) reflects over `frontMatter`'s yaml tags and asserts both `knownFrontMatterKeys` and `reservedKeyFlags` match exactly. Reverse-mutation verified (removing an entry turns it red).

Non-blocking findings deferred per the reviewer's own verdict (not re-litigated here, not fixed in this epic): B02 (--where's error names --set instead of --where), B03 (empty --set key silently accepted), I02/I03 (two tests assert on Go structs rather than the file/wire — the properties they guard hold, verified manually by the reviewer, but the automated coverage is weaker than the SC wording).

`go build ./...` clean, `go test ./...` all green at HEAD (46b113a).

Still to-review — ready for the PO now.

## Review 2026-08-10

US-01 · Custom front matter keys survive a `beans update` round-trip instead of being silently dropped · a
US-02 · `beans create/update --set/--unset` lets you write and remove custom front matter keys from the CLI, reserved keys rejected with the native flag named · a

US-03 · \`list\`/\`show --json\` expose custom front matter keys under an \`extra\` field, omitted when empty · a
US-04 · GraphQL exposes \`extra\` on query and preserves it across mutations (API-only, no frontend UI yet) · a
US-05 · \`beans list --where key=value\` filters on custom front matter keys · a
US-06 · \`beans version --json\` reports the custom-front-matter capability · a
