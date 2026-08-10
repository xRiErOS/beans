---
# beans-n3hl
title: Parse keeps unknown front matter keys in Bean.Extra
status: completed
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:16:50Z
parent: beans-2ark
---

Add `Extra map[string]any` to `Bean` (`pkg/bean/bean.go:136`) tagged `yaml:"-" json:"extra,omitempty"`. In `Parse` (`pkg/bean/bean.go:185`), parse the front matter a second time into a `map[string]any`, subtract the keys the `frontMatter` struct already owns (`pkg/bean/bean.go:170`), and keep the remainder. The reader is consumed by the first pass — buffer it before parsing.

Scalars are the case that matters. Lists and maps do not have to be settable from the CLI, but they must not break on the way through.

### Requirement 1: Unknown front matter keys survive parsing

**Objective:** As a planning tool author, I want beans to keep front matter keys it does not know, so that a container can carry a sentence that a tag cannot hold.

#### Acceptance Criteria

1. WHEN Parse reads a bean file carrying keys outside the known schema THE parser SHALL store every such key and its value in Bean.Extra
2. WHEN Parse reads a bean file THE parser SHALL leave every known key on its own typed field and SHALL keep it out of Bean.Extra
3. WHERE an unknown key holds a list or a map THE parser SHALL preserve the value unchanged
4. WHEN Parse reads a bean file with no unknown keys THE parser SHALL leave Bean.Extra empty

#### Success Criteria

- SC-01: A bean file carrying three extra scalars, one list and one map parses into a Bean whose Extra holds exactly those five entries with values equal to the file, while Title, Status, Type and Priority stay on their own fields.

_Requirements: R-01_

## Recommended Skills

- `tdd`


## Summary of Changes

`Bean.Extra map[string]any` (`pkg/bean/bean.go:169`, `yaml:"-" json:"extra,omitempty"`) added. `Parse` (`pkg/bean/bean.go:202`) now buffers the reader via `io.ReadAll`, runs `frontmatter.Parse` twice against the buffered bytes — once into the typed `frontMatter` struct, once into `map[string]any` — and copies every key not in `knownFrontMatterKeys` into `Extra`.

### Deviation: map key/value normalization

The `adrg/frontmatter` library decodes YAML via `gopkg.in/yaml.v2`, which produces `map[interface{}]interface{}` for nested mappings — a type `encoding/json` cannot marshal (it would error at `MarshalJSON` time, breaking the GraphQL/CLI API surface that reads `Bean.Extra`). Added `normalizeYAMLValue` (`pkg/bean/bean.go:187`) to recursively convert `map[interface{}]interface{}` to `map[string]any` and normalize nested `[]interface{}` elements. Values are unchanged (AC3 holds); only the Go/JSON-safe container type differs from a raw yaml.v2 decode.

## Test-Output

RED (compile failure, `Bean.Extra` undefined):
```
pkg/bean/bean_test.go:1751:11: b.Extra undefined (type *Bean has no field or method Extra)
FAIL	github.com/hmans/beans/pkg/bean [build failed]
```

GREEN:
```
=== RUN   TestParseExtraFrontMatter
--- PASS: TestParseExtraFrontMatter (0.00s)
=== RUN   TestParseNoExtraFrontMatter
--- PASS: TestParseNoExtraFrontMatter (0.00s)
PASS
ok  	github.com/hmans/beans/pkg/bean	0.288s
```

## Smoke

`go build ./...` clean. `go test ./...` — all packages `ok`, none failed (ran directly, not via `mise test`: the `mise test` pipeline's `codegen`/`build:frontend` step fails on a pre-existing, already-uncommitted `frontend/pnpm-workspace.yaml` unrelated to this task — see BRIEFING gotchas. That same interrupted `mise test` run also transiently deleted `internal/graph/generated.go` and `pkg/beangraph/model/models_gen.go`; both were restored via `git checkout --` before re-verifying, unrelated to this task's diff).

## Notes for T(n+1)

`beans-54rb` (Render outputs Extra, sorted by key) can now rely on `Bean.Extra` existing and being `map[string]any` with JSON-safe nested values (no `map[interface{}]interface{}` to handle).
