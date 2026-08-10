---
# beans-9ftf
title: Accept an explicit index in beans create --order
status: completed
type: task
priority: low
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T11:07:51Z
parent: beans-zb0r
blocked_by:
    - beans-nda7
---

For importing an order that already exists elsewhere, so that a migration does not have to run a placement command per bean.

### Requirement 1: An order value can be set at creation

**Objective:** As someone importing an existing backlog, I want to set the order value while creating a bean, so that the imported sequence arrives in one pass.

#### Acceptance Criteria

1. WHEN beans create --order <value> runs THE CLI SHALL write that value to the order field of the new bean
2. WHEN beans create runs without --order THE CLI SHALL leave the order field empty
3. IF the given value is not a valid fractional index THEN THE CLI SHALL exit non-zero with an error

#### Success Criteria

- SC-01: Five beans created with ascending explicit order values come back from `beans list --sort order` in exactly that sequence.

_Requirements: R-11_

## Recommended Skills

- `tdd`



## Implementation Report

Implemented per AC1-3 / SC-01 via TDD (RED verified before implementation).

- `pkg/bean/fractional.go`: added exported `IsValidOrderKey(s string) bool` — non-empty and every rune present in `base62Digits`.
- `internal/commands/create.go`: added `--order` string flag. Fail-fast validation (`cmd.Flags().Changed("order") && !bean.IsValidOrderKey(createOrder)`) runs before `CreateBean`, alongside `validateExtraKeys`. On success, `b.Order = createOrder` is set and persisted via `core.Update` — combined into the same second write already used for `--set`/`--unset` (one write when order and extra keys are both given, none when neither is).
- Tests: `pkg/bean/fractional_test.go` (`TestIsValidOrderKey`, table-driven), `internal/commands/create_test.go` (`TestCreateCmdOrderWritesValue` AC1, `TestCreateCmdWithoutOrderLeavesEmpty` AC2, `TestCreateCmdOrderInvalidValueFails` AC3, `TestCreateCmdOrderAscendingSortOrder` SC-01 via `bean.SortByOrder`).

Deviation: tests drive `--order` through a throwaway `*cobra.Command` (`createCmdWithOrderFlag()`) carrying only that flag, rather than registering it on the shared `createCmd` singleton in test setup. Reason: `createCmd` is a package-level var, and `RegisterCreateCmd` is already called once elsewhere in the test binary (`path_test.go` via `RegisterCoreCommands`); registering the flag a second time panics with "flag redefined". `createCmd.RunE` reads flag state off whatever `*cobra.Command` it's invoked with, so this is behaviorally equivalent to the real CLI path without double-registering.

## Validation

```
go build ./...  -> exit 0
go test ./...   -> ok (all packages)
```

Manual CLI smoke test (SC-01): five beans created with `--order 1/3/5/7/9`, `beans list --sort order` returned them in exactly that ascending sequence. `--order 'bad key!'` exited non-zero with "invalid order value"; no bean created. `create` without `--order` left the field empty.

Commits: <to be filled after commit>
