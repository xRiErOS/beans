# Project Profiles

Project profiles are named type-table presets that `beans init --profile` expands into a project's `.beans.yml` at creation time. Each profile picks a hierarchy shape — how many container ranks exist above a leaf, and what those ranks are called — so a new project starts with a type table that matches how its team actually organizes work, instead of always inheriting the built-in default.

## How a profile is applied

Running `beans init --profile <name>` resolves the named profile before creating anything on disk, so an unknown profile name fails cleanly with no half-initialized project left behind. A valid profile writes its full type list into the new `.beans.yml` under `types`, together with `types_exclusive: true`, which tells `beans` that this list is the complete type table rather than a set of overrides layered onto the built-in defaults. `--profile` cannot be combined with `--beans-path`, because `--beans-path` skips writing a `.beans.yml` entirely and a profile has nowhere to be written. The available profile names are `classic`, `todo`, `simple`, and `complex`; passing anything else surfaces the same four names in the error message.

## Profiles expand once, not forever

The type list a profile writes is a one-time expansion, not a live reference back to the profile definition. Once `.beans.yml` exists, its `types` block is the project's own data: a future change to how the `complex` profile is defined in a newer `beans` release will never reach back into an already-initialized project and silently move or rename its types. This matters for teams that adopt beans early and expect their configuration to stay exactly as they set it up, and it matters for anyone diffing `.beans.yml` in version control, since the file always shows the real, current type table rather than an indirection that requires cross-referencing profile source code to understand.

## classic

The `classic` profile reproduces the built-in defaults exactly: `beans init --profile classic` is indistinguishable from a plain `beans init` with no profile flag at all. Its hierarchy is a single chain: `milestone` (rank 1) contains `epic` (rank 2), which contains `feature` (rank 3), which is worked on directly alongside the leaf types `bug` and `task` (both rank 4, the leaf rank). This is the profile every existing beans store already runs on, so it is the safe choice for migrating an established project onto explicit profile config without changing its type semantics.

Use `classic` when a team already ships software in the traditional milestone-and-epic style and just wants that structure written out explicitly. Use `classic` for onboarding a new repository into an existing multi-project beans workflow where every other project uses the default hierarchy. Use `classic` as the baseline to fork from when writing a hand-edited `.beans.yml` that starts from familiar names before adding project-specific overrides.

## todo

The `todo` profile is a flat list with a single leaf type, `task` (rank 4, the leaf rank), and no container ranks at all. There is no milestone, no epic, nothing to group work under except tags; every bean sits at the same level.

Choose `todo` for a personal project or a solo side effort where a hierarchy of milestones and epics would be pure overhead for a handful of open items. Choose `todo` for a lightweight backlog that only ever needs "what do I need to do" without any notion of releases or themes. Choose `todo` for a short-lived spike or prototype repository where the tracker itself should stay out of the way.

## simple

The `simple` profile splits rank 2 by subject matter instead of by a single generic container: `epic` is the thematic container and `feature` is the user-facing capability, and both share rank 2 so neither nests inside the other. Rank 1 offers two entry points: `milestone` for work that is meant to ship, and `bucket` for a parking lot of topics that might be picked up someday. The `bucket` type sets `roadmap: false`, so it and everything under it are excluded from `beans roadmap` and `beans milestones` even though it otherwise behaves like a normal rank-1 container. The leaf rank keeps `bug` and `task`, unchanged from `classic`.

Use `simple` for a small product team that wants "epic" and "feature" as two parallel ways to slice rank 2 without inventing an extra hierarchy level. Use `simple` for a project that wants a visible parking lot of someday-ideas that never clutters the roadmap or milestone list until they graduate into a real milestone. Use `simple` for a team migrating off a flatter tracker that is ready for containers but not for the four-rank depth of `complex`.

## complex

The `complex` profile is the deepest of the four, and it splits rank 2 by where a piece of work's value comes from rather than by subject matter: first whether it delivers customer benefit at all, then, within "yes", whether it is new capability or an improvement to something that already exists. Rank 1 holds `release` (a version that ships and gets a release tag) and `bucket` (a someday parking lot, again with `roadmap: false`, excluded from `beans roadmap` and `beans milestones`). Rank 2 holds three siblings: `feature` (new capability with customer benefit), `improvement` (a change to existing capability that the customer notices), and `chore` (internal work with no customer-visible benefit — tooling, upgrades, documentation, refactoring, new or existing alike). The leaf rank adds `story` (a demonstrable slice of user-visible work) alongside the familiar `bug` and `task`.

Use `complex` for a product organization that reports on customer value separately from internal engineering investment and needs that split baked into the type table itself. Use `complex` for a team that ships tagged releases and wants `release` as the top-level rank-1 container instead of an open-ended `milestone`. Use `complex` for a mature project that has already outgrown two-level hierarchies and needs `story` as a distinct, demonstrable leaf type separate from `task`.

## Choosing between profiles

The four profiles differ in how many ranks they use and what a container rank is named, not in what a leaf bean looks like day to day: every profile keeps `bug` and `task` (or, for `todo`, `task` alone) as ordinary leaf work items. Picking a profile is a one-time structural decision made at `beans init` time; changing hierarchy shape afterward means hand-editing the `types` block in `.beans.yml`; the `roadmap` field is what determines whether a rank-1 container's whole subtree is excluded from `beans roadmap` and `beans milestones`, and it is what makes both `simple`'s and `complex`'s `bucket` type a genuine parking lot rather than just another visible container.

## Related documentation

- [Feature Overview](feature-overview.md)
