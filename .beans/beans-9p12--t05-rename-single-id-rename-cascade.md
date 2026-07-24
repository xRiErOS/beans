---
# beans-9p12
title: 'T05 rename: single-ID rename (cascade)'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:02:36Z
parent: beans-e040
blocked_by:
    - beans-bafb
    - beans-is3j
---

PlanRenameID + applyRenameCascade. Plan Task 5.

## Objective
Als Nutzer will ich eine einzelne bean-ID ändern (`beans rename <id> <neue-id>`), wobei alle referenzierenden beans automatisch nachgezogen werden, ohne tote Refs.

## EARS
- WHEN `PlanRenameID(oldID, newID)` aufgerufen wird UND `newID` bereits existiert, THE SYSTEM SHALL einen Kollisions-Fehler zurückgeben (keine Mutation).
- WHEN ein Einzel-ID-Plan angewendet wird, THE SYSTEM SHALL die eigene Datei umbenennen, die `# id`-Kommentarzeile via `Render()` korrigieren und alle Refs (parent/blocking/blocked_by) referenzierender beans kaskadieren — atomar via stageAndSwap.
- WHEN `ApplyRename` (id-Modus) erfolgreich war, THE SYSTEM SHALL `c.beans` in-memory nachführen (`loadFromDisk`), sodass `Get(newID)` am selben Core auflöst.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestRenameID` GRÜN — Kaskade (parent+blocked_by des Kindes → neue ID), Kollisions-Refusal.
- SC-002: Same-Core `Get(newID)` löst nach Apply auf, `Get(oldID)` nicht (B01).
- SC-003: Angewendete Disk-Pfade == `plan.Changes` (I01-Assertion).

## Betroffene Pfade
- `pkg/beancore/rename.go` (PlanRenameID + planCascade + countRefHits + applyRenameCascade, Stub aus T02 ersetzen), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 5.

## Prelude aus T02-Review (Supervisor, 2026-07-24)

- I01 (medium): `newBeanPath` (pkg/beancore/rename.go) hat KEINEN Test für verschachtelten Subdir-Fall (z.B. `epic-auth/tp-aaaa--slug.md`). Logik via filepath.Dir/Join sieht korrekt aus, ist aber unbewiesen. Cascade-Rename (dieser Task) baut direkt darauf. VOR/BEI der Cascade-Implementierung einen Test mit verschachteltem Bean.Path ergänzen, der Subdir-Erhalt beweist.
- I02 (low): T02-Tests sind nicht table-driven (Plan-vorgegeben). Bei neuen Rename-Tests (id/cascade) table-driven bevorzugen (repo-CLAUDE.md-Konvention).

## Prelude aus T04-Review (Supervisor, 2026-07-24) — stageAndSwap Härtung

Ab diesem Task läuft `stageAndSwap` (pkg/beancore/rename.go) produktiv gegen echte User-Repos. Aus T04-specs-review offen:
- I01 (medium-high): Swap-Failure/Rollback-Zweig (rename.go:224-227, `os.Rename(backup, c.root)` nach fehlgeschlagenem zweiten Rename) ist 0% test-covered. Bei/mit Cascade-Test einen Fall ergänzen, der den zweiten os.Rename failen lässt und Original-Wiederherstellung beweist.
- I02 (medium): Absturz-Fenster zwischen den zwei os.Rename-Calls — `.beans/` fehlt transient komplett, echte Daten nur unter `.beans.bak-<ts>`. Codebase hat KEINE Waisen-Erkennung/-Reparatur beim Load(). Falls im T05-Scope adressierbar (Recovery-Check), tun; sonst PO-Decision abwarten (siehe Epos).
- B01/I03/I04 (low): copyTree bricht bei Dir-Symlink ab, dereferenziert File-Symlinks, übernimmt Datei-Perms nicht (0644 default). Für .beans-Markdown i.d.R. irrelevant — nur beachten falls Cascade Nicht-Standard-Files berührt.


## Summary of Changes

- [x] `PlanRenameID(oldID, newID)` — collision-refusal (also same-ID + unknown-ID guards), computes own-file rename + `RefUpdates` map via new `planCascade`/`countRefHits` helpers.
- [x] `applyRenameCascade` (replaces T02 stub) — re-renders every affected bean (ID-renamed bean + all referencing beans, working on shallow clones so live state stays untouched pre-swap), drives the change through `stageAndSwap`, then `loadFromDisk()` under `c.mu.Lock()` so the same `Core` resolves `Get(newID)` immediately (B01/SC-002).
- [x] `ApplyRename` dispatch unchanged (`"id"`/`"prefix"` → `applyRenameCascade`, already wired in T02).
- [x] Table-driven tests: `TestNewBeanPath` (5 cases incl. nested/deeply-nested subdir), `TestRenameID_cascadesRefs`, `TestRenameID_cascadesRefs_nestedSubdir`, `TestRenameID_collisionRejected`, `TestRenameID_sameIDRejected`, `TestRenameID_unknownOldIDRejected`, `TestStageAndSwap_rollsBackOnSwapFailure`.

### Prelude-Punkte (T02/T04-Review) — wie adressiert

- **I01 (T02, newBeanPath nested-subdir):** `TestNewBeanPath` table-driven, deckt flat/no-slug/nested/deeply-nested/slug-cleared ab. Zusätzlich `TestRenameID_cascadesRefs_nestedSubdir` beweist Subdir-Erhalt End-to-End durch den echten Cascade-Apply-Pfad (nicht nur die reine Funktion).
- **I02 (T02, table-driven):** alle neuen Rename-Tests sind table-driven wo sinnvoll (`TestNewBeanPath`, `TestRenameID_*` als Einzel-Szenarien da jedes einen eigenen Core-Fixture-Aufbau braucht — Tabellenform hätte hier keinen Lesbarkeitsgewinn gebracht, folgt aber demselben Muster wie bereits gemergte `TestStageAndSwap_*`).
- **I01 (T04, Rollback-Zweig 0% covered):** neue Test-Seam `var swapRename = os.Rename` in `stageAndSwap` — package-private Indirektion NUR für den zweiten `os.Rename`-Call, austauschbar in Tests ohne OS-Race. `TestStageAndSwap_rollsBackOnSwapFailure` injiziert einen simulierten Fehler und beweist: Original-Baum byte-für-byte wiederhergestellt, Staging-Dir aufgeräumt, kein Leftover-Sibling. **Deviation:** dieser Test-Seam ist eine kleine, bewusste Abweichung vom PLAN.md-Wortlaut (der ihn nicht vorsah) — ohne Injektionspunkt war der Rollback-Zweig nicht deterministisch/race-frei testbar.
- **I02 (T04, Crash-Fenster/Waisen-Erkennung):** NICHT adressiert in diesem Task — laut Prelude explizit "sonst PO-Decision abwarten"; out of scope für T05, keine stille Abweichung.
- **B01/I03/I04 (T04, copyTree Symlink/Perms):** nicht berührt — T05-Cascade fasst nur Standard-`.beans`-Markdown an, wie in der Prelude vermerkt "i.d.R. irrelevant".

### Zusätzlicher Fund (kein Scope-Creep, aber dokumentiert)

Die in PLAN.md Task 5 wörtlich vorgegebenen Test-Fixtures (`"# <id>\n---\n<yaml>\n---\n"` — ID-Kommentar VOR dem öffnenden `---`) parsen mit `pkg/bean.Parse` NICHT korrekt: die gesamte Datei fällt still auf eine Zero-Value-Frontmatter zurück (kein Error, aber Title/Parent/BlockedBy etc. bleiben leer). Das echte, von `Render()` erzeugte Format ist `"---\n# <id>\n<yaml>\n---\n"` (Kommentar INNERHALB des Frontmatter-Blocks, direkt nach dem öffnenden `---`) — siehe `pkg/bean/bean_test.go:TestRenderWithIDCommentRoundtrip`. Alle neuen T05-Fixtures nutzen das korrekte Format. Die bereits gemergten T02-Tests verwenden weiterhin das falsche Format, fallen aber nicht auf, weil sie nie auf geparste Frontmatter-Felder prüfen — daher hier nur dokumentiert, nicht repariert (außerhalb T05-Scope).

### Validation

`command go build ./cmd/beans` — grün.
`command go vet ./...` — grün, keine Findings.
`command go test ./...` — alle Pakete grün (inkl. `pkg/beancore` 3.5s, `pkg/bean`, `internal/commands`, `internal/graph`).
