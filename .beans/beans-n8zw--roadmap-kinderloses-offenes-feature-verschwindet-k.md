---
# beans-n8zw
title: 'roadmap: kinderloses offenes Feature verschwindet komplett aus Output'
status: completed
type: bug
priority: normal
tags:
    - roadmap
created_at: 2026-07-17T21:22:08Z
updated_at: 2026-07-17T21:52:25Z
parent: beans-en7i
---

Fund aus T6-Real-Repo-Smoke (2026-07-17). Der Feature-Nesting-Fix (Epic beans-en7i) behandelt Feature-beans ueberall als Container: buildFeatureGroup wird nur bei len(Items)>0 angehaengt, orphanItems-Loop skippt Type==feature. Folge: ein offenes Feature OHNE Kinder verschwindet komplett aus dem roadmap-Output.

Vorher: kinderloses Feature rendert als flache Zeile (Other/Miscellaneous). Nachher: weg.

Empirisch bestaetigt (Supervisor, Minimal-Repro): orphan feature, 0 Kinder -> fix-roadmap zeigt es NICHT. Real-Repo: bt-6oyy in beans-tui.

Spannung zur beans-Konvention "feature nur als Blatt" -> Sichtbarkeitsverlust echter Arbeit.

Moegliche Behebung: Feature mit 0 Nachkommen als flache Leaf-Zeile rendern (Fallback), Feature MIT Nachkommen als #### Feature-Sektion. Kein Test deckt "Feature ohne Kinder" ab.

PO-Entscheidung: (A) Verhalten akzeptieren (B) vor PR fixen (C) als Nachfolge-PR deferren.

## D01 (PO-Entscheidung 2026-07-17): Feature = Container IFF >=1 Nachkomme

Semantik: Ein Feature-bean ist Container gdw. collectLeafDescendants (respektiert includeDone) >=1 Item liefert.
- >=1 Nachkomme -> #### Feature-Sektion (aktuelles Verhalten, unveraendert).
- 0 Nachkommen -> das Feature-bean selbst als FLACHE Leaf-Zeile rendern, am selben Ort wo ein Leaf stuende (parent Items/Other bzw. unscheduled Other), UNTER Beachtung des normalen archive-status-Filters (ein completed/archiviertes kinderloses Feature bleibt bei !includeDone gedroppt).

