---
# beans-e040
title: beans rename command
status: todo
type: epic
priority: normal
created_at: 2026-07-24T10:09:33Z
updated_at: 2026-07-24T11:40:31Z
---

Slug-, Einzel-ID- und Prefix-Rebrand-Rename fuer beans (heute unmoeglich). Direct-Core-Architektur, 10 TDD-Tasks. Plan: docs/beans-rename-command/PLAN.md (ce-plan-reviewer GRUEN R3, PO-accepted).

## Kontext (geteilt — gilt für alle Kinder, DRY)

**Ziel:** `beans rename` CLI-Command — heute ist eine bean-ID ab Create permanent, es gibt keinen Rename-Mechanismus. Motivation: überlange Prefixe wie `bew_BeWiki-Python-Download-` auf `bew-` kürzen.

**Drei Modi:** (1) Slug-Rename (Single-File), (2) Einzel-ID-Rename (Kaskade über Refs), (3) Prefix-Rebrand (projekt-weit + `.beans.yml`).

**Architektur — Direct-Core (D04 REVIDIERT, PO-approved 2026-07-24):** Grounding zeigte, dass die CLI KEIN GraphQL gegen einen laufenden Server nutzt — jeder Command baut einen eigenen `beancore.Core` (`internal/commands/root.go:56`, package-level `core`/`cfg`). Alle 3 Modi laufen als Direct-Core-Operationen. Kein CLI→GraphQL-Layer. GraphQL `renameBean` ist optional/deferred (nur UI, T09).

**Pfadmodell (kritisch):** `Bean.Path` ist relativ zum `.beans/`-Dir (`c.root`), kann verschachtelt sein — NICHT repo-root-relativ. Rename erhält Subdir.

**Atomarität:** Slug = Single-File `os.Rename`. Einzel-ID + Prefix = gemeinsames atomic staging+swap-Primitiv (kein Atomic-Write-Helfer existiert — T04 baut ihn). Etag entfällt (PO-Q01 akzeptiert: CLI ist single-shot).

**Plan (autoritativ, self-contained, on-disk):** `docs/beans-rename-command/PLAN.md` — enthält je Task vollständigen TDD-Code (Steps, Tests, Commit). Jedes Kind referenziert seine Task-Nummer; der Plan trägt das Wie, das bean die Akzeptanz.

**Konventionen:** TDD (RED→GREEN), ein Commit je Task (`Refs: beans-e040`), `mise test`/`mise build`, table-driven Tests. STOPP am git-Merge-Gate (Agent merged nicht nach main).

**Review-Historie:** ce-plan-reviewer R1 ROT → R2 ROT → R3 GRÜN. Alle Findings (B01-B08, I01-I06, Q01-Q02) eingearbeitet + gegen echten Code verifiziert.

## KRITISCH — Tooling-Nicht-Ableitbarkeiten (SSTD, vor JEDEM Testlauf)

Der Plan zitiert `go test .../...` und `mise test`/`mise build`. Auf DIESER Maschine gilt stattdessen (SSTD D19/D21, LL-02/LL-11):
- **`go` ist eine Shell-Funktion** (dotfiles-Sync), die den Compiler verdeckt. `go test ./...` endet still mit Exit 0 OHNE Tests. IMMER `command go test ./...` / `command go build` / `command go vet`.
- **`mise test` ist KEIN brauchbares Gate** — es zieht `test:e2e` mit, Playwright-Browser fehlt lokal, alle e2e failen als Setup-Fail. Backend-Gate ist `command go test ./...`.
- **`awk` misst Bytes, nicht Zeichen** (nicht multibyte-aware). Für Breiten `wc -m`/`command python3`. (Hier nur relevant, falls Ausgaben geprüft werden.)
- Build-Target ist `./cmd/beans`; Version-Stamp per ldflags. `mise codegen` (T09) ist ok — nur `mise test` ist das Problem.

## Review-Preludes (Supervisor, 2026-07-24)

### Q01 (non-blocking, fürs PO-Gate) — Co-Authored-By-Trailer
T01-specs-review flaggt: Commits tragen `Co-Authored-By`-Trailer. lean-stack/CLAUDE.md fordert ihn explizit (Abschnitt 'Commits autonom'), globaler tools/CLAUDE.md-Hook E2 verbietet ihn tools-weit. Layer-Konflikt. Commit ging durch (Hook greift in beans-src offenbar nicht). ENTSCHEIDUNG PO: gilt beans-src als bewusster Fork-Override von E2 (Trailer behalten) oder Trailer droppen? Betrifft alle Task-Commits dieses Epos.

### I01 (non-blocking) — vorbestehendes gofmt in pkg/bean/id_test.go
gofmt -l meldet Alignment-Abweichung in TestContainsBlockedWord (Z.198f), NICHT durch T01 eingeführt. Bei nächstem Touch von pkg/bean via gofmt -w mitnehmen.
