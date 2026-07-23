---
# beans-36fa
title: 'roadmap: kinderlose Orphan-Epic verschwindet komplett aus Output'
status: todo
type: bug
priority: normal
created_at: 2026-07-23T21:50:30Z
updated_at: 2026-07-23T21:50:30Z
---

Fund aus dem T4-Review des Epos `beans-1ec3` (ce-specs-reviewer, 2026-07-23), beim Lauf
gegen das echte `.beans/` dieses Repos (278 beans).

## Symptom

Eine **kinderlose Orphan-Epic** (Typ `epic`, Status offen, **kein** Parent) erscheint in
**keinem** der beiden Roadmap-Ausgabepfade — weder im bestehenden Markdown (`beans roadmap`
gepiped) noch im neuen TTY-Pretty-Renderer. Verifiziert per `grep` gegen beide Ausgaben,
beide liefern nichts.

**Reales Beispiel in diesem Repo:** `beans-en7i` (Typ epic, status todo, kein Parent).

## Ursache

Liegt in `buildRoadmap` (`internal/commands/roadmap.go`), nicht im Renderer:

- `roadmap.go:163-176` — `unscheduledEpics` filtert mit `len(eg.Items) > 0 || len(eg.Features) > 0`.
  Eine Epic ohne Kinder faellt durch.
- `roadmap.go:213-243` — der `orphanItems`-Loop skippt Epics pauschal
  (`if b.Type == "milestone" || b.Type == "epic" { continue }`).

Damit gibt es fuer eine kinderlose Orphan-Epic **keinen** Aufnahmepfad in die Datenstruktur.

## Abgrenzung

- **Keine Regression und kein Defekt des TTY-Epos.** Das Verhalten ist im Markdown-Pfad
  identisch und aelter als der Pretty-Renderer. Der TTY-Renderer erbt es lediglich, weil er
  bewusst auf derselben `buildRoadmap`-Struktur aufsetzt.
- **Exakt analog zu `beans-n8zw`** (kinderloses offenes *Feature* verschwindet, gefunden im
  ti53-T6-Smoke, inzwischen `completed`). Dort war die Behebung: Feature mit 0 Nachkommen als
  flache Leaf-Zeile rendern. Derselbe Ansatz duerfte fuer Epics tragen.
- Verwandtes Muster in LESSONS-LEARNED: **LL-08** (Real-Repo-Smoke faengt, was Unit-Tests
  strukturell nicht sehen) — auch dieser Fund kam aus dem Lauf gegen echte Daten, nicht aus
  Fixtures.

## Vorgeschlagene Behebung

Kinderlose Orphan-Epic als flache Leaf-Zeile aufnehmen (Fallback), analog zur bereits
existierenden Childless-Feature-Behandlung. Betrifft **nur** `buildRoadmap` — beide Renderer
profitieren dann automatisch.

**Achtung:** Eine Aenderung an `buildRoadmap` beruehrt den Markdown-Pfad und damit dessen
Byte-Identitaets-Garantie. Vor der Umsetzung klaeren, ob die zusaetzliche Zeile im
Markdown-Output erwuenscht ist — sie ist eine gewollte Ausgabe-Aenderung, keine stille.

## Akzeptanzkriterien (Entwurf)

- [ ] Eine kinderlose Orphan-Epic erscheint in `beans roadmap` (Markdown) als Zeile.
- [ ] Sie erscheint ebenso im TTY-Pretty-Pfad.
- [ ] Test deckt "Epic ohne Kinder" ab und wird durch Mutation nachweislich rot.
- [ ] `beans-en7i` ist in beiden Ausgaben dieses Repos sichtbar.
