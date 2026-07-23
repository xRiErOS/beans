---
# beans-f1t4
title: Fork-beans lokal live setzen (roadmap Feature-Nesting)
status: completed
type: epic
priority: normal
created_at: 2026-07-21T18:39:03Z
updated_at: 2026-07-21T18:42:34Z
---

Lokales beans-Binary aus Fork-Branch fix/beans-ti53-roadmap-nested-hierarchy (PR #207 hmans/beans) gebaut und offizielles Homebrew-Cask ersetzt. Fork bringt gleichberechtigtes Epic+Feature-Nesting in 'beans roadmap'. Ein globales Binary, Daten pro Repo schema-kompatibel -> kein Repo-Umzug.

## Summary of Changes
- Fork gebaut: 0.4.2-fork.ti53 (3419e8a), ldflags-Version-Stamp, ./cmd/beans
- brew uninstall --cask beans (Original-Symlink entfernt)
- Fork -> /opt/homebrew/bin/beans, cross-repo verifiziert
- Follow-ups (Brew-Overwrite-Guard, Rueckkehr-auf-Cask) bewusst vom PO uebernommen (scrapped)
