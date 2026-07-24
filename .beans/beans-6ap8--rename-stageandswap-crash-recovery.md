---
# beans-6ap8
title: 'rename: stageAndSwap Crash-Recovery'
status: todo
type: task
priority: normal
created_at: 2026-07-24T13:15:09Z
updated_at: 2026-07-24T13:15:09Z
parent: beans-a29l
---

Aus beans-e040 D01 (T04-specs-review). stageAndSwap (pkg/beancore/rename.go) macht den Verzeichnis-Swap via zwei os.Rename. Dazwischen fehlt .beans/ transient komplett; ein Absturz/Kill exakt dort hinterlässt echte Daten nur unter .beans.bak-<ts>, und es gibt KEINE Waisen-Erkennung/-Reparatur beim Load(). Kein Go-stdlib-atomarer Cross-FS-Dir-Exchange verfügbar.

## Ziel
Startup-/Load-Check der .beans.bak-*-Waisen erkennt und repariert (oder mindestens warnt), sodass ein Absturz im Swap-Fenster nicht in stillem Datenverlust endet. Zusätzlich: Rollback-Zweig von stageAndSwap (rename.go:224-227, zweiter os.Rename failt) hat 0% Test-Coverage — via swapRename-Seam (existiert seit T05) einen Test ergänzen.

## Quelle
beans-e040 Epic-Body D01 + T04-specs-review (2026-07-24).
