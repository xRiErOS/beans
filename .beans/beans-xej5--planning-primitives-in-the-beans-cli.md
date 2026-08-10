---
# beans-xej5
title: Planning primitives in the beans CLI
status: in-progress
type: milestone
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:39:28Z
---

beans should be able to carry a planning model that goes beyond title, status, type and priority. Today it silently drops every front matter key it does not know, and the one field it already declares for manual ordering is wired to no command at all.

The brief, the measured evidence and the numbered requirements R-01 to R-12 live in `docs/beans-planning-primitives/BRIEFING.md`. The model that triggered it is `~/dev/sproutling/docs/roadmap-release-planning/DESIGN.md`, which cannot be built until the first epic below ships.

## Scope

- **Custom front matter** — preserve unknown keys through the read-write cycle, expose them in `--json`, set and filter them from the CLI (R-01 to R-08).
- **Manual ordering** — connect the declared `Bean.Order` fractional index to `--sort`, to a placement command and to import (R-09 to R-12).
- Four pieces of related work already in the store are pulled in below: the rename follow-ups, the workflow CLI commands, `update --json` returning the resulting bean, and the roadmap bug that hides childless containers.

## Release risk

The bean data format is shared by at least five stores. An **older** binary writing a file that carries extra keys deletes them exactly as measured in the brief — the data loss then moves from the format into the version spread. Release belongs on a version bump, and `beans version` has to make the capability visible (R-08).
