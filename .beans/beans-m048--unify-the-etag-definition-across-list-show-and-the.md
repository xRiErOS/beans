---
# beans-m048
title: Unify the etag definition across list, show and the write path
status: todo
type: bug
priority: high
created_at: 2026-08-10T16:29:06Z
updated_at: 2026-08-10T16:29:06Z
parent: beans-xqsq
---

## Steps

1. Find where the etag is computed on each of the three paths and write down what each one hashes.
2. Decide the ONE definition. It has to be stable under a read and sensitive to every field a write can change — including the body, since `--body-append` is a write like any other.
3. Route all three through it.
4. Tests as listed in the parent's acceptance block.

## Repro

Any bean, untouched store:

```
beans list --json | jq -r '.[] | select(.id=="<id>") | .etag'
beans show --json <id> | jq -r '.etag'
beans update <id> --priority normal --if-match "<either value>"
```

On 2026-08-10 the third command reported a third value for `SPF-34aa` in the Sproutling store.

## Notes

The parent carries the measurement and the affected consumer.
