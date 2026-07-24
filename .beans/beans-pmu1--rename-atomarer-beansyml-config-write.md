---
# beans-pmu1
title: 'rename: atomarer .beans.yml Config-Write'
status: todo
type: task
priority: normal
created_at: 2026-07-24T13:15:09Z
updated_at: 2026-07-24T13:15:09Z
parent: beans-a29l
---

Aus beans-e040 D02 (T06-specs-review). Prefix-Rebrand schreibt .beans.yml via config.Save() (pkg/config/config.go:343-363) NACH dem atomaren stageAndSwap — aber Save() ist selbst nicht-atomar (os.WriteFile, kein temp+rename). Bricht der Save ab: Beans schon auf newPrefix umbenannt, .beans.yml hält noch oldPrefix. Folge: config.Beans.Prefix treibt projekt-weite Short-ID-Normalisierung (core.go:321,344,697,834) UND Prefix-Vergabe neuer Beans (core.go:490-499) → nach Reload brechen Short-ID-Lookups + neue Beans bekommen alten Prefix = Mixed-Prefix-Zustand, den B04 beim nächsten Rebrand verweigert.

## Ziel
config.Save() atomar machen (temp-Datei + os.Rename) ODER den Config-Write in den stageAndSwap einbeziehen. Zusätzlich Post-Reload-Prefix-Konsistenz-Validierung, die einen Mixed-Prefix-Zustand erkennt. Test für den Partial-Failure-Pfad ergänzen (aktuell 0 Coverage).

## Quelle
beans-e040 Epic-Body D02 + T06-specs-review (2026-07-24). Verwandt mit D01 (beide Partial-Failure-Residualrisiken der Rename-Op).
