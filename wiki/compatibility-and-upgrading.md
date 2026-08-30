# Compatibility and Upgrading

This page describes how to check whether a new `beans` binary is compatible with an existing project's data, and what to do before and after upgrading. For obtaining a binary, see [Installation](installation.md).

## Check the version you are running

Run `beans version` to print the human-readable version, commit, and build date, or `beans version --json` for the same information as machine-readable JSON.

The JSON output also reports a `custom_front_matter` boolean, which tells you whether that specific binary preserves unknown ("custom") YAML front matter keys on beans across a read/write round trip instead of silently dropping them; treat a mismatch on this field between two binaries as a real behavioral difference worth checking against your own beans before relying on it.

## What "compatible" means here

Bean data is plain Markdown with YAML front matter, stored as ordinary files under a `.beans` directory in your project, so most of what "compatibility" means in practice is whether a given binary reads and writes that front matter, that directory layout, and your `.beans.yml` configuration the way you expect.

Because the on-disk format is just text files, most upgrades are compatible in the sense that older data continues to parse; the exceptions are changes that add, rename, or reinterpret a front matter field, tighten validation in `beans check`, or change how paths and configuration are resolved.

Do not assume a specific past or future release is compatible with your data purely from a version number; verify it against your own project as described below, since this document cannot make version-specific compatibility guarantees on your behalf.

## Before upgrading: back up your data

Because `.beans` is meant to be tracked in the same version control system as your code, commit any pending bean changes (or otherwise snapshot the `.beans` directory) before installing a new binary, so that you can diff or revert if the new binary behaves unexpectedly.

Do not rely on any binary's internal safety mechanisms as a substitute for your own backup; those mechanisms exist to protect against interrupted internal operations, not to replace normal version control hygiene.

## Review changes before upgrading

The fork has no published releases yet, so review its [commit history](https://github.com/xRiErOS/beans/commits/main) before replacing a binary, paying particular attention to changes marked as breaking. Once the [Releases page](https://github.com/xRiErOS/beans/releases) carries entries, prefer their release notes as the upgrade fixpoint.

The project uses Conventional Commit prefixes such as `feat:`, `fix:`, `perf:`, and `refactor:`; a `!` in the commit type or scope marks a breaking change, and the generated release notes group those entries under their own heading.

If a change affects the bean file format itself, prefer an explicit migration step over guessing; where no automated migration is provided, edit the affected front matter fields by hand across your `.beans` directory before relying on the new binary for further writes.

## After upgrading: validate your project

Run `beans check` against your project to validate configuration and bean integrity, including broken links, self-references, and circular dependencies; `beans check --strict` additionally treats policy warnings as failures, and `beans check --fix` can automatically remove broken links and self-references (cycles still require manual resolution).

Run `beans path` to confirm which `.beans` directory the binary resolved. Precedence is `--beans-path`, then a found or explicitly named `.beans.yml` and its anchored path, then `BEANS_PATH` only when no config was found, then the default `.beans` directory.

Without `beans.anchor`, a secondary worktree resolves `beans.path` relative to its own `.beans.yml` location and therefore uses a separate store. Set `beans.anchor: repo-root` to opt into sharing the main worktree's store. The fork also auto-renames legacy `worktrees/` and `conversations/` directories to `.worktrees/` and `.conversations/` when loading a project that still has the old names.

## Verifying a specific command against your data

For any command that mutates beans, verify it first against a disposable copy of your data rather than your real store, for example:

```
beans --beans-path /tmp/beans-upgrade-check/.beans init
beans --beans-path /tmp/beans-upgrade-check/.beans check
```

The explicit `--beans-path` flag overrides both your project configuration and the `BEANS_PATH` environment variable, which makes it the safest way to point any exploratory or destructive command at a throwaway directory instead of your real project.

## Related documentation

- [Fork Lineage](fork-lineage.md)
