# Fork-Status (xRiErOS/beans) — unabhängiges Produkt

Dieses Repo ist **Erik-Fork** von `hmans/beans`. Der Upstream-Autor (hmans) hat auf GitHub
öffentlich erklärt, **beans nicht mehr weiterzuentwickeln**. Damit gilt (bestätigt D14):

- **Wir entwickeln beans selbst weiter** — der Fork ist das Produkt, nicht ein PR-Kandidat.
- **Keine Abhängigkeit mehr zu hmans/upstream.** `origin` (hmans) wird nicht mehr verfolgt;
  Push ausschließlich nach `fork` (xRiErOS). Kein Chasing von Upstream-PRs.
- Diese `CLAUDE.md` ist damit **nicht** mehr „Upstream-Datei, nicht anfassen" — sie ist unsere.
  Fork-Delta ist ab jetzt erwünscht, nicht Kostenfaktor.

Der Rest dieser Datei ist die (weiterhin gültige) Projekt-/Codebase-Doku aus dem Upstream.

# What we're building

Beans is an agentic-first issue tracker. Issues ("beans") live as markdown files in a `.beans/` directory inside a project repo. The system has three interfaces:

- **CLI** (`beans` binary) — create, list, update, and query beans from the terminal
- **Terminal TUI** — Bubbletea-based interactive interface
- **Beans UI** (`beans serve`) — SvelteKit SPA served by an embedded Go HTTP server, communicating via GraphQL (queries, mutations, subscriptions over WebSocket)

The Beans UI is the primary development focus. It includes a backlog board, per-bean agent chat (spawning Claude Code processes), git worktree management, file change diffs, and terminal sessions.

# Commits

- Use conventional commit messages ("feat", "fix", "chore", etc.) when making commits.
- Include the relevant bean ID(s) in the commit message (please follow conventional commit conventions, e.g. `Refs: bean-xxxx`).
- Mark commits as "breaking" using the `!` notation when applicable (e.g., `feat!: ...`).
- When making commits, provide a meaningful commit message. The description should be a concise bullet point list of changes made.

# Pull Requests

- When we're working in a PR branch, make separate commits, and update the PR description to reflect the changes made.
- Include the relevant bean ID(s) in the PR title (please follow conventional commit conventions, e.g. `Refs: bean-xxxx`).

# Project Structure

Key packages:

- `pkg/bean/` — Bean data model, parsing, sorting, validation (no I/O)
- `pkg/beancore/` — Core engine: disk I/O, file watching, search indexing, worktree watching
- `internal/graph/` — GraphQL schema and resolvers (the API layer for both Beans UI and CLI)
- `internal/agent/` — Agent session manager: spawns Claude Code processes, parses JSONL output, pub/sub for real-time updates
- `internal/worktree/` — Git worktree lifecycle management
- `internal/terminal/` — PTY session management for embedded terminals
- `internal/commands/` — CLI command implementations (Cobra)
- `frontend/` — SvelteKit SPA (embedded into the Go binary via `//go:embed`)

## GraphQL

- When making changes to the GraphQL schema (`internal/graph/schema.graphqls`), run `mise codegen` to regenerate both backend (`generated.go`) and frontend (`frontend/src/lib/graphql/generated.ts`) types.
- When adding or changing frontend GraphQL operations (queries, mutations, subscriptions), update `frontend/src/lib/graphql/operations.graphql` and run `mise codegen`. Do NOT use inline `gql` strings — all operations must go through codegen for type safety.
- All CLI commands that interact with beans should internally use GraphQL queries/mutations against the local server.
- Subscriptions use WebSocket transport. The `beanChanged` subscription supports `includeInitial: true` to send a full snapshot on connect, avoiding race conditions between initial load and live updates.

## Build

- `mise build` to build a `./beans` executable
- The frontend is built and embedded into the Go binary at compile time

# GraphQL Subscriptions

- When a mutation removes or clears state (e.g., deleting a session), the subscription resolver must still send an explicit "empty" payload to the frontend. Never skip `nil` results with `continue` — the frontend needs to know the state changed.

# Worktree State Architecture

