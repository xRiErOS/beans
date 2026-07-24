---
# beans-z1we
title: 'T06 rename: prefix-rebrand + .beans.yml'
status: todo
type: task
priority: high
created_at: 2026-07-24T10:10:14Z
updated_at: 2026-07-24T12:08:31Z
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
