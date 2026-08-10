---
# beans-w98m
title: 'roadmap: respect order/priority/dependency instead of created_at only'
status: completed
type: bug
priority: normal
created_at: 2026-08-10T17:26:34Z
updated_at: 2026-08-10T17:33:34Z
---

beans roadmap sorts milestones/epics/features/other by created_at/title only, ignoring the manual order key, priority, and Blocking/BlockedBy dependencies. See plan at /Users/erik/.claude/plans/beans-roadmap-sortiert-milestones-cozy-summit.md for full design (new pkg/bean/sort_dependency.go with dependency-aware topological sort, roadmap.go call-site replacement).


## Summary of Changes

Added `pkg/bean/sort_dependency.go` with `SortRoadmapContainers` (type-homogeneous lists: milestones, epics/features within a milestone) and `SortRoadmapLeaves` (type-mixed leaf lists), both implementing status -> dependency (Blocking/BlockedBy, topological via a tie-break-aware Kahn's algorithm) -> manual order -> priority -> created_at. Cycles fall back deterministically instead of hanging.

Replaced all seven sort call sites in `internal/commands/roadmap.go` (milestones, unscheduled epics/features, milestone epics/features, orphan/other leaf items) with the new functions, removing the dead `sortByStatusThenCreated` and `sortByTypeThenStatus`. Added two small helpers (`sortEpicGroups`, `sortFeatureGroups`) to reorder the `[]epicGroup`/`[]featureGroup` parallel structures by their underlying bean's rank.

Tests: `pkg/bean/sort_dependency_test.go` (status ordering, order-key, dependency precedence, chains, cycle safety, out-of-slice edges, priority/created_at fallback, type grouping for leaves) and two new cases in `internal/commands/roadmap_test.go` proving `beans roadmap` now reflects a manual `order` key and a `Blocking` relation between milestones. Full suite (`go test ./...`) green, `go vet` clean.

Manually verified end-to-end against a scratch copy of `~/dev/sproutling/.beans` (345 real beans, not mutated): `beans order <milestone> --first` moved a low-priority milestone ahead of higher-priority ones, and a temporary `--blocking` relation moved the blocker ahead of the blocked milestone, both within the same status group.
