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

# Name des WIP-Binaries für Testläufe. Bewusst ein eigener Name statt eines
# zweiten Zielverzeichnisses: in Eriks PATH gewinnt {{bin_dir}}, ein dort
# abgelegter Teststand würde also das reguläre Binary verdecken.
wip_bin := env_var_or_default("BEANS_WIP_BIN", "beanst")

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

# Liegt neben dem regulären Binary statt es zu ersetzen: nach dem Merge auf
# main wirkt `just install` wieder regulär, dieses Rezept ist kein Ersatz
# dafür. Stempelt denselben Version/Commit/Date-Stand wie `build`.

# Teststand des ungemergten CLI als {{wip_bin}} nach {{bin_dir}} installieren
install-wip: build
    install -m 755 beans "{{bin_dir}}/{{wip_bin}}"
    @"{{bin_dir}}/{{wip_bin}}" version

# Ein liegengebliebenes {{wip_bin}} täuscht in einem späteren Testlauf einen
# Stand vor, den das Repo nicht mehr hat.

# Teststand {{wip_bin}} wieder aus {{bin_dir}} entfernen
uninstall-wip:
    rm -f "{{bin_dir}}/{{wip_bin}}"

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
    RUN_ID=""
    for i in $(seq 1 20); do
        RUN_ID=$(gh run list -R xRiErOS/beans -w release --branch "$NEW_TAG" -L1 --json databaseId -q '.[0].databaseId' 2>/dev/null || true)
        [ -n "$RUN_ID" ] && break
        sleep 3
    done
    if [ -z "$RUN_ID" ]; then
        echo "no release run appeared yet, check https://github.com/xRiErOS/beans/actions" >&2
        exit 1
    fi
    gh run watch -R xRiErOS/beans "$RUN_ID" --exit-status

# Watch the most recent release Actions run without cutting a new release
release-watch:
    #!/usr/bin/env bash
    set -euo pipefail
    RUN_ID=$(gh run list -R xRiErOS/beans -w release -L1 --json databaseId -q '.[0].databaseId')
    gh run watch -R xRiErOS/beans "$RUN_ID"

# Remove local goreleaser snapshot output (dist/ is gitignored)
release-clean:
    rm -rf dist
