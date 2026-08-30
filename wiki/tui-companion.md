# TUI Companion

This page explains the three separate layers of terminal tooling around this fork's beans data: the `beans` CLI itself, the retained `beans-tui` compatibility binary, and the actively developed `bt` companion project. Read this before assuming any terminal UI is the fork's primary interface.

## Layer 1: `beans` CLI is the source of truth

Every terminal UI described on this page is a client of the `beans` CLI and its underlying `.beans` data directory; none of them hold state of their own. If a terminal UI and the `beans` CLI ever disagree about a bean's fields, the `.beans` files on disk (as read by `beans show`/`beans list`) are authoritative, not the UI's cached rendering. Command and data details live in [`commands/separate-binaries.md`](commands/separate-binaries.md), [`feature-overview.md`](feature-overview.md), and [`configuration.md`](configuration.md).

## Layer 2: `beans-tui`, retained for compatibility

`beans-tui` is a separate binary built from `cmd/beans-tui` in this repository; running it with no arguments opens the same `tui` view that used to ship inside the `beans` binary itself. In this fork, the `tui` subcommand has been removed from the main `beans` binary: running `beans tui` now prints `The "tui" command has moved to a separate binary: beans-tui` and exits with a non-zero status, so `beans-tui` is required to use that view at all. This binary is kept for compatibility with existing installs and muscle memory, not as the fork's actively developed interface — it receives no new features here. Build and install instructions for `beans-tui` are in [`commands/separate-binaries.md`](commands/separate-binaries.md); general setup is in [`installation.md`](installation.md).

## Layer 3: `bt`, the actively developed PO cockpit

The actively developed terminal UI for this fork's data model is a separate project, [xRiErOS/beans-tui](https://github.com/xRiErOS/beans-tui), whose binary is invoked as `bt`, not `beans-tui`. The project name and the compatibility binary's filename collide by coincidence; the binary names `bt` versus `beans-tui` are what distinguish them at the command line. `bt` is a keyboard-first, mouse-friendly Product-Owner cockpit built as a port of a prior DevDash TUI onto the beans data layer, and it is licensed Apache-2.0 to match this fork, but it ships and versions independently with its own CHANGELOG and releases. `bt` reads and writes the same `.beans` directory that `beans` and `beans-tui` use, so it requires no separate data migration to try.

Capabilities documented in its own README, summarized without duplicating it in full: a live-reloading Tree (Milestones → Epics → Tasks) with a master-detail Accordion for Meta/Body/Relations/History; a flat, sortable Backlog view for unscheduled work; a Command-Center (`ctrl+k` / `K`) for fuzzy actions and bean search; full mutation support (create/edit/delete, Status/Type/Priority menu, Tag/Parent/Blocking pickers, `$EDITOR` body edit) with ETag-conflict handling; local live-filter search plus full-text search via Bleve from three characters; a facet filter for Status/Type/Priority/Tag shared across Tree and Backlog; a fullscreen single-pane mode with its own relations-jump history stack; a Lobby/Repo-Picker for switching between multiple beans repos, each with its own file-watcher lifecycle; and mouse support for wheel, click, and double-click. Review state in `bt` is shown only as ordinary tag visibility on the `to-review`/`accepted`/`rejected` trio — review itself happens in chat with an agent or PO, not inside the TUI.

## Choosing between the two TUIs

Use `beans-tui` only if you specifically need the original bundled view and do not need any capability listed above. For day-to-day Product-Owner work on beans data, install `bt` from [xRiErOS/beans-tui](https://github.com/xRiErOS/beans-tui) instead; it is where active TUI development happens in this fork's ecosystem, and `beans-tui` will not receive equivalent features. Neither `beans-tui` nor `bt` is endorsed by or affiliated with the original [hmans/beans](https://github.com/hmans/beans) project that this fork's CLI and data model descend from; see [`fork-lineage.md`](fork-lineage.md) for how this fork positions itself relative to that upstream.

## Related documentation

- [fork-lineage.md](fork-lineage.md)
- [installation.md](installation.md)
- [feature-overview.md](feature-overview.md)
- [project-profiles.md](project-profiles.md)
- [commands/separate-binaries.md](commands/separate-binaries.md)
- [configuration.md](configuration.md)
- [web-ui-and-api.md](web-ui-and-api.md)
