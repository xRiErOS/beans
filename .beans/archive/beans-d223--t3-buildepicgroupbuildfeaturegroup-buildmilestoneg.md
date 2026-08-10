---
# beans-d223
title: T3 buildEpicGroup/buildFeatureGroup + buildMilestoneGroup rewire
status: completed
type: task
priority: high
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T21:02:21Z
parent: beans-en7i
blocked_by:
    - beans-ezcm
---

Plan Task 3. Feature-Aufloesung im Milestone/Epic-Pfad.

Akzeptanz:
- [ ] buildFeatureGroup + buildEpicGroup ergaenzt
- [ ] buildMilestoneGroup loest Epic->Feature->Leaf und Milestone->Feature->Leaf auf
- [ ] Milestone-Inclusion-Check ergaenzt || len(group.Features) > 0
- [ ] TestBuildMilestoneGroupResolvesFeatureNesting gruen


## Summary (2026-07-17)

Task 3 aus dem Plan umgesetzt: buildMilestoneGroup komplett ersetzt (trennt Epics/Rest, ruft buildEpicGroup je Epic sowie splitByContainerType+buildFeatureGroup fuer feature-typisierte Milestone-Direktkinder auf). buildEpicGroup neu ergaenzt (leaf/feature-Split via splitByContainerType, buildFeatureGroup je Feature-Kind, sortByTypeThenStatus auf leafItems). buildFeatureGroup neu ergaenzt (collectLeafDescendants + Pflicht-sortByTypeThenStatus, siehe Notes-for-T3 aus beans-ezcm). Milestone-Inclusion-Check in buildRoadmap um || len(group.Features) > 0 erweitert.

[x] buildFeatureGroup + buildEpicGroup ergaenzt
[x] buildMilestoneGroup loest Epic->Feature->Leaf und Milestone->Feature->Leaf auf
[x] Milestone-Inclusion-Check ergaenzt || len(group.Features) > 0
[x] TestBuildMilestoneGroupResolvesFeatureNesting gruen

## Test-Output (2026-07-17)

RED:

  $ command go test ./internal/commands/ -run 'TestBuildRoadmap|TestBuildMilestoneGroupResolvesFeatureNesting' -v
  ...
  === RUN   TestBuildMilestoneGroupResolvesFeatureNesting
      roadmap_test.go:375: epic.Items = [0x23e466e84a00 0x23e466e84800], want [b1]
      roadmap_test.go:378: got 0 feature groups, want 1
  --- FAIL: TestBuildMilestoneGroupResolvesFeatureNesting (0.00s)
  FAIL
  FAIL	github.com/hmans/beans/internal/commands	0.539s
  FAIL

  (Die neue TestBuildRoadmap-Tabellenzeile "leaf nested under feature under epic under milestone is not lost" passte bereits vor der Aenderung, wie im Plan dokumentiert - kein RED-Signal, nur Regressionsnetz. Der echte RED-Beweis ist der Standalone-Test oben.)

GREEN:

  $ command go test ./internal/commands/ -run 'TestBuildRoadmap|TestBuildMilestoneGroupResolvesFeatureNesting' -v
  === RUN   TestBuildRoadmap
  ... (alle Subtests PASS, inkl. leaf_nested_under_feature_under_epic_under_milestone_is_not_lost)
  --- PASS: TestBuildRoadmap (0.00s)
  === RUN   TestBuildMilestoneGroupResolvesFeatureNesting
  --- PASS: TestBuildMilestoneGroupResolvesFeatureNesting (0.00s)
  PASS
  ok  	github.com/hmans/beans/internal/commands	0.533s

Voll-Gate:

  $ command go test ./internal/commands/ -count=1
  ok  	github.com/hmans/beans/internal/commands	0.568s

  $ command go vet ./internal/commands/...
  (exit 0, keine Ausgabe)

  $ command gofmt -l internal/commands/roadmap.go
  (exit 0, leer - nur roadmap.go geprueft, wie instruiert)

## Deviations/ERRATA (2026-07-17)

ERRATUM: gofmt -l internal/commands/roadmap_test.go ist NICHT leer (2 Fundstellen: struct-Tag-Ausrichtung in TestBuildRoadmap-Tabelle Z35-42 und TestFirstParagraph Z138-140). Verifiziert per gofmt -d: beide Diffs betreffen ausschliesslich vorbestehende, von T2 bereits dokumentierte Struct-Feld-Ausrichtung - nicht meine neue Test-Tabellenzeile und nicht mein neuer TestBuildMilestoneGroupResolvesFeatureNesting-Funktionskoerper. Instruktion verlangte ohnehin nur gofmt -l auf roadmap.go, nicht auf die Testdatei - Scope-Gate eingehalten.

Sonst keine Abweichungen vom Plan-Snippet (Step 3 Code 1:1 uebernommen).

## Notes for T4 (2026-07-17)

