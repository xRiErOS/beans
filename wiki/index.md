# beans documentation

This directory is the canonical documentation for the actively maintained xRiErOS/beans fork. Start with the quick path below, then open only the reference page needed for the current task.

## Start here

- [Installation](installation.md) — build and install the current fork and its three binaries.
- [Quick start](quick-start.md) — initialize a project, create work, and render the first roadmap.
- [Feature overview](feature-overview.md) — understand the product surface without reading command help.
- [Project profiles](project-profiles.md) — choose between `classic`, `todo`, `simple`, and `complex`.

## Use beans with coding agents

- [Agent integration](agent-integration.md) — prime agents, automate session setup, and choose safe machine-readable interfaces.
- [Querying and automation](commands/querying-and-automation.md) — use GraphQL, JSON output, and structured errors.
- [Data model and file format](data-model.md) — understand the Markdown files shared by humans, scripts, and agents.

## Configure a project

- [Configuration reference](configuration.md) — configure stores, profiles, types, statuses, priorities, display, and completion policies.
- [Compatibility and upgrading](compatibility-and-upgrading.md) — assess changes and migrate deliberately.
- [Fork lineage](fork-lineage.md) — understand the relationship to the original hmans/beans project.

## Browse every command

- [Complete command index](commands/index.md) — map every top-level command to its canonical page.
- [Project setup](commands/project-setup.md) — `init`, `path`, `prime`, `version`, `help`, and `completion`.
- [Inspection and search](commands/inspection-and-search.md) — `list`, `show`, and `next`.
- [Lifecycle](commands/lifecycle.md) — `create`, `start`, `complete`, `scrap`, `archive`, and `delete`.
- [Organization and relationships](commands/organization-and-relations.md) — `update`, `tag`, `order`, and `rename`.
- [Planning and reporting](commands/planning-and-reporting.md) — `roadmap`, `milestones`, `progress`, and `graph`.
- [Querying and automation](commands/querying-and-automation.md) — `graphql`, JSON output, and structured errors.
- [Validation and maintenance](commands/validation-and-maintenance.md) — `check`, policies, and integrity checks.
- [Separate applications](commands/separate-binaries.md) — `beans-serve` and the retained `beans-tui` binary.

## Use visual interfaces

- [Web UI and API](web-ui-and-api.md) — run the planning workspace, board, and GraphQL service.
- [TUI companion](tui-companion.md) — distinguish the retained original TUI from the actively developed `bt` companion.

## Solve problems

- [Troubleshooting](troubleshooting.md) — diagnose store discovery, ready work, policies, configuration, and server startup.

## Documentation contract

`beans --help`, each `beans <command> --help`, and `beans prime` are the primary behavioral sources for this documentation. Release-specific pages and screenshots must be regenerated from the exact public build they describe.

Each detailed subject has one canonical page. The repository README presents the product and links here instead of duplicating the reference material.
