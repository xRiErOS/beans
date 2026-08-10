---
# beans-7ohz
title: Expose extra keys in list --json
status: todo
type: task
priority: high
created_at: 2026-08-10T10:05:30Z
updated_at: 2026-08-10T10:05:30Z
parent: beans-2ark
blocked_by:
    - beans-54rb
---

`beans list --json` emits a fixed schema today: id, slug, path, title, status, type, priority, tags, parent, created_at, updated_at, etag. An extra key would stay invisible even once it survives the round trip.

### Requirement 1: JSON output carries the extra keys

**Objective:** As a tool reading beans as data, I want the extra front matter keys in the JSON output, so that a generated plan can be built without reading every bean file.

#### Acceptance Criteria

1. WHEN list --json runs over a bean carrying extra keys THE CLI SHALL emit those keys as an object under the field extra
2. WHEN list --json runs over a bean carrying no extra keys THE CLI SHALL omit the field extra
3. WHEN show --json runs over a bean carrying extra keys THE CLI SHALL emit them under the same field name as list --json

#### Success Criteria

- SC-01: `beans list --json` over a store with one bean carrying `release: 0-4-1` yields an entry whose `extra.release` equals `0-4-1`, and beans without extra keys carry no `extra` field.

_Requirements: R-05_

## Recommended Skills

- `tdd`
