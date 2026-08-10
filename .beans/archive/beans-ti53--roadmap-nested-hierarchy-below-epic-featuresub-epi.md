---
# beans-ti53
title: 'roadmap: nested hierarchy below Epic (Feature/sub-Epic) not resolved, leafs vanish'
status: completed
type: bug
priority: high
tags:
    - roadmap
created_at: 2026-07-17T20:15:59Z
updated_at: 2026-07-17T22:15:17Z
parent: beans-en7i
---

buildMilestoneGroup() in internal/commands/roadmap.go walks only one level of children per epic (filterChildren, non-recursive). Any bean type is a valid --parent target, but roadmap.go only recognizes "milestone" and "epic" as container types. Result: Milestone -> Epic -> Feature -> Leaf (bug/task) hierarchies lose the leafs entirely, and epics whose only open descendants sit below an intermediate node get dropped from output (len(epicItems) == 0 check). Repro: any repo with a milestone whose epic has a feature child that itself has open bug/task children -- those never render, and the epic vanishes if it has no *direct* leaf child.

Scope (per plan): full fix -- extend the roadmap data model (epicGroup) to support nested Feature groups, not just flat leaf lists, and update roadmap.tmpl to render the extra level. Ship as upstream PR to hmans/beans.

Plan: docs/plans/<this-bean-id>-roadmap-nested-hierarchy/plan.md (local-only, not part of PR diff).

## Erledigt 2026-07-18
Fix umgesetzt (8 Commits f0f31fb..3419e8a), PO-Review 6/6 accepted, PR offen: https://github.com/hmans/beans/pull/207
