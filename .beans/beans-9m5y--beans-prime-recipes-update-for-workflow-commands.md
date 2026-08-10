---
# beans-9m5y
title: 'beans prime: recipes update for workflow commands'
status: completed
type: task
priority: normal
created_at: 2026-08-10T12:40:50Z
updated_at: 2026-08-10T14:30:29Z
parent: beans-mmyp
blocked_by:
    - beans-0ajg
    - beans-r780
    - beans-jvkq
    - beans-m364
    - beans-18db
    - beans-p17z
---

Once complete/scrap/start/next exist as real commands, rewrite
internal/commands/prompt.tmpl so prime hands agents one authoritative
command per situation instead of derived list/update incantations.

## Scope

- Consolidate the scattered recipe prose (EXTREMELY_IMPORTANT task-lifecycle
  block, "Finding Work" section) into a single `## Recipes` section:
  use-case -> command, one line each, using the new first-class commands
  (beans start <id>, beans complete <id>, beans next, ...) instead of
  beans update <id> -s in-progress etc.
- Verify every use-case mentioned in the template maps to a command that
  actually exists by then (close the gap this task exists to close).
- Document beans ready / beans blocked as the existing list --ready /
  list --is-blocked flags, unless a sibling task promotes them to
  first-class commands before this one starts.
- Remove or wire up the dead GraphQLSchema field in promptData/prime.go
  (currently computed in prime.go, never referenced in prompt.tmpl).
- Add substring assertions to prime_test.go per the established pattern
  (see the comment at prime_test.go:34-37) for each new recipe, so future
  CLI features cannot ship undocumented in prime again.

## Not in scope

- Building the workflow commands themselves (tracked in sibling tasks).
- Cutting the release (tracked in the sibling release task).

## Acceptance

- [ ] prompt.tmpl has one `## Recipes` section covering: starting work,
      finding work, completing work, scrapping work, checking progress,
      listing milestones, handling blocked work
- [ ] No recipe references a command that does not exist in this binary
- [ ] GraphQLSchema field is either rendered in the template or removed
      from promptData/prime.go
- [ ] prime_test.go has a substring assertion per new recipe command

## Summary of Changes

Rewrote internal/commands/prompt.tmpl into a single ## Recipes section documenting all six new commands plus existing list --ready/--is-blocked flags; removed the dead GraphQLSchema field from prime.go. Final whole-branch review found and fixed stale Relationships prose (bare 'ready' command reference, false claim that 'start' respects blocking) and added a concrete search command to the Starting-work recipe.
