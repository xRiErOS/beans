---
# beans-2ark
title: Preserve and expose custom front matter
status: todo
type: epic
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-xej5
---

Package 1 of `docs/beans-planning-primitives/BRIEFING.md`. Requirements R-01 to R-08.

`Parse` (`pkg/bean/bean.go:185`) reads through `github.com/adrg/frontmatter` into the `frontMatter` struct (`pkg/bean/bean.go:170`), and `Render` (`pkg/bean/bean.go:227`) writes back out of `renderFrontMatter` (`pkg/bean/bean.go:212`). Whatever is in neither struct stops existing after the first write. Measured on 2026-08-10: `beans update <id> --priority high` removed three hand-added keys without a trace.

This epic goes first and alone — it changes the data format, and everything else builds on it.
