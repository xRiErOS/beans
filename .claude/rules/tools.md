# Tooling

Entry point is `just` (`just --list`) — it wraps `mise` for the two most common side tasks (`just build`, `just test`) plus `just install`, which builds and installs `beans`/`beans-serve`/`beans-tui` to `BEANS_BIN_DIR` (default `/opt/homebrew/bin`, where the active binary on this machine actually resolves). `mise` remains the underlying build implementation for everything else — use it directly for anything not covered by the justfile.

All build/dev/test tasks use `mise`. Key commands:

- `mise dev` — start the development environment (backend on :22880, frontend on :5173, with hot reload)
- `mise build` — build the `./beans` executable (includes frontend embed)
- `mise beans` — compile and run the beans CLI (use instead of `go run` or `./beans`)
- `mise test` — run all Go tests
- `mise test:e2e` — run Playwright e2e tests
- `mise codegen` — regenerate GraphQL code after schema changes
- `mise setup` — install all dependencies and generate code (first-time setup)
