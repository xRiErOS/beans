---
# beans-2ark
title: Preserve and expose custom front matter
status: in-progress
type: epic
priority: high
tags:
    - to-review
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:55:15Z
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
