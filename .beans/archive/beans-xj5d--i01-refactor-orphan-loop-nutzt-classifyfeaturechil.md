---
# beans-xj5d
title: 'I01 Refactor: Orphan-Loop nutzt classifyFeatureChild'
status: completed
type: task
priority: low
tags:
    - roadmap
created_at: 2026-07-17T22:10:23Z
updated_at: 2026-07-17T22:12:42Z
parent: beans-en7i
---

Non-blocking DRY-Finding aus B01-Review. Der Orphan-Items-Loop in buildRoadmap (roadmap.go ~Z218-241) dupliziert die len(collectLeafDescendants)>0-Pruefung + Archive-Filter inline, statt den bestehenden Helper classifyFeatureChild zu nutzen (den buildEpicGroup/buildMilestoneGroup verwenden).

REIN INTERN, KEIN Verhaltensaenderung. Alle bestehenden Tests (insb. Ta TestOrphanChildlessFeatureAppearsAsFlatLeaf, TestUnscheduledFeatureResolvesNesting, TestUnscheduledNestedFeatureNotDoubleRendered) muessen unveraendert gruen bleiben.

Akzeptanz:
- [x] Orphan-Loop nutzt classifyFeatureChild konsistent zu den anderen 2 Sites
- [x] KEIN Verhaltensaenderung: volle Suite gruen, kein Test geaendert
- [x] Doppel-Render-Guard + Archive-Filter-Semantik erhalten
- [x] Voll-Gate: command go test ./internal/commands/ -count=1, vet, gofmt -l roadmap.go leer

## Summary

Orphan-Items-Loop in buildRoadmap (roadmap.go, feature-Fall) rief bisher inline `len(collectLeafDescendants(b.ID, children, includeDone)) > 0` auf, um Container (>=1 Leaf-Nachfahre) von kinderlosen Leaf-Features zu unterscheiden -- dieselbe Pruefung, die intern bereits in `classifyFeatureChild` (genutzt von buildEpicGroup/buildMilestoneGroup) steckt. Ersetzt durch `if fg, _ := classifyFeatureChild(b, children, includeDone); fg != nil { continue }` -- container-Fall bleibt ein `continue` (bereits ueber den unscheduledFeatures-Loop gerendert), leaf-Fall faellt wie bisher durch die bestehende generische Archive-Status-Filterung am Ende des Loops. Kommentar unveraendert gelassen (beschreibt weiterhin exakt das Verhalten). 1-Zeilen-Diff, keine Signaturaenderungen, kein Test angefasst.

## Test-Output

```
$ command go test ./internal/commands/ -count=1
ok  	github.com/hmans/beans/internal/commands	0.628s

$ command go test ./internal/commands/ -count=1 -run 'TestOrphan|TestUnscheduled' -v
=== RUN   TestUnscheduledFeatureResolvesNesting
--- PASS: TestUnscheduledFeatureResolvesNesting (0.00s)
=== RUN   TestUnscheduledEpicWithFeatureNesting
--- PASS: TestUnscheduledEpicWithFeatureNesting (0.00s)
=== RUN   TestUnscheduledNestedFeatureNotDoubleRendered
--- PASS: TestUnscheduledNestedFeatureNotDoubleRendered (0.00s)
=== RUN   TestOrphanChildlessFeatureAppearsAsFlatLeaf
--- PASS: TestOrphanChildlessFeatureAppearsAsFlatLeaf (0.00s)
PASS
ok  	github.com/hmans/beans/internal/commands	0.456s

$ command go vet ./internal/commands/
(exit 0, no output)

$ command gofmt -l internal/commands/roadmap.go
(empty)
```

`git diff --stat` gegen den Vorgaenger-Commit (8cadcdf) zeigt ausschliesslich `internal/commands/roadmap.go` (1 insertion, 1 deletion) -- keine _test.go geaendert.

Commit: 3419e8a refactor(roadmap): route orphan-feature classification through classifyFeatureChild

## Deviations/ERRATA

Keine. Umsetzung 1:1 wie im Task beschrieben.
