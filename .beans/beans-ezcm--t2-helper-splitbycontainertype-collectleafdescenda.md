---
# beans-ezcm
title: T2 Helper — splitByContainerType + collectLeafDescendants
status: completed
type: task
priority: high
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T20:50:04Z
parent: beans-en7i
blocked_by:
    - beans-mnw0
---

Plan Task 2. Zwei Helper, TDD.

Akzeptanz:
- [ ] splitByContainerType(beans) -> (leafs, features), trennt an Type=="feature"
- [ ] collectLeafDescendants rekursiv, flacht Leafs unter Feature-Zwischenknoten ein
- [ ] visited-Guard (collectLeafDescendantsVisited) — kein Stack-Overflow bei hand-editiertem Parent-Zyklus
- [ ] TestSplitByContainerType + TestCollectLeafDescendants gruen (inkl. Zyklus-Subtest)


## Summary (2026-07-17)

Task 2 aus dem Plan umgesetzt: `splitByContainerType(beans []*bean.Bean) (leafs, features []*bean.Bean)` trennt direkte Kinder an `Type=="feature"`. `collectLeafDescendants(parentID, children, includeDone)` delegiert an `collectLeafDescendantsVisited(..., visited map[string]bool)`, die rekursiv durch verschachtelte Feature-Container läuft, Leafs an jeder Tiefe einsammelt und via `visited`-Map gegen hand-editierte Parent-Zyklen abgesichert ist. Beide Helper direkt nach `filterChildren`, vor `containsStatus` in `internal/commands/roadmap.go` eingefügt — Position exakt wie von T1 vorgemerkt.

- [x] splitByContainerType(beans) -> (leafs, features), trennt an Type=="feature"
- [x] collectLeafDescendants rekursiv, flacht Leafs unter Feature-Zwischenknoten ein
- [x] visited-Guard (collectLeafDescendantsVisited) — kein Stack-Overflow bei hand-editiertem Parent-Zyklus
- [x] TestSplitByContainerType + TestCollectLeafDescendants gruen (inkl. Zyklus-Subtest)

## Test-Output (2026-07-17)

RED:

    $ command go test ./internal/commands/ -run 'TestSplitByContainerType|TestCollectLeafDescendants' -v
    # github.com/hmans/beans/internal/commands [github.com/hmans/beans/internal/commands.test]
    internal/commands/roadmap_test.go:265:21: undefined: splitByContainerType
    internal/commands/roadmap_test.go:295:10: undefined: collectLeafDescendants
    internal/commands/roadmap_test.go:306:10: undefined: collectLeafDescendants
    internal/commands/roadmap_test.go:313:10: undefined: collectLeafDescendants
    internal/commands/roadmap_test.go:333:10: undefined: collectLeafDescendants
    FAIL	github.com/hmans/beans/internal/commands [build failed]
    FAIL

GREEN:

    $ command go test ./internal/commands/ -run 'TestSplitByContainerType|TestCollectLeafDescendants' -v
    === RUN   TestSplitByContainerType
    --- PASS: TestSplitByContainerType (0.00s)
    === RUN   TestCollectLeafDescendants
    === RUN   TestCollectLeafDescendants/flattens_through_nested_features,_excludes_done_by_default
    === RUN   TestCollectLeafDescendants/includes_done_when_requested
    === RUN   TestCollectLeafDescendants/no_children_returns_empty,_not_nil_panic
    === RUN   TestCollectLeafDescendants/hand-authored_parent_cycle_does_not_stack-overflow
    --- PASS: TestCollectLeafDescendants (0.00s)
        --- PASS: TestCollectLeafDescendants/flattens_through_nested_features,_excludes_done_by_default (0.00s)
        --- PASS: TestCollectLeafDescendants/includes_done_when_requested (0.00s)
        --- PASS: TestCollectLeafDescendants/no_children_returns_empty,_not_nil_panic (0.00s)
        --- PASS: TestCollectLeafDescendants/hand-authored_parent_cycle_does_not_stack-overflow (0.00s)
    PASS
    ok  	github.com/hmans/beans/internal/commands	0.558s

Voll-Gate:

    $ command go test ./internal/commands/ -count=1
    ok  	github.com/hmans/beans/internal/commands	0.436s

    $ command go vet ./internal/commands/...
    (exit 0, keine Ausgabe)

    $ command gofmt -l internal/commands/roadmap.go
    (exit 0, keine Ausgabe)

## Deviations/ERRATA (2026-07-17)

ERRATUM: Plan-Code war bereits gofmt-clean, `gofmt -w` lief trotzdem als Vorsichtsmaßnahme (keine Änderung).

ERRATUM: Instruktion verlangte `gofmt -l internal/commands/*.go` (Glob, leer erwartet) statt nur der geänderten Datei. Dieser Glob ist NICHT leer — 16 vorbestehende Dateien im Package sind bereits gofmt-unclean, verifiziert per `git stash` vor meiner Änderung (identische Liste inkl. roadmap_test.go, unverändert durch mich). Der gofmt-Diff für roadmap_test.go betrifft ausschließlich vorbestehende Struct-Tag-Ausrichtung in TestBuildRoadmap (Zeilen ~33-42) und TestFirstParagraph (Zeilen ~126-131) — beides weit vor meinem Append ans Dateiende, von mir nicht berührt. `roadmap.go` selbst ist gofmt-clean. Scope-Gate daher auf die zwei von mir geänderten Dateien begrenzt (deckungsgleich mit dem git-add-Scope); package-weite Formatierungs-Altlasten sind nicht Teil von T2.

## Notes for T3 (2026-07-17)

- `splitByContainerType(beans []*bean.Bean) (leafs, features []*bean.Bean)` — reine Trennfunktion, keine Sortierung, kein Status-Filter.
- `collectLeafDescendants(parentID string, children map[string][]*bean.Bean, includeDone bool) []*bean.Bean` — flache Leaf-Liste, unsortiert (T3/buildFeatureGroup muss selbst `sortByTypeThenStatus` aufrufen, siehe Plan Step 3 in Task 3).
- Beide Helper liegen zwischen `filterChildren` (Ende ~Z291 nach T1-Erweiterung) und `containsStatus` in roadmap.go — Position unverändert für nachfolgende Tasks nutzbar.
- Zyklus-Guard ist rein defensiv (CLI verhindert Feature-unter-Feature via ValidateParent), aber real getestet — buildFeatureGroup (T3) kann collectLeafDescendants ungeschützt aufrufen, der Guard sitzt intern.
