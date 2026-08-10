---
# beans-zb0r
title: Wire up Bean.Order for manual ordering
status: completed
type: epic
priority: normal
tags:
    - accepted
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T12:21:48Z
parent: beans-xej5
---

Package 2 of `docs/beans-planning-primitives/BRIEFING.md`. Requirements R-09 to R-12.

`Bean.Order` already exists at `pkg/bean/bean.go:154`, commented as a "fractional index string for manual sorting", and it survives Parse and Render. But no command writes it and `beans list --sort` does not offer it — the accepted keys are created, updated, status, priority and id. The capability sits in the format and is connected to nothing.

Decided by the PO on 2026-08-10: **order is scoped per parent** (R-12). Siblings under the same parent form one sequence; two children of different parents share no sequence.

Independent of epic one — the two can run in parallel.

## Opus review 2026-08-10 (post-completion, independent verification)

Second Opus review pass verified all four leaves against their acceptance criteria and the real CLI binary (not just `go test`). `beans-uo43` and `beans-9ftf` passed cleanly. `beans-y2a2` and `beans-nda7` had genuine, reproduced bugs — Improvement/Question findings (I01-I03, Q01) were discarded per PO instruction ("Opus ist immer sehr kreativ, alles Mögliche zu finden"); only confirmed bugs were fixed:

- **B01** (critical, `beans-y2a2`): `order.go`'s `--last`/`--after`/`--before` picked neighbours straight from the sorted sibling list without checking whether that sibling actually carried an `Order`. Since `SortByOrder` puts unordered siblings *last*, and every real store starts with zero ordered beans, the very first `beans order --last` call already hit the broken path — every subsequent call got the same constant key `"V"`. Fixed: `--last` now walks back to the last sibling that actually has an `Order`; `--after`/`--before` now reject a reference bean that has none, with a clear error, instead of silently computing a meaningless key.
- **B02** (high, `beans-y2a2`): `order.go` called `core.Update(b, nil)`, bypassing ETag concurrency control entirely — same class as the B01 fix already applied to `beans-2ark`'s `create.go`/`update.go`. Under `require_if_match: true`, `beans order` failed outright on every call.
- **B05** (discovered while fixing B02, not in the original review): naively mirroring `create.go`'s `etag := b.ETag()` pattern broke `beans order` for *every* ordinary bean, worse than B02 itself. Every `beans order` run is a separate process; `b` is loaded via `Core.Load`, which defaults empty `Priority`/`Type` in memory (fields an on-disk file written without `-p`/`-t` never has). Re-rendering that defaulted bean produces bytes `core.Update`'s on-disk etag check never sees. Fixed by hashing the actual on-disk bytes (`bean.ETagOf`, extracted from `Bean.ETag`) instead of re-deriving the etag from the in-memory bean.
- **B03/B04** (high/medium, `beans-nda7`, surfaced only once `beans-9ftf` opened `--order` to arbitrary values): `midpoint()`'s zero-padding makes a key ending in the zero digit indistinguishable from a longer key sharing its prefix, so `OrderBetween("1","10")` returned `"10V"` — greater than `b`, violating the core between-a-and-b contract. `decrementKey("0")` returned `"0"` unchanged (duplicate keys). Both close at the same entry point: `IsValidOrderKey` now rejects any key ending in the zero digit (no internally generated key ever does), so neither malformed state is reachable via `--order` anymore.

All four fixed via TDD (RED/GREEN + reverse-mutation proof per repo CLAUDE.md), verified against the real built binary across separate process invocations (not just in-process `go test` fixtures), `go build`/`go vet`/`go test ./...` green. Commits: `dbafaa0` (B03/B04), `7acf6b7` (B01/B02/B05).

I01-I03/Q01 (test blind-spot fixture gap, `OrderBetween` precondition contract, default-sort/Order divergence, missing `--if-match` on `order`) intentionally left open per PO — improvement/question class, not bugs.

## Review 2026-08-10 (PO-Review)

US-07 · `beans order --after/--before/--first/--last` places a bean relative to its siblings, one file written, rejects an unordered reference bean with a clear error · a
US-08 · `beans list --sort order` shows beans in the manually assigned sequence, unordered siblings last · a
US-09 · `beans create --order <value>` sets an explicit order at creation time, invalid/trailing-zero values rejected · a
