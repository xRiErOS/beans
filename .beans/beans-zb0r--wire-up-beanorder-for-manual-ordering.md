---
# beans-zb0r
title: Wire up Bean.Order for manual ordering
status: todo
type: epic
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-xej5
---

Package 2 of `docs/beans-planning-primitives/BRIEFING.md`. Requirements R-09 to R-12.

`Bean.Order` already exists at `pkg/bean/bean.go:154`, commented as a "fractional index string for manual sorting", and it survives Parse and Render. But no command writes it and `beans list --sort` does not offer it — the accepted keys are created, updated, status, priority and id. The capability sits in the format and is connected to nothing.

Decided by the PO on 2026-08-10: **order is scoped per parent** (R-12). Siblings under the same parent form one sequence; two children of different parents share no sequence.

Independent of epic one — the two can run in parallel.
