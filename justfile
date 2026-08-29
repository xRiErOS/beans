# beans dev CLI — just ist der Einstieg (dev-cli-canon), mise bleibt die
# Build-Implementierung des Upstream-Forks.

# Zielverzeichnis der Installation. NICHT ~/.local/bin wie `mise run install`:
# in Eriks PATH gewinnt /opt/homebrew/bin, und dort liegt das aktive Binary.
bin_dir := env_var_or_default("BEANS_BIN_DIR", "/opt/homebrew/bin")

# List available recipes
default:
    @just --list

# Build beans, beans-serve und beans-tui mit Version/Commit/Date-Stempel
build:
    mise run build

# Build und Installation nach {{bin_dir}} — macht den Stand systemweit wirksam
install: build
    install -m 755 beans beans-serve beans-tui "{{bin_dir}}/"
    @"{{bin_dir}}/beans" version

# Run the Go test suite, e.g. `just test ./internal/bean/...`
test ARGS='./...':
    go test {{ARGS}}

# Run the frontend unit tests, e.g. `just test-web --project server`
test-web ARGS='':
    cd frontend && pnpm test {{ARGS}}

# Type- and a11y-check the frontend (svelte-check)
check-web:
    cd frontend && pnpm check

# Run the frontend end-to-end suite, e.g. `just test-e2e e2e/filter.spec.ts`
test-e2e ARGS='':
    mise run build:embed
    cd frontend && pnpm test:e2e {{ARGS}}
