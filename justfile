# beans dev CLI — just ist der Einstieg (dev-cli-canon), mise bleibt die
# Build-Implementierung des Upstream-Forks.

# Zielverzeichnis der Installation. NICHT ~/.local/bin wie `mise run install`:
# in Eriks PATH gewinnt /opt/homebrew/bin, und dort liegt das aktive Binary.
bin_dir := env_var_or_default("BEANS_BIN_DIR", "/opt/homebrew/bin")

# Remote to push release tags to. This workspace keeps `origin` on the
# read-only hmans/beans upstream and `fork` on the writable xRiErOS/beans;
# a plain clone of the fork itself has no separate `fork` remote, so this
# picks `fork` when present and falls back to `origin` otherwise.
release_remote := env_var_or_default("BEANS_RELEASE_REMOTE", `git remote get-url fork >/dev/null 2>&1 && echo fork || echo origin`)

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
    cd frontend && mise exec -- pnpm test {{ARGS}}

# Type- and a11y-check the frontend (svelte-check)
check-web:
    cd frontend && mise exec -- pnpm check

# Run the frontend end-to-end suite, e.g. `just test-e2e e2e/filter.spec.ts`
test-e2e ARGS='':
    mise run build:embed
    cd frontend && mise exec -- pnpm test:e2e {{ARGS}}

# Run the Go test suite under the race detector
test-race ARGS='./...':
    go test -race {{ARGS}}

# Validate .goreleaser.yaml (schema, templates) without building anything
release-check:
    mise exec goreleaser@latest -- goreleaser check

# Build every release platform locally without publishing (goreleaser --snapshot)
release-snapshot: release-check
    mise run build:embed
    mise exec goreleaser@latest -- goreleaser release --snapshot --clean --skip=publish

# Cut+push a release tag (patch|minor|major) via svu, then watch Actions
release LEVEL='patch': test release-snapshot
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -n "$(git status --porcelain)" ]; then
        echo "working tree not clean, aborting" >&2
        exit 1
    fi
    mise run "release:{{LEVEL}}"
    NEW_TAG=$(git describe --tags --abbrev=0)
    echo "About to push $(git branch --show-current) and tag ${NEW_TAG} to remote '{{release_remote}}'."
    read -p "Continue? [y/N] " confirm
    case "$confirm" in
        y|Y) ;;
        *)
            echo "Aborted (tag ${NEW_TAG} was created locally; delete with: git tag -d ${NEW_TAG})." >&2
            exit 1
            ;;
    esac
    git push {{release_remote}} HEAD
    git push {{release_remote}} "$NEW_TAG"
    gh run watch -R xRiErOS/beans

# Watch the most recent release Actions run without cutting a new release
release-watch:
    gh run watch -R xRiErOS/beans

# Remove local goreleaser snapshot output (dist/ is gitignored)
release-clean:
    rm -rf dist
