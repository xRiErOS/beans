---
# beans-4lui
title: T4 Unscheduled-Pfad — Orphan-Features + Doppel-Render-Guard
status: completed
type: task
priority: high
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T21:07:43Z
parent: beans-en7i
blocked_by:
    - beans-d223
---

Plan Task 4. Unscheduled-Epics + Orphan-Features.

Akzeptanz:
- [ ] unscheduled-Epics-Loop nutzt buildEpicGroup
- [ ] Orphan-Feature-Loop; Guard skippt Parent Type epic ODER feature (kein Doppel-Render)
- [ ] orphanItems-Loop skippt Type feature
- [ ] unscheduledGroup verdrahtet Features
- [ ] TestUnscheduledFeatureResolvesNesting + ...EpicWithFeatureNesting + ...NestedFeatureNotDoubleRendered gruen

[x] unscheduled-Epics-Loop nutzt buildEpicGroup
[x] Orphan-Feature-Loop; Guard skippt Parent Type epic ODER feature (kein Doppel-Render)
[x] orphanItems-Loop skippt Type feature
[x] unscheduledGroup verdrahtet Features
[x] TestUnscheduledFeatureResolvesNesting + ...EpicWithFeatureNesting + ...NestedFeatureNotDoubleRendered gruen

## Summary (2026-07-17)

Task 4 aus dem Plan umgesetzt: unscheduled-Epics-Loop in buildRoadmap auf buildEpicGroup umgestellt (statt Inline-filterChildren). Neuer unscheduled-Features-Loop ergaenzt: findet feature-typisierte Beans ohne Milestone, mit Guard der Parent-Type epic ODER feature ueberspringt (verhindert Doppel-Render von genesteten Orphan-Features). orphanItems-Loop-Guard um Type feature erweitert. unscheduledGroup-Konstruktion um Features: unscheduledFeatures + Inclusion-Bedingung || len(unscheduledFeatures) > 0 erweitert.

## Test-Output (2026-07-17)

RED:

$ command go test ./internal/commands/ -run 'TestUnscheduled' -v
=== RUN   TestUnscheduledFeatureResolvesNesting
    roadmap_test.go:413: got 0 unscheduled features, want 1
--- FAIL: TestUnscheduledFeatureResolvesNesting (0.00s)
=== RUN   TestUnscheduledEpicWithFeatureNesting
    roadmap_test.go:447: eg.Features = [], want feature f1
--- FAIL: TestUnscheduledEpicWithFeatureNesting (0.00s)
=== RUN   TestUnscheduledNestedFeatureNotDoubleRendered
    roadmap_test.go:472: got 0 unscheduled features, want 1 (f1 only, f2 must not double-render)
--- FAIL: TestUnscheduledNestedFeatureNotDoubleRendered (0.00s)
FAIL
FAIL	github.com/hmans/beans/internal/commands	0.599s

GREEN:

$ command go test ./internal/commands/ -run 'TestUnscheduled' -v
=== RUN   TestUnscheduledFeatureResolvesNesting
--- PASS: TestUnscheduledFeatureResolvesNesting (0.00s)
=== RUN   TestUnscheduledEpicWithFeatureNesting
--- PASS: TestUnscheduledEpicWithFeatureNesting (0.00s)
=== RUN   TestUnscheduledNestedFeatureNotDoubleRendered
--- PASS: TestUnscheduledNestedFeatureNotDoubleRendered (0.00s)
PASS
ok  	github.com/hmans/beans/internal/commands	0.569s

Voll-Gate:

$ command go test ./internal/commands/ -count=1
ok  	github.com/hmans/beans/internal/commands	0.456s
$ command go vet ./internal/commands/...
(exit 0, keine Ausgabe)
$ command gofmt -l internal/commands/roadmap.go
(exit 0, leer)

## Deviations/ERRATA (2026-07-17)

keine. Plan-Snippet (Steps 3-5) 1:1 uebernommen, Code-Anker per grep verifiziert (Zeilennummern im Plan waren veraltet wie erwartet).

## Notes for T5 (2026-07-17)

Datenmodell-Stand nach T4: unscheduledGroup.Epics, .Features und .Other sind jetzt ALLE befuellt (Features vorher nur strukturell vorhanden, jetzt aktiv gesetzt). epicGroup.Features (fuer unscheduled Epics) ist ebenfalls befuellt via buildEpicGroup-Wiederverwendung -- identische Struktur wie im Milestone/Epic-Pfad aus T3. T5 (Template-Task, Feature-Sektionen rendern) kann also fuer den unscheduled-Pfad exakt dieselbe Rendering-Logik wie fuer den Milestone-Pfad verwenden -- kein Sonderfall noetig. Reihenfolge der Funktionen in roadmap.go unveraendert (buildMilestoneGroup, buildEpicGroup, buildFeatureGroup, filterChildren, dann buildRoadmap-interne unscheduled-Logik).

Commit: c0bc49d fix(roadmap): resolve Feature nesting for unscheduled epics and orphan features
