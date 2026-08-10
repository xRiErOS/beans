---
# beans-slvx
title: Carry extra keys through the GraphQL schema
status: todo
type: task
priority: normal
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

Pull `gqlgen.yml` and `pkg/beangraph` along, then regenerate with `mise codegen`. Without it the CLI and the web UI show different beans for the same file, which is worse than the feature being absent.

### Requirement 1: GraphQL and CLI agree on a bean

**Objective:** As a user of the beans web UI, I want extra front matter keys to appear there too, so that the CLI and the UI do not describe the same bean differently.

#### Acceptance Criteria

1. WHEN a bean carrying extra keys is queried over GraphQL THE API SHALL return those keys
2. WHEN a bean is mutated over GraphQL THE API SHALL preserve every extra key the bean carried before the mutation
3. WHEN the schema changes THE generated code SHALL be regenerated with mise codegen and committed in the same change

#### Success Criteria

- SC-01: A GraphQL query for a bean carrying `release: 0-4-1` returns that pair, and a mutation of its priority over the API leaves the pair in the file.

_Requirements: R-07_

## Recommended Skills

- `tdd`
