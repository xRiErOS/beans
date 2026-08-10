---
# beans-d7y9
title: Fork nach /opt/homebrew/bin/beans installieren + cross-repo verifizieren
status: completed
type: task
created_at: 2026-07-21T18:39:26Z
updated_at: 2026-07-21T18:39:26Z
parent: beans-f1t4
---

Fork-Binary an die PATH-Stelle des Originals legen, Cross-Repo-Wirksamkeit pruefen.

## Akzeptanz
- [x] cp /tmp/beans-fork -> /opt/homebrew/bin/beans, chmod +x
- [x] which beans -> /opt/homebrew/bin/beans, version = fork
- [x] Aus fremdem Repo (beans-tui) list/roadmap lesen fremdes .beans/ ohne Fehler -> kein Umzug noetig

## Summary of Changes
Fork als echte Datei (kein Symlink) installiert. Cross-Repo verifiziert: beans-tui .beans (bt-apmy) laeuft. Ein globales Binary, Daten pro Repo schema-kompatibel.
