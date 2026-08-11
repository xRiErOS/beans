---
type: DesignSpec
title: "DESIGN — roadmap-scope"
description: "Scope beans roadmap to a single milestone, epic, or feature via an optional positional bean ID"
tags:
  - tpic
  - roadmap-scope
timestamp: 2026-08-11T11:58:39Z
---

# roadmap-scope Design

## Problem (Ist)

`beans roadmap` (`internal/commands/roadmap.go`) always walks the entire bean graph and renders every milestone plus an "Unscheduled" section for everything else. There is no way to render just the subtree relevant to one milestone or epic currently being worked on. On a repository with many milestones this makes the roadmap output unwieldy for the common "I'm focused on milestone X" case.

## Goal

Add an optional positional bean ID argument to `beans roadmap` that scopes the output to that bean's subtree: `beans roadmap <id>`. `<id>` may resolve to a `milestone`, `epic`, or `feature` (features and epics can both be children of a milestone, so both are valid scope roots). `beans roadmap` with no argument is unchanged.

## CLI Surface

```
beans roadmap              # unchanged: full roadmap
beans roadmap <id>         # new: roadmap rooted at this bean
```

- `<id>` is resolved via `core.NormalizeID` (same short-ID resolution as `beans show`).
- Unknown ID: error `unknown bean: <id>`.
- ID resolves to a type other than milestone/epic/feature: error naming the allowed types.
- `--status` / `--no-status` filter the milestone selection in the unscoped view; they are meaningless once a single root is chosen. Combining them with `<id>` is a **usage error** (Cobra-level validation), not a silent no-op.
- `--include-done`, `--json`, `--no-links`, `--link-prefix` are unchanged and still apply within the scoped subtree.
- An empty scope (e.g. everything under the root is done and `--include-done` was not passed) still renders the root container itself, just with no children — so the caller can tell "ID found, nothing visible" apart from "ID not found."

## Data Model (`roadmap.go`)

`buildRoadmap` gains a `rootID string` parameter.

- **Root = milestone**: `milestones` is reduced to the single resolved bean (reusing the existing `buildMilestoneGroup` path unchanged); `Unscheduled` is not built at all — the concept doesn't apply once scoped to one milestone. No renderer changes needed for this case; it's structurally identical to today's output with one milestone in the list.
- **Root = epic or feature**: new field `roadmapData.Root *rootGroup`, where

  ```go
  type rootGroup struct {
      Epic    *epicGroup    `json:"epic,omitempty"`
      Feature *featureGroup `json:"feature,omitempty"`
  }
  ```

  exactly one of `Epic`/`Feature` is populated (using the existing `buildEpicGroup`/`buildFeatureGroup`). `Milestones` and `Unscheduled` stay nil in this case.

## Rendering

- `roadmap.tmpl`: wrap the existing body in `{{if .Root}}...{{else}}<existing body>{{end}}`. The `.Root` branch calls the already-defined `epicGroup`/`featureGroup` named templates directly — no new template logic.
- `roadmap_pretty.go` (`renderRoadmapPretty`): a new branch at the top — when `data.Root != nil`, print a heading for the root bean and call `renderRoadmapEpicGroup`/`renderRoadmapFeatureGroup` directly at indent 0, skipping the "■ Milestone" / "No Milestone" framing used by the unscoped view.
- JSON output: `roadmapData` simply gains the `root` field; existing consumers reading `milestones`/`unscheduled` see those empty/absent when scoped.

## Out of Scope

- No breadcrumb showing an epic/feature's ancestor milestone in the scoped view.
- No scoping to arbitrary leaf bean types (task, bug, ...) — only milestone/epic/feature are valid roots.
- No change to the unscoped (`beans roadmap` with no argument) output or its existing flags.
