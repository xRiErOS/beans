---
# beans-lk3p
title: Fork-Binary aus fix/beans-ti53 bauen (version-stamped)
status: completed
type: task
created_at: 2026-07-21T18:39:26Z
updated_at: 2026-07-21T18:39:26Z
parent: beans-f1t4
---

Fork-beans aus Worktree fix-ti53 (branch fix/beans-ti53-roadmap-nested-hierarchy) bauen.

## Akzeptanz
- [x] Build-Target ./cmd/beans (nicht repo-root)
- [x] ldflags Version-Stamp: 0.4.2-fork.ti53, Commit-SHA, Datum
- [x] Binary laeuft: 'beans version' -> 0.4.2-fork.ti53 (3419e8a)

## Summary of Changes
command go build -o /tmp/beans-fork mit ldflags -X .../version.{Version,Commit,Date} aus cmd/beans gebaut. dist-Stub (internal/web/dist/index.html) vorhanden -> Backend kompiliert.
