---
# beans-n3hl
title: Parse keeps unknown front matter keys in Bean.Extra
status: todo
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
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
