---
# beans-mnw0
title: T1 Datenmodell — featureGroup
status: completed
type: task
priority: high
created_at: 2026-07-17T20:37:19Z
updated_at: 2026-07-17T20:46:35Z
parent: beans-en7i
---

Plan Task 1. featureGroup zum Roadmap-Datenmodell.

Akzeptanz:
- [x] featureGroup struct (Feature *bean.Bean, Items []*bean.Bean) in internal/commands/roadmap.go
- [x] milestoneGroup + epicGroup + unscheduledGroup tragen Features []featureGroup (json omitempty)
- [x] go build ./internal/commands/... gruen (unbenutzte Felder ok in Go)



## Summary (2026-07-17)
Task 1 aus dem Plan umgesetzt: featureGroup{Feature *bean.Bean, Items []*bean.Bean} neu eingeführt, plus Features []featureGroup (json:"features,omitempty") in milestoneGroup, epicGroup und unscheduledGroup. Reine Struct-Erweiterung in internal/commands/roadmap.go, kein Verhaltenscode geändert — Wiring folgt in Task 2/3.

## Test-Output (2026-07-17)
```
$ command go build ./internal/commands/...
(exit 0, keine Ausgabe)

$ command gofmt -l internal/commands/roadmap.go
(exit 0, keine Ausgabe)

$ command go vet ./internal/commands/...
(exit 0, keine Ausgabe)
```

## Deviations/ERRATA (2026-07-17)
ERRATUM: Plan-Snippet für den Struct-Block war nicht gofmt-clean (fehlausgerichtete struct-tags, z.B. `Milestones  []milestoneGroup ` gegen das längere `*unscheduledGroup`-Tag daneben). Übernommen, aber mit `gofmt -w` neu ausgerichtet — inhaltlich identisch zum Plan, nur Spaltenausrichtung der Tags korrigiert. Zusätzlich stand vor `var roadmapCmd` eine doppelte Leerzeile (pre-existing, nicht Teil des Plan-Diffs); `gofmt -w` hat sie auf eine reduziert, da sonst `gofmt -l`-Gate rot gewesen wäre.

## Notes for T2 (2026-07-17)
- featureGroup{Feature, Items} existiert bereits — buildFeatureGroup (Task 3) kann direkt darauf schreiben.
- Alle drei Container (milestoneGroup, epicGroup, unscheduledGroup) haben jetzt Features []featureGroup (omitempty, nil-Zero-Value) — noch nirgends befüllt.
- splitByContainerType/collectLeafDescendants (Task 2) sollen laut Plan direkt nach filterChildren eingefügt werden (vor containsStatus) — Positionen im File unverändert durch T1.
- Commit f0f31fb enthält ausschließlich roadmap.go (Struct-only), keine Test-Datei-Änderung.
