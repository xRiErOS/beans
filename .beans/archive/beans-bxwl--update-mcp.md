---
# beans-bxwl
title: update cli info
status: completed
type: task
priority: critical
created_at: 2026-08-10T10:13:02Z
updated_at: 2026-08-10T12:13:21Z
parent: beans-xej5
---

PO-Nachtrag: Nach Abschluss der Arbeit muss überprüft werden, ob 'beans prime' sowie die '--help' Texte sauber implementiert sind. Eine CLI ohne hilfreiche Einführung und Hilfestellung verpasst ihren Wert.

## Nachtrag 2026-08-10 (Erik, Scope-Korrektur)

beans hat kein MCP. Scope ist ausschliesslich \`beans prime\` sowie \`beans --help\`/\`-h\` und \`beans <cmd> --help\` fuer die in beans-xej5 neu/geaenderten Commands (--set/--unset/--order, order-Subcommand, --where, --json bei version) pruefen und ggf. anpassen.

## Summary of Changes

`beans --help` and every `<cmd> --help` already documented --set/--unset/--order/--where/order via cobra flag descriptions (implementer coverage from beans-2ark/beans-zb0r was complete there). The real gap was `beans prime` (internal/commands/prompt.tmpl): the agent primer never mentioned custom front matter or manual ordering at all, so an agent reading only prime (the common case) had no way to discover either feature short of --help exploration or reading source.

Added two sections to prompt.tmpl (Custom Front Matter, Manual Ordering) plus CLI Commands examples for `beans order`/`--order`. Locked in with internal/commands/prime_test.go (renders the embedded template, asserts the new keywords are present) -- RED/GREEN + reverse-mutation verified.

Rebuilt and reinstalled globally (0.4.4-fork-2-g7b70af3); `beans prime` now surfaces both features.

Commit: 7b70af3
