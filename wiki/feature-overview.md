# Feature Overview

This page maps what `beans` and its companion binaries actually do, grouped by capability rather than by command name. It is the starting point for understanding the whole toolset before diving into the per-command reference pages.

## Issue tracking on plain Markdown files

Every bean is a Markdown file with YAML front matter, stored under a `.beans` directory inside the project it tracks. `beans create` writes a new bean, `beans show` renders one, `beans update` changes its properties, and `beans delete` removes it outright; `beans complete`, `beans start`, and `beans scrap` move a bean through its status lifecycle without needing to touch the file by hand. Because beans are files next to the code they describe, they read, diff, and merge like any other text in version control, and an editor or a coding agent can open one directly with no separate database to query.

## A configurable hierarchy of types

Beans are organized by type, and each type occupies a hierarchy rank: rank 1 is the topmost container, higher-numbered ranks nest under it, and the leaf rank (the highest number) holds the actual units of work. The built-in default hierarchy is `milestone` → `epic` → `feature`, with `bug` and `task` as leaves, but this table is fully configurable per project through `.beans.yml`, either by overriding individual built-in entries or by writing an exclusive type table that replaces the defaults entirely. `beans init --profile` writes one of four ready-made hierarchy shapes at project creation time; see [Project Profiles](project-profiles.md) for what each profile contains and when to pick it.

## Roadmap and progress reporting

`beans roadmap` generates a Markdown roadmap document from the current milestones, epics, and their child items, either for the whole project or scoped to one item's subtree. `beans milestones` lists every top-container bean with a rollup of how many of its descendants are complete, and `beans progress` reports counts by status across the whole workspace or a single parent's subtree, including a percent-complete figure. A type can opt out of both `beans roadmap` and `beans milestones` by setting `roadmap: false`, which is how the `bucket` type in the `simple` and `complex` profiles stays a genuine parking lot instead of cluttering release-facing views.

## Querying and filtering

`beans list` and `beans next` share the same filtering vocabulary — `--type`, `--tag`, `--parent`, and `--sort` — so a query can move between "show me everything matching this" and "show me the single highest-priority ready item" without rewriting it. For anything the built-in flags cannot express, `beans graphql` runs an arbitrary GraphQL query or mutation directly against the beans data, which is the same mechanism a coding agent uses to pull exactly the fields it needs while keeping token use low.

## Structure and relationships

Beans can be nested (a feature under an epic, a task under a feature), reordered relative to their siblings with `beans order`, and connected through blocking and parent/child links. `beans graph` renders those relationships as Graphviz DOT, a terminal edge list, or JSON. `beans tag` adds or removes tags for cross-cutting grouping that does not depend on the type hierarchy. `beans rename` covers three distinct operations behind one command: renaming a bean's slug, renaming a single bean's ID with all references to it updated, and renaming the project-wide ID prefix across every bean at once.

## Validation and integrity

`beans check` validates both configuration and data integrity in one pass: that colors and the default type resolve correctly, that no bean carries a type the configuration no longer defines, that links do not point at beans that do not exist, that no bean links to itself, and that no cycle exists in the blocking or parent relationships. `--fix` automatically removes broken links and self-references; cycles always require manual intervention because there is no safe automatic resolution for one. `--strict` turns policy warnings into failures for use in a pre-commit hook or CI check.

## Completion policy: required fields

A project can require specific front-matter fields to be filled in before a bean may enter a given status, configured under `beans.require_fields_on` in `.beans.yml`. The most common use is gating the `completed` status on a `commit` field, so `beans complete` refuses to finish a bean until the commit that resolved it has been recorded; `beans prime` reflects whatever policy a project has configured so an agent knows the rule before it hits the refusal.

## Priming coding agents

`beans prime` prints a ready-to-paste prompt that tells an AI coding agent how to use the `beans` CLI on this specific project: its configured types and their hierarchy ranks, its statuses and priorities, and any required-fields completion policy currently in force. Because the prompt is generated from the live configuration rather than a static document, it stays accurate after a project changes its type table or its policy without anyone having to hand-edit onboarding instructions. See [Agent Integration](agent-integration.md) for the full agent workflow this supports.

## Git-aware worktrees

Beans configuration includes a `worktree` section covering the base ref new work branches from, a setup command to run after a worktree is created, a run command exposed as a workspace action, and an integration strategy for merging finished work back — either squash-merging locally or pushing a branch and opening a pull request. This lets a coding agent spin up isolated git worktrees for parallel work items without every agent needing its own bespoke branching convention.

## Interfaces beyond the CLI

The core `beans` binary is deliberately narrow; the terminal UI and the web UI ship as separate binaries, `beans-tui` and `beans-serve`, so installing the tracker does not require pulling in either interface. See [TUI Companion](tui-companion.md) for the terminal UI and [Web UI and API](web-ui-and-api.md) for the web UI and its HTTP surface.

## What is deliberately out of scope

`beans` stores issues, not time tracking, not CI pipeline state, and not a general project wiki; its roadmap and progress views only ever reflect the type hierarchy and status table a project has configured, so a type marked `roadmap: false` — such as a profile's `bucket` type — is excluded from those views by design rather than by omission. The tool assumes a single-repository, file-based store per project; cross-project aggregation is not a built-in feature of the core CLI.

## Related documentation

- [Project Profiles](project-profiles.md)
- [Configuration Reference](configuration.md)
- [Command Reference: Project Setup](commands/project-setup.md)
- [Command Reference: Inspection and Search](commands/inspection-and-search.md)
- [Command Reference: Lifecycle](commands/lifecycle.md)
- [Command Reference: Organization and Relations](commands/organization-and-relations.md)
- [Command Reference: Planning and Reporting](commands/planning-and-reporting.md)
- [Command Reference: Querying and Automation](commands/querying-and-automation.md)
- [Command Reference: Validation and Maintenance](commands/validation-and-maintenance.md)
- [Command Reference: Separate Binaries](commands/separate-binaries.md)
- [Data Model](data-model.md)
- [Agent Integration](agent-integration.md)
- [TUI Companion](tui-companion.md)
- [Web UI and API](web-ui-and-api.md)