- buildEpicGroup(epic *bean.Bean, children map[string][]*bean.Bean, includeDone bool) epicGroup ist jetzt die kanonische Epic-Aufloesung (leaf/feature-Split + Features-Feld befuellt, Items sortiert). T4 (Unscheduled-Pfad) soll laut Plan diese Funktion fuer unscheduled Epics wiederverwenden statt der alten Inline-filterChildren-Logik in buildRoadmap (aktuell noch bei "Find unscheduled epics" unveraendert - ruft filterChildren+epicGroup{} direkt, nicht buildEpicGroup).
- buildFeatureGroup(feature *bean.Bean, children map[string][]*bean.Bean, includeDone bool) featureGroup ist die kanonische Feature-Aufloesung fuer orphan-Feature-Resolution in T4.
- Alle drei neuen/geaenderten Funktionen liegen unmittelbar nach buildRoadmap, vor filterChildren in roadmap.go - Reihenfolge: buildMilestoneGroup, buildEpicGroup, buildFeatureGroup, filterChildren.
- unscheduledGroup.Features (bereits im Struct vorhanden seit T1) ist von buildRoadmap noch NICHT befuellt - das ist explizit T4-Scope (orphan Feature -> Leaf Aufloesung + unscheduled Epics via buildEpicGroup).

## REVIEW 2026-07-17 — CHANGES_REQUIRED (Quelle: unabh. Reviewer T3, Mutations-Probe A)

Code korrekt (Diff = Plan, keine Fehlfunktion). ABER: load-bearing Coverage-Lücke bestätigt — Milestone-Inclusion-Zeile `|| len(group.Features) > 0` in buildRoadmap ist ungetestet. Mutations-Probe: Zeile auf alten Stand zurück → KEIN Test failt. Ursache: jeder bestehende Feature-Test hat immer auch ein Epic-Geschwister unter der Milestone → group.Epics nie leer wenn Features gefüllt.

Restaufgabe (1 Commit):
1. Neue TestBuildRoadmap-Tabellenzeile: Milestone mit NUR Feature-Direktkind (kein Epic), Leaf muss gerendert werden — wantMilestones:1. Muss failen wenn Zeile 142 zurückgedreht wird.
2. Optional (nicht blockierend): zweites Item unter einem Feature für Sort-Beobachtbarkeit (Mutations-Probe B).


## REVIEW-2026-07-17 (unabh. Reviewer, Mutations-Probe) — CHANGES_REQUIRED, behoben

Befund: Milestone-Inclusion-Zeile in buildRoadmap (`|| len(group.Features) > 0`) war ungetestet — jeder bestehende Feature-Test hatte immer ein Epic-Geschwister unter der Milestone, daher blieb ein Zuruecksetzen der Klausel unentdeckt.

Massnahme:
[x] Neue Tabellenzeile in TestBuildRoadmap: Milestone mit NUR Feature-Direktkind (kein Epic) — "milestone with direct feature child and no epic is not dropped"
[x] RED-Beweis per temporaerer Mutation (Klausel zurueckgedreht) erbracht und zitiert
[x] Zeile wiederhergestellt, GREEN erneut erbracht
[x] Voll-Gate (Package-Test, vet, gofmt -l roadmap.go) erneut gruen

RED (mutiert, `|| len(group.Features) > 0` entfernt):

  $ command go test ./internal/commands/ -run TestBuildRoadmap -v
  === RUN   TestBuildRoadmap/milestone_with_direct_feature_child_and_no_epic_is_not_dropped
      roadmap_test.go:127: got 0 milestones, want 1
  --- FAIL: TestBuildRoadmap/milestone_with_direct_feature_child_and_no_epic_is_not_dropped (0.00s)
  FAIL
  FAIL	github.com/hmans/beans/internal/commands	0.520s

GREEN (Klausel wiederhergestellt):

  $ command go test ./internal/commands/ -run TestBuildRoadmap -v
  --- PASS: TestBuildRoadmap (0.00s)
      --- PASS: TestBuildRoadmap/milestone_with_direct_feature_child_and_no_epic_is_not_dropped (0.00s)
  PASS
  ok  	github.com/hmans/beans/internal/commands	0.718s

Voll-Gate:

  $ command go test ./internal/commands/ -count=1
  ok  	github.com/hmans/beans/internal/commands	0.461s
  $ command go vet ./internal/commands/...
  (exit 0)
  $ command gofmt -l internal/commands/roadmap.go
  (exit 0, leer)

Scope: NUR roadmap_test.go geaendert (git status --short: ` M internal/commands/roadmap_test.go` vor Commit). Punkt 3 aus der Review-Aufgabe (zweites Item unter f1 fuer Sortierungs-Beobachtbarkeit) bewusst ausgelassen — als "optional, nicht blockierend" markiert, Scope-Minimalismus bevorzugt.

Commit: ca4ab06 test(roadmap): pin milestone-direct-feature inclusion line