- Git worktrees are created **outside** the main repo, in `~/.beans/worktrees/<project-name>/`. This avoids nested repo confusion and accidental tool/search interference. The location is configurable via `worktree.path` in `.beans.yml`.
- `beans-serve` holds **runtime state** as the authoritative view of all beans. It initializes from main repo disk, then merges in changes from worktrees and the GraphQL API.
- The CLI in a worktree uses the **worktree's local `.beans/`** directory — it does NOT redirect to the main repo. This means worktree agents' bean changes travel with their PR.
- `beans-serve` watches active worktrees' `.beans/` dirs and merges file changes into runtime state as "dirty" (not persisted to main disk).
- The `startWork` mutation uses `WithPersist(false)` — status changes are runtime-only until the PR merges.
- When a PR merges and the bean file lands on main, the main watcher picks it up and the dirty flag clears.
- Each worktree has a **metadata file** (`<id>.meta.json`) stored as a sibling in the worktree root directory (e.g. `~/.beans/worktrees/<project>/<id>.meta.json`). This file persists per-worktree state that must survive server restarts: name, description, allocated port, and last-active timestamp. Use `worktree.Manager.SavePort`/`GetPort` etc. to read and write fields — don't access the file directly.

# Agent Architecture

- The central (main workspace) agent session uses ID `__central__` (defined as `CentralSessionID` in `internal/graph/resolver.go` and `MAIN_WORKSPACE_ID` in `frontend/src/lib/worktrees.svelte.ts`). These must stay in sync — the backend uses this ID to determine work directory and system prompt.
- Worktree agent sessions use the worktree ID as their session ID.

# Renaming Beans

`beans rename` changes a bean's slug, a single bean's ID (cascading refs), or the project-wide ID prefix. Exactly one mode per invocation:

- **slug** — `beans rename <id> --slug "new-slug"` / `--no-slug` (clears the slug, filename becomes `id.md`) / `--reslug` (regenerate from the title). Single-file `os.Rename`; no ref cascade (the slug isn't stored in frontmatter or in any cross-ref).
- **id** — `beans rename <id> <new-id>` (full new ID) or `beans rename <id> --suffix k7x2` (new suffix, keeps the configured prefix; refuses if `<id>` doesn't already start with that prefix). Cascades: rewrites `parent`/`blocking`/`blocked_by` refs across every bean, corrects the `# <id>` comment, applied via an atomic staging+swap of the whole `.beans/` tree (`pkg/beancore/rename.go`).
- **prefix** — `beans rename --prefix "bew-"` project-wide ID rebrand (e.g. shortening `bew_BeWiki-Python-Download-` to `bew-`). Same cascade mechanism as id-mode, plus writes the new prefix to `.beans.yml`. Refuses if any bean's ID doesn't already start with the current configured prefix (avoids silently double-prefixing on a mixed-prefix repo).

Flags (all modes): `--dry-run` prints the plan without writing anything; `--json` renders the plan as JSON instead of the human summary.

Prefix rebrand additionally: requires `--yes` (or an interactive `y`/`N` confirm) before applying, and refuses outright while `beans serve` is listening on the configured port or while any active worktree (`~/.beans/worktrees/<project>/*.meta.json`) exists — verified error text: `beans serve appears to be running on port <port>; stop it before a prefix rebrand`.

**`--json` on apply (non-dry-run) writes two separate JSON documents to stdout, not one.** First the plan document (same shape as `--dry-run --json`: `{"Mode","Changes","RefUpdates","NewPrefix","ConfigWrite"}`), then on its own line a second, unrelated JSON object with the result (`{"success":true,"mode":...,"changed":<n>}`). Verified against the built binary for both `id` and `prefix` modes. A naive `json.loads(stdout)` in a scripting consumer parses only the first document and silently drops the result — read stdout as an NDJSON-like multi-document stream, or parse the last line if only the result is needed. (Collapsing this into a single JSON document is a possible future follow-up, not yet tracked as a bean.)

Non-goals (by design):
- **No auto-git.** All rename modes are plain filesystem operations (`os.Rename` / atomic directory swap) — beans never stages, commits, or runs `git mv`. You stage/commit the changed bean files yourself; git detects renames by content, not by beans telling it to.
- **External ID references are not rewritten.** Anything outside `.beans/` that mentions an old ID — commit messages, docs, SSTD entries — is left as-is after a rename. Out of scope by design, not an oversight.

# Extra rules for our own beans/issues

- Use the `idea` tag for ideas and proposals.

# Testing

- Always write or update tests for the changes you make.

## Unit Tests

- Run all tests: `mise test`
- Run specific package: `go test ./internal/bean/`
- Use table-driven tests following Go conventions

## E2E Tests

- Write or update Playwright e2e tests for any web UI changes.
- Run e2e tests: `mise test:e2e`
- See `frontend/e2e/` for fixtures, page objects, and specs.

## Manual CLI Testing

- `mise beans` will compile and run the beans CLI. Use it instead of building and running `./beans` manually.
- When testing read-only functionality, feel free to use this project's own `.beans/` directory. But for anything that modifies data, create a separate test project directory. All commands support the `--beans-path` flag to specify a custom path.
