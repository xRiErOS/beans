# SPEC (Skelett) — beans-rename-command

> Think-Phase-Deliverable (TPIC). Konzeptionelles Gerüst, **noch keine EARS** — User-Story + EARS
> entstehen erst bei der Operationalisierung im bean. Grounding: siehe REFERENCES.md, Entscheidungen: DECISIONS.md.

## 1. Zweck

Ein `beans rename` CLI-Command, das bean-IDs und -Slugs nachträglich ändern kann — heute unmöglich
(ID ist ab Create permanent, kein Rename-Mechanismus). Motivation: überlange Prefixe wie
`bew_BeWiki-Python-Download-` auf `bew-` kürzen → IDs `bew-ljs5` statt `bew_BeWiki-Python-Download-ljs5`.

## 2. Scope

Drei Modi:
1. **Slug-Rename** — Dateiname-Teil nach `--` setzen/entfernen/regenerieren. Kein Cross-Ref-Risiko (Slug wird nirgends referenziert).
2. **Einzel-ID-Rename** — eine bean bekommt neue ID; Refs anderer beans kaskadieren.
3. **Prefix-Rebrand** — projekt-weit: uniformer Prefix-Tausch über alle beans (Suffix bleibt), inkl. `.beans.yml`.

## 3. Non-Goals

- Externe ID-Referenzen (Commit-Messages, docs, SSTD) werden NICHT rewritten (D11).
- Kein auto-`git mv`/commit — beans committet nie selbst (D10).
- Worktree-`.beans/` werden NICHT mitmigriert; Rebrand verweigert stattdessen bei aktiven Worktrees (D05).
- Kein Live-Rename der Massen-Prefix-Migration im UI (offline-Batch, D04).

## 4. Architektur (Hybrid, D04)

| Modus | Ausführung | Ort (grob) |
|-------|-----------|------------|
| Slug, Einzel-ID | GraphQL-Mutation `renameBean` über laufenden Server | `internal/graph/schema.graphqls`, `pkg/beangraph/mutations.go`, `internal/commands/rename.go` |
| Prefix-Rebrand | offline Disk-Batch (Server + Worktrees müssen aus/leer) | `internal/commands/rename.go` + Rebrand-Kern in `pkg/beancore` |

Gemeinsamer Kaskade-Kern (reine Transformation, testbar isoliert):
- Kandidat: neue Funktion in `pkg/bean` (ID/Slug/Filename-Transform) + Orchestrierung in `pkg/beancore` (Multi-File-Rewrite).

## 5. Kaskade-Algorithmus

Für ID-Änderung `old → new`:
1. `BuildFilename(new, slug)` → Datei `old--slug.md` → `new--slug.md`.
2. `# old`-Kommentarzeile (`Render`) → `# new`.
3. Alle beans scannen, in `parent`/`blocking`/`blocked_by` jedes Vorkommen `old` → `new`.

Prefix-Rebrand:
- new-ID-Map `{old_id → new_prefix + suffix}` über alle beans (Suffix = ID ohne alten Prefix).
- Schritte 1–3 über gesamte Menge; anschließend `.beans.yml` `prefix:` schreiben (D12).

## 6. Atomarität (D06)

- **Offline-Rebrand**: alle beans in Memory laden → vollständige Map + Ref-Rewrite + Filenames berechnen →
  in Temp-Staging-Dir schreiben → atomarer Swap (alte `.beans/` als Backup). Fehler vor Swap → Abbruch, Original unberührt.
- **Live-Mutation** (Slug/Einzel-ID): bestehender transaktionaler Schreibpfad + etag-Validierung.

## 7. Guards & Sicherheit (D05, D07, D09)

- Rebrand refuse: `beans serve` läuft (Lock/Port-Check) ODER aktive Worktrees in `~/.beans/worktrees/<proj>/`.
- Einzel-ID: Kollisions-Check gegen bestehende IDs.
- `--dry-run` auf allen Modi (zeigt geplante ID-/Datei-/Ref-Änderungen, ändert nichts).
- Rebrand ohne `--yes` → Summary + Confirm-Prompt.

## 8. CLI-Oberfläche (D08, D09)

```
beans rename <id> --slug "neuer-slug"   # Slug setzen
beans rename <id> --no-slug             # Slug entfernen → id.md
beans rename <id> --reslug              # aus Title regenerieren (Slugify)
beans rename <id> <neue-id>             # volle neue ID
beans rename <id> --suffix k7x2         # nur Suffix, Prefix bleibt
beans rename --prefix "bew-"            # projekt-weiter Prefix-Rebrand
# quer: --dry-run, --yes, --json, --beans-path
```

## 9. Tests (Pflicht)

- Table-driven Unit: ID/Slug/Prefix-Transform (`pkg/bean`).
- Kaskade-Integrität: nach Rename kein toter Ref, keine Kollision, `.beans.yml` konsistent.
- `--dry-run`: keine Mutation, Plan == tatsächliche Änderung.
- Rebrand-Atomarität: simulierter Fehler vor Swap → Original unberührt.
- Guard-Tests: Server-läuft / aktive-Worktrees → refuse.
- E2E: nur falls UI berührt — aktuell nein (reines CLI/Backend).

## 10. Offene Feinheiten für Plan-Phase

- Exakter Ort des Kaskade-Kerns (`pkg/bean` vs `pkg/beancore`) — beim Plan an bestehende Struktur anlegen.
- Server-Lauf-Detektion: Lock-File vs Port-Probe — bestehenden Mechanismus prüfen.
- GraphQL-Schema-Form der `renameBean`-Mutation (Input-Typ, Rückgabe) + `mise codegen`.
- Legacy-Filename-Formate (`id.slug`, `id-slug`) beim Rename normalisieren auf `id--slug`?
