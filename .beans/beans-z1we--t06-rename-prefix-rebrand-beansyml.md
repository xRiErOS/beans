---
# beans-z1we
title: 'T06 rename: prefix-rebrand + .beans.yml'
status: completed
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:12:44Z
parent: beans-e040
blocked_by:
    - beans-9p12
    - beans-n1ow
---

PlanRebrand + config write. Plan Task 6.

## Objective
Als Nutzer will ich projekt-weit den ID-Prefix tauschen (`beans rename --prefix "bew-"`), sodass alle beans (Suffix erhalten), alle Refs und `.beans.yml` konsistent migriert werden — die Kern-Motivation (überlange Prefixe kürzen).

## EARS
- WHEN `PlanRebrand(newPrefix)` aufgerufen wird, THE SYSTEM SHALL jede bean-ID via `RebrandID` auf `newPrefix` mappen (Suffix erhalten) und `Mode:"prefix"`, `NewPrefix`, `ConfigWrite:true` setzen.
- IF eine bean-ID nicht mit dem aktuellen Prefix beginnt, THE SYSTEM SHALL den Rebrand verweigern (kein Doppel-Prefix, B04-Guard).
- WHEN ein prefix-Plan angewendet wird, THE SYSTEM SHALL die Kaskade atomar ausführen UND danach den neuen `prefix:` in `.beans.yml` via `Config.Save(ConfigDir())` schreiben.

## Success Criteria
- SC-001: `go test ./pkg/beancore/ -run TestRebrand` GRÜN — alle IDs remapped, Refs intakt nach Reload, `.beans.yml` prefix geschrieben.
- SC-002: `TestRebrand_mixedPrefixRefused` GRÜN — Mixed-Prefix-Repo wird verweigert.

## Betroffene Pfade
- `pkg/beancore/rename.go` (PlanRebrand + ApplyRename prefix-Zweig + writeRebrandConfig; Import `strings`), `pkg/beancore/rename_test.go`. Details: PLAN.md Task 6.

## Prelude aus T05-Review (Supervisor, 2026-07-24)

- I01 (low): `countRefHits` (pkg/beancore/rename.go) — der `Blocking`-Zweig ist 0% test-covered (kein TestRenameID_* nutzt eine `blocking:`-Fixture). Betrifft nur die angezeigte Ref-Anzahl (plan.RefUpdates, Dry-Run), NICHT die Cascade-Korrektheit (rewriteRefs Blocking-Zweig ist separat unit-getestet). Prefix-Rebrand nutzt countRefHits projekt-weit — bei den T06-Tests eine Fixture mit `blocking:`-Feld einbauen und die Ref-Zählung asserten, dann ist die Lücke geschlossen.
- I02 (low): T02-Legacy-Fixtures in rename_test.go nutzen weiterhin das kaputte `# id`-vor-`---`-Format (parsen zu Zero-Value, fallen nicht auf da nichts Geparstes geprüft wird). Falls du rename_test.go anfasst, bei Gelegenheit mitreparieren (korrekte Form: `---
# id
<yaml>
---`).

## Summary of Changes

- [x] `PlanRebrand(newPrefix)` (`pkg/beancore/rename.go`): mappt jede bean-ID via `bean.RebrandID(id, oldPrefix, newPrefix)` (Suffix erhalten), `Mode:"prefix"`, `NewPrefix`, `ConfigWrite:true`. Guards: gleicher Prefix, leerer Match, Kollision.
- [x] B04-Guard: jede ID, die nicht mit dem aktuellen `c.config.Beans.Prefix` beginnt, verweigert den gesamten Rebrand (kein Doppel-Prefix) — `TestRebrand_mixedPrefixRefused`.
- [x] `ApplyRename` Mode `"prefix"`: führt zuerst die atomare Kaskade (`applyRenameCascade`/`stageAndSwap`, wiederverwendet aus T05) aus, danach `writeRebrandConfig` → `c.config.Beans.Prefix = newPrefix; c.config.Save(c.config.ConfigDir())`. Config-Write bewusst NACH dem atomaren Swap (nicht Teil desselben Swaps) — bean-Dateien tragen die ID im Dateinamen/Frontmatter, ein unterbrochener Config-Write hinterlässt daher keinen inkonsistenten bean-Zustand, nur einen veralteten `.beans.yml`-Prefix (Deviation, s.u.).
- [x] SC-001 GRÜN: `go test ./pkg/beancore/ -run TestRebrand -v` → 4/4 PASS (`TestRebrand_countsBlockingRefs`, `TestRebrand_remapsAllAndWritesConfig`, `TestRebrand_mixedPrefixRefused`, `TestRebrand_samePrefixRejected`).
- [x] SC-002 GRÜN: `TestRebrand_mixedPrefixRefused` Teil der obigen Suite.
- [x] I01 adressiert: neue `TestRebrand_countsBlockingRefs`-Fixture mit `blocking:`-Feld schließt die 0%-Lücke im `countRefHits`-Blocking-Zweig (vorher nur `TestRewriteRefs` direkt getestet, `countRefHits` selbst nie über einen Blocking-Ref).
- [x] I02 adressiert: die beiden T02-Legacy-Fixtures (`TestApplyRenameSlug_setsSlug`, `TestApplyRenameSlug_idAndRefsUnchanged`) auf die korrekte Form `---\n# id\n<yaml>\n---` umgestellt und je eine Assertion auf ein geparstes Feld (`Title`) ergänzt, damit ein zukünftiger Format-Regress nicht mehr tautologisch grün bleibt.
- [x] Vollständiges Gate: `command go vet ./...` clean, `command go build ./cmd/beans` OK, `command go test ./...` alle Pakete PASS (keine Regression in bestehenden T01-T05-Tests).

### Deviation (begründet)
- Testnamen an SC-001 angepasst: die im PLAN.md/T06-Abschnitt vorgeschlagenen Namen (`TestRebrand_remapsAllAndWritesConfig`, `TestRebrand_mixedPrefixRefused`) wurden übernommen; zwei zusätzliche von mir zunächst als `TestPlanRebrand_*` benannte Tests (Blocking-Coverage, Same-Prefix-Guard) liefen dadurch am literalen `-run TestRebrand`-Filter aus SC-001 vorbei und wurden zu `TestRebrand_*` umbenannt (Commit `test(rename): align rebrand test names with SC-001`), damit SC-001 als geschriebene Akzeptanz auch tatsächlich alle vier neuen Tests erfasst.
- Guards für laufenden Server / aktive Worktrees (`checkServerNotRunning`/`checkNoActiveWorktrees`) sind laut Aufgabenstellung explizit T07-Scope und wurden NICHT in `PlanRebrand` verdrahtet — bewusst außerhalb des T06-Scopes belassen.
