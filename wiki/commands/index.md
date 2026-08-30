# Command reference

This index maps every top-level `beans` command to its canonical reference page. Run `beans <command> --help` for the complete option list of the installed version.

## Project setup

- [`init`](project-setup.md#beans-init) — initialize a repository and optionally expand a project profile.
- [`path`](project-setup.md#beans-path) — print the resolved beans data directory.
- [`prime`](project-setup.md#beans-prime) — emit project-specific instructions for coding agents.
- [`version`](project-setup.md#beans-version) — print build and version information.
- [`help`](project-setup.md#beans-help) — open command help.
- [`completion`](project-setup.md#beans-completion) — generate shell completion scripts.

## Inspection and search

- [`list`](inspection-and-search.md#beans-list-alias-ls) — filter, sort, render, or export beans.
- [`show`](inspection-and-search.md#beans-show) — display one bean and its content.
- [`next`](inspection-and-search.md#beans-next) — select the highest-priority ready bean.

## Lifecycle

- [`create`](lifecycle.md#beans-create-aliases-c-new) — create one bean with type, status, priority, body, tags, and relationships.
- [`start`](lifecycle.md#beans-start) — mark one or more beans in progress.
- [`complete`](lifecycle.md#beans-complete) — complete beans under the configured policies.
- [`scrap`](lifecycle.md#beans-scrap) — mark work as intentionally abandoned.
- [`archive`](lifecycle.md#beans-archive) — move completed or scrapped beans into project memory.
- [`delete`](lifecycle.md#beans-delete-alias-rm) — permanently remove beans after confirmation or explicit force.

## Organization and relationships

- [`update`](organization-and-relations.md#beans-update) — change fields, content, and relationships.
- [`tag`](organization-and-relations.md#beans-tag) — add or remove tags in batches.
- [`order`](organization-and-relations.md#beans-order) — position a bean among its siblings.
- [`rename`](organization-and-relations.md#beans-rename) — rename a slug, ID, or project prefix while updating references.

## Planning and reporting

- [`roadmap`](planning-and-reporting.md#beans-roadmap) — render a scoped project roadmap for terminals or Markdown.
- [`milestones`](planning-and-reporting.md#beans-milestones) — summarize top-level planning containers and descendant progress.
- [`progress`](planning-and-reporting.md#beans-progress) — summarize project or subtree status.
- [`graph`](planning-and-reporting.md#beans-graph) — render parent and blocking relationships as DOT, ASCII, or JSON.

## Querying and automation

- [`graphql`](querying-and-automation.md#beans-graphql) — execute precise GraphQL queries and mutations.
- [JSON output and error contracts](querying-and-automation.md#json-output-contract) — integrate beans with scripts and coding agents.

## Validation and maintenance

- [`check`](validation-and-maintenance.md#beans-check) — validate configuration, files, links, policies, and optional commit references.

## Separate applications

- [`beans-serve`](separate-binaries.md#beans-serve) — run the browser workspace and GraphQL service.
- [`beans-tui`](separate-binaries.md#beans-tui) — run the retained original terminal UI as a separate binary.
- [`bt`](../tui-companion.md#layer-3-bt-the-actively-developed-po-cockpit) — use the separately maintained Product Owner cockpit.

## Related documentation

- [Feature overview](../feature-overview.md)
- [Quick start](../quick-start.md)
- [Configuration reference](../configuration.md)
- [Agent integration](../agent-integration.md)
- [Troubleshooting](../troubleshooting.md)
