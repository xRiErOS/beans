---
# beans-xqsq
title: 'Optimistic locking: the etag is not the same in all three places'
status: todo
type: epic
priority: high
created_at: 2026-08-10T16:28:57Z
updated_at: 2026-08-10T16:28:57Z
parent: beans-xej5
---

## Goal

`--if-match` is usable: the token a caller can read is the token the write path compares against.

## The finding

Measured on 2026-08-10 against the Sproutling store (`SPF-` prefix, `beans 0.4.4-fork`) while building `just roadmap apply`, which writes planning changes back through `beans update`. Three sources of an etag, and they do not agree.

`SPF-34aa`, untouched store, three calls seconds apart:

```
beans list --json   -> 7324924c67a7dbdd
beans show --json   -> 7324924c67a7dbdd
beans update --parent … --if-match 7324924c67a7dbdd
                    -> Error: etag mismatch: provided 7324924c67a7dbdd,
                       current is 54db8405dcda129c
```

`SPF-ukwd`, same session, a different disagreement — here `list` and `show` do not even agree with each other, and `show` happened to match the write path:

```
beans list --json   -> d7296f6fe13ff03b
beans show --json   -> 0f309c7b654ee8d0   (updated_at identical)
beans update … --if-match 0f309c7b654ee8d0   -> accepted
```

So there is no read command whose etag can be handed to `--if-match` with confidence. The value the write path checks is obtainable **only from the error message of a write that has already failed**.

## Why this matters beyond one consumer

An optimistic lock whose token you learn after the failure is not a lock. A consumer has three options, and all three are bad: skip `--if-match` and lose the protection, retry with the etag parsed out of the error message and thereby overwrite exactly what the lock exists to protect, or fail every write.

The Sproutling side took the first option. `scripts/roadmap-apply.mjs` there dropped `--if-match` entirely and now guards concurrency by comparing each bean's `updated_at` against a timestamp recorded when its control file was generated. That covers "someone touched this bean since the file was written" and explicitly does not cover the window between the tool's own read and its write — a named gap that closes when this is fixed. The measurement is written into the file above the call, so whoever fixes this finds the consumer.

## Suspicion, not a diagnosis

The list path presumably hashes the list struct without the body, and the write path hashes the file or the full record. It was not traced in the source; the numbers above are the whole evidence.

## Acceptance

- [ ] One definition of the etag, used by `list`, `show` and the write path alike.
- [ ] A test that reads the etag from each read command and performs a successful `--if-match` write with it — one case per command, so a future divergence fails by name.
- [ ] A test that a genuinely concurrent modification makes `--if-match` fail, so the fix does not turn the lock into a no-op.
- [ ] The Sproutling consumer (`scripts/roadmap-apply.mjs`, bean `SPF-8pk2`) can restore `--if-match`; note the version it needs.

## Notes

Reported from Sproutling, epic `SPF-ukwd`, ERRATA in `SPF-8pk2` and a line in that repo's `roadmap-changes.md`. The data format is shared by at least five stores, so a consumer on an older binary is a second reason to keep the definition in one place.
