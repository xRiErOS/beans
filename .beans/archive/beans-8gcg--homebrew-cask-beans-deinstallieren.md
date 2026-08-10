---
# beans-8gcg
title: Homebrew-Cask beans deinstallieren
status: completed
type: task
created_at: 2026-07-21T18:39:26Z
updated_at: 2026-07-21T18:39:26Z
parent: beans-f1t4
---

Offizielles Homebrew-Cask-Binary entfernen, damit nur der Fork wirkt.

## Akzeptanz
- [x] brew uninstall --cask beans
- [x] Symlink /opt/homebrew/bin/beans entfernt

## Summary of Changes
brew uninstall --cask beans -> Symlink + Caskroom 0.4.2 gepurged. Download-Cache bleibt (schneller Reinstall moeglich).
