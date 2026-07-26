# Tasks — cli-agent-output-contract

Work that has become concrete. Latent convergence: this list is written *during*
divergence, but nothing here is dispatched until the open D/Q are resolved.

| ID | Prio | Task | Blocks | Status |
|---|---|---|---|---|
| T01 | high | Inventory every JSON-emitting command (envelope vs. bare) and every consumer that reads mutation JSON, in-repo and out. | D05, Q02, Q03 | 🟢 |
| T02 | high | Confirm cobra v1.10.2's `ExecuteC` error path against the module-cache source: order of `SilenceUsage` evaluation, root-vs-command flag precedence. | D07, Q01 | 🟢 |
| T10 | high | Correct the implementation note in `beans-ra75` — it prescribes `cmd.SilenceUsage = false`, which R16 proves insufficient. Leaving it would ship AC-2 broken. | — | 🟣 |
| T11 | normal | Decide the fate of the `rename --json` two-document emission and reconcile `beans-src/CLAUDE.md` with it. | D05, Q08 | 🟣 |
| T03 | high | Rewrite `beans-13ae` body to the D01/D05 reframe: state that the payload already exists (cite M04–M06), name the envelope inconsistency as the actual defect, restate the acceptance around the bare shape, and record the falsified premise so nobody re-files it. | D05 | 🟣 |
| T12 | high | Rewrite `beans-ra75` AC-2 and its success criteria to D07: `Run '<cmd> --help' for usage.` for every error class, SC-03 no longer expects a usage block on flag errors. | D07 | 🟣 |
| T13 | normal | Version `docs/` — exclusion already lifted from `.git/info/exclude`; commit the pre-existing untracked contents (`SSTD.md`, `LESSONS-LEARNED.md`, `beans-rename-command/`, `roadmap-tty-output/`) alongside this collection. | D10 | 🔵 |
| T04 | normal | Add the M03 double-emission finding to the record — as a scope widening of ra75 or as its own bean, per D06. | D06 | 🟣 |
| T05 | normal | File the EARS-validator defect as a bean in this repo (D03), flagged as a foreign-repo defect with its path. | — | 🟣 |
| T06 | normal | Re-type `beans-ra75` per D09. | D09 | 🟣 |
| T07 | low | Decide whether `beans query` and `beans create` join the contract (Q03/Q05) and record the boundary explicitly in the DESIGN. | D05, Q03, Q05 | 🟣 |
| T08 | normal | Reconstruct what actually broke the two updates in the okf-tools incident, or record it as unexplained in both beans. | Q07 | 🟣 |
| T09 | low | Establish a Go-only build/test path for CLI work so the failing frontend build does not gate it (Q06). | Q06 | 🟣 |