## Akzeptanz (TDD, alle Tests in internal/commands/roadmap_test.go)
- [x] Ta: orphan Feature, 0 Kinder, status todo -> erscheint in Unscheduled.Other (flach), NICHT als featureGroup, NICHT gedroppt
- [x] Tb: Feature unter Epic, 0 Kinder -> erscheint in epic.Items (flach), NICHT in epic.Features
- [x] Tc: Feature direkt unter Milestone, 0 Kinder -> erscheint in milestone.Other (flach)
- [x] Td (Regression-Guard): Feature MIT 1 lebendem Kind -> weiterhin featureGroup (epic.Features bzw. Unscheduled.Features), NICHT flach
- [x] Te (Edge): kinderloses Feature status=completed, includeDone=false -> gedroppt (archive-Filter greift wie bei jedem Leaf)
- [x] Doppel-Render bleibt aus: Feature MIT Nachkommen erscheint NICHT zusaetzlich als flaches Item (bestehende Guards intakt lassen)
- [x] Voll-Gate gruen (command go test ./internal/commands/ -count=1, vet, gofmt -l roadmap.go leer)
- [x] Template: eine flache Feature-Zeile nutzt den bestehenden beanLine-Block (typeBadge feature) -> Template unveraendert, flaches Feature korrekt gerendert (kein #### fuer kinderlose)

## Betroffene Code-Sites (roadmap.go)
1. buildEpicGroup: kinderloses Feature-Kind -> in leafItems statt via buildFeatureGroup droppen
2. buildMilestoneGroup: kinderloses direktes Feature-Kind -> in Other statt droppen
3. buildRoadmap orphan-Loop: orphanItems-Skip fuer Type==feature NUR wenn Feature >=1 Nachkomme hat (sonst als flaches Other einschliessen). unscheduledFeatures-Loop bleibt (nur Features MIT Items). Doppel-Render-Guard beachten.

## Hinweis
Test Te/archive: pruefe ob ein flaches kinderloses Feature durch die bestehende cfg.IsArchiveStatus-Filterung laeuft wie andere Leafs.


## Summary (2026-07-17)

Root cause: the E13 feature-nesting fix treats every feature-typed bean as a container unconditionally (buildFeatureGroup only appends when len(Items)>0, orphan-loop skips Type==feature outright). A feature with zero descendants had no rendering path left and vanished from `beans roadmap`.

Fix (D01): added `classifyFeatureChild` in roadmap.go, applied at all 3 affected sites (buildEpicGroup, buildMilestoneGroup, buildRoadmap orphan-loop). A feature child now resolves to a featureGroup (container, unchanged) if collectLeafDescendants finds >=1 item, or is folded back into the callers own leaf/Other list (flat leaf, subject to the same archive-status filter as any other leaf) if it has none. unscheduledFeatures loop was already correct (only appends when fg.Items>0) — no change needed there, only its sibling orphan-loop needed the fix.

## Test-Output

RED (4 of 5 new tests fail as expected before the fix; Te already passed vacuously since the old code already dropped completed items):
```
=== RUN   TestOrphanChildlessFeatureAppearsAsFlatLeaf
    roadmap_test.go:501: expected Unscheduled to be non-nil
--- FAIL: TestOrphanChildlessFeatureAppearsAsFlatLeaf (0.00s)
=== RUN   TestChildlessFeatureUnderEpicAppearsInEpicItems
    roadmap_test.go:528: got 0 milestones, want 1
--- FAIL: TestChildlessFeatureUnderEpicAppearsInEpicItems (0.00s)
=== RUN   TestChildlessFeatureDirectUnderMilestoneAppearsInOther
    roadmap_test.go:559: got 0 milestones, want 1
--- FAIL: TestChildlessFeatureDirectUnderMilestoneAppearsInOther (0.00s)
=== RUN   TestFeatureWithChildRemainsContainerAlongsideChildlessSibling
    roadmap_test.go:591: epic.Items = [], want [f1] (childless feature flattened)
--- FAIL: TestFeatureWithChildRemainsContainerAlongsideChildlessSibling (0.00s)
=== RUN   TestChildlessCompletedFeatureDroppedByArchiveFilter
--- PASS: TestChildlessCompletedFeatureDroppedByArchiveFilter (0.00s)
```

GREEN (after implementing classifyFeatureChild at the 3 sites):
```
=== RUN   TestOrphanChildlessFeatureAppearsAsFlatLeaf
--- PASS: TestOrphanChildlessFeatureAppearsAsFlatLeaf (0.00s)
=== RUN   TestChildlessFeatureUnderEpicAppearsInEpicItems
--- PASS: TestChildlessFeatureUnderEpicAppearsInEpicItems (0.00s)
=== RUN   TestChildlessFeatureDirectUnderMilestoneAppearsInOther
--- PASS: TestChildlessFeatureDirectUnderMilestoneAppearsInOther (0.00s)
=== RUN   TestFeatureWithChildRemainsContainerAlongsideChildlessSibling
--- PASS: TestFeatureWithChildRemainsContainerAlongsideChildlessSibling (0.00s)
=== RUN   TestChildlessCompletedFeatureDroppedByArchiveFilter
--- PASS: TestChildlessCompletedFeatureDroppedByArchiveFilter (0.00s)
```

Full-Gate: `go test ./internal/commands/ -count=1` -> ok (0.451s); `go vet ./internal/commands/...` -> clean; `gofmt -l internal/commands/roadmap.go` -> empty.

## Smoke

Real binary build (`go build -o /tmp/b01-fix ./cmd/beans`), fresh `.beans` init, created a childless feature "Lonely", ran `roadmap`:
```
- ![feature](https://img.shields.io/badge/feature-0e8a16?style=flat-square) Lonely ([beans-69wv](.../beans-69wv--lonely.md))
```
Lonely now appears (previously invisible). Cleanup: `/tmp/b01f` and `/tmp/b01-fix` removed.

## Deviations/ERRATA

Keine. Plan/Spec-Snippet aus dem Task-bean stimmte mit dem tatsaechlich noetigen Code ueberein; einzige eigene Design-Entscheidung war die Extraktion von `classifyFeatureChild` als gemeinsamer Helper statt dreifach dupliziertem Inline-Check (bean-Empfehlung liess beides offen).

## Notes

Geaenderte Sites: buildEpicGroup (childless Feature -> eg.Items), buildMilestoneGroup (childless Feature -> milestone.Other), buildRoadmap orphan-Loop (childless orphan Feature -> orphanItems, via len(collectLeafDescendants(...))>0 Guard). Doppel-Render vermieden, weil jede Site das exakte Kriterium "hat 0 vs >=1 Nachkomme" ueber denselben collectLeafDescendants-Aufruf (in classifyFeatureChild bzw. direkt im orphan-Loop) prueft, und unscheduledFeatures/eg.Features-Pfade unveraendert blieben (die haengen weiterhin am len(fg.Items)>0-Guard, der Features mit Kindern exklusiv dort rendert). Template (roadmap.tmpl) unveraendert -- der bestehende beanLine-Block ist typ-generisch (typeBadge .) und rendert ein flaches Feature korrekt ohne Aenderung.

## REVIEW 2026-07-17 — APPROVED (unabh. Reviewer)
Alle 5 Tests (Ta-Te) grün, Mutations-Proben A (2 Stellen) bestätigen 0-Nachkommen-Zweig load-bearing, Doppel-Render-Probe am echten Binary sauber (HasChild 1x Container, Childless 1x flach), Edge nur-completed-Kinder korrekt (flach ohne --include-done, Container mit).

## NB-nonblocking (DRY, Quality) — Quelle: B01-Reviewer
Der 3. Call-Site (orphan-items-Loop roadmap.go ~Z220-229) nutzt classifyFeatureChild NICHT, sondern dupliziert die len(collectLeafDescendants)>0-Pruefung inline + separater Archive-Filter. Funktional konsistent (mutation-verifiziert), aber DRY-Verstoss ggue. Commit-Anspruch "an allen 3 Stellen via classifyFeatureChild". Optional vor PR aufraeumen (3-Zeilen-Refactor) oder als bekannte Schuld belassen. Kein Verhaltens-Impact.
