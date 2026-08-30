# Fork Lineage

This page explains where this codebase comes from and how this fork relates to the original project. It exists so that anyone evaluating, contributing to, or reporting issues against this fork understands the attribution and continuity that apply to it.

## Origin

This project is a fork of [hmans/beans](https://github.com/hmans/beans), an agentic-first issue tracker created by Hendrik Mans that stores tasks as Markdown files alongside a project's code.

The original idea, the core data model (Markdown beans with YAML front matter, stored in a `.beans` directory), the initial command surface, and the initial GraphQL query engine all originate from that upstream project, and this fork is grateful for that foundation.

## License and attribution

Both the upstream project and this fork are distributed under the Apache License, Version 2.0; the license text is unchanged and is included in this repository's `LICENSE` file.

Apache-2.0 permits forking, modification, and independent redistribution, provided the license and any existing copyright and attribution notices are preserved, which this fork does.

Any code, documentation, or design carried over from `hmans/beans` remains attributed to that project; new work added in this fork is layered on top of it under the same license.

## Why this fork exists

This fork, maintained at [xRiErOS/beans](https://github.com/xRiErOS/beans), is an independent, actively maintained continuation of the beans codebase, developed outside the upstream `hmans/beans` repository.

It is not a rebrand or a hostile fork; it exists so that development, bug fixes, and new features can continue on an independent schedule and roadmap, separate from upstream's own release cadence.

This fork does not claim to speak for or represent the upstream project, and it does not assert that the upstream project has been discontinued or abandoned; users who need the canonical upstream state should consult [hmans/beans](https://github.com/hmans/beans) directly.

## What has changed so far

Development in this fork has focused on hardening existing behavior: fixing data-integrity edge cases in bean storage and rename operations, correcting path-resolution precedence between the `--beans-path` flag, project configuration, and the `BEANS_PATH` environment variable, and adding regression coverage around those fixes.

The on-disk data format (Markdown files with YAML front matter under `.beans`) and the overall command-line surface remain consistent with the upstream design described above; see [Compatibility and Upgrading](compatibility-and-upgrading.md) for how to verify compatibility for a specific version.

Commit history in this fork follows the Conventional Commits convention (`feat:`, `fix:`, `perf:`, `refactor:`, with a `!` marker for breaking changes), which is also what upstream uses and what this fork's own release changelog is generated from.

## How to reference this fork

When filing issues, requesting features, or citing behavior, reference the fork's own repository, [xRiErOS/beans](https://github.com/xRiErOS/beans), rather than the upstream repository, so that reports land with the maintainers who are actually responsible for this codebase.

When citing the underlying design or crediting the original author, reference [hmans/beans](https://github.com/hmans/beans) as the source of the original concept and implementation.

## Related documentation

- [Compatibility and Upgrading](compatibility-and-upgrading.md)
