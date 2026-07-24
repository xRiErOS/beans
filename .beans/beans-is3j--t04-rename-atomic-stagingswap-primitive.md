---
# beans-is3j
title: 'T04 rename: atomic staging+swap primitive'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T11:51:44Z
parent: beans-e040
blocked_by:
    - beans-eeqn
---

stageAndSwap + copyTree. Plan Task 4.

## Objective
Als Nutzer will ich, dass ein kaskadierender Rename (viele Dateien) atomar ist — entweder alle Änderungen greifen oder keine — damit ein Abbruch nie einen halb-migrierten `.beans/`-Zustand hinterlässt. Kein Atomic-Write-Helfer existiert heute.

## EARS
- WHEN `stageAndSwap(writes, removes)` aufgerufen wird, THE SYSTEM SHALL einen vollständigen neuen `.beans/`-Baum in einem Temp-Sibling-Dir bauen (Klon + removes + writes, Keys `.beans`-relativ), dann atomar per Dir-Swap einschwenken (alt → `.bak-*`, neu → `.beans`, Backup löschen).
- IF vor dem Swap ein Fehler auftritt, THE SYSTEM SHALL den Original-`.beans/`-Baum unberührt lassen und das Staging-Dir entfernen.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestStageAndSwap` GRÜN — beide Subtests (Pre-Swap-Failure lässt Original intakt + räumt Siblings; writes/removes werden korrekt angewendet, unberührte Dateien bleiben).
- SC-002: Nicht-bean-Dateien (z.B. `archive/`) im `.beans/` überleben den Swap.

## Betroffene Pfade
- `pkg/beancore/rename.go` (append copyTree + stageAndSwap), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 4.

## Summary of Changes

- [x] `copyTree(src, dst string, skip map[string]bool) error` — rekursiver Klon per `filepath.Walk`, skip-Set (Datei oder ganzer Subtree via `SkipDir`) für `removes`.
- [x] `func (c *Core) stageAndSwap(writes map[string][]byte, removes []string) error` — Staging-Sibling via `os.MkdirTemp(repo, ".beans-staging-*")`, Klon + removes(skip) + writes, dann atomarer Zwei-Rename-Swap (`c.root` → `.beans.bak-<ts>`, staging → `c.root`, Backup gelöscht). Bei Fehler vor dem Swap: Original unberührt, `defer` räumt Staging-Dir.
- [x] SC-001: `TestStageAndSwap_atomicOnPreSwapFailure` + `TestStageAndSwap_appliesWritesAndRemoves` — table-driven, RED→GREEN verifiziert.
- [x] SC-002: eigener Subtest `non-bean file (e.g. archive/) survives the swap` deckt Nicht-bean-Dateien ab.
- Deviation vom Plan-Snippet (gerechtfertigt): Plan-Skizze nutzt zwei separate `Test...`-Funktionen mit Einzel-Asserts; hier table-driven umgesetzt (Repo-Konvention, siehe bestehende `TestRewriteRefs`/`TestApplyRenameSlug_*` im selben File) und um einen dritten "no writes/removes"-Fall ergänzt. Kein Verhalten geändert, nur Testform.
- Kein Scope-Creep: `applyRenameCascade` bleibt Stub (T05/T06), Cascade-/Prefix-Logik nicht angerührt.

**Validation:**
```
go test ./pkg/beancore/ -run TestStageAndSwap -v   → PASS (alle Subtests)
go test ./...                                       → alle Pakete ok
go vet ./...                                         → clean
go build ./cmd/beans                                 → OK
```
