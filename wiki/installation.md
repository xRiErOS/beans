# Installation

This page covers the four ways to install the xRiErOS fork of Beans and explains the difference between its three binaries.

## Origin

Beans was originally created by [hmans](https://github.com/hmans/beans); this fork, maintained at [xRiErOS/beans](https://github.com/xRiErOS/beans), is an actively maintained independent continuation of that project. The fork's Go module path is `github.com/xRiErOS/beans`.

## Installation channels

All four channels install the same three binaries described below. Pick one:

### Homebrew (macOS)

```sh
brew install xRiErOS/beans/beans
```

Upgrade with `brew upgrade --cask beans`.

### Install script (Linux, macOS)

```sh
curl -fsSL https://raw.githubusercontent.com/xRiErOS/beans/main/install.sh | sh
```

Configurable via environment variables: `BEANS_VERSION` (default: latest release), `BEANS_BIN_DIR` (default: `$HOME/.local/bin`), `BEANS_REPO` (default: `xRiErOS/beans`).

### go install (Linux, macOS, Windows)

```sh
go install github.com/xRiErOS/beans/cmd/beans@latest
go install github.com/xRiErOS/beans/cmd/beans-tui@latest
```

A `beans-serve` built this way embeds no web assets — the repository tracks only a placeholder (`internal/web/dist/.gitkeep`) so the `//go:embed` directive in `internal/web/embed.go` compiles, but the actual frontend build is not part of the module source. Requesting `/` returns 404. Use Homebrew, the install script, or a release archive for a working `beans-serve`.

A binary installed this way still reports a real version, commit, and date: `internal/version/buildinfo.go` reads them from `runtime/debug.ReadBuildInfo()` when the build did not receive `-ldflags` (which `go install` never does).

### GitHub Releases

Download the `linux`/`darwin`/`windows` × `amd64`/`arm64` (`i386` on Linux only) archive for your platform from the [Releases page](https://github.com/xRiErOS/beans/releases), verify against the accompanying `checksums.txt`, and extract `beans`, `beans-serve`, `beans-tui` onto your `PATH`.

## Prerequisites (build from source only)

Install [mise](https://mise.jdx.dev/getting-started.html) first. The repository's `mise.toml` declares the Go, Node.js, and pnpm toolchains the build needs and installs them; the current Go module requires Go 1.24.6.

## The three binaries

The fork builds three separate binaries, one per entrypoint under `cmd/`: `beans` (`cmd/beans`, the CLI you use for day-to-day issue tracking), `beans-serve` (`cmd/beans-serve`, the standalone web UI and GraphQL API server), and `beans-tui` (`cmd/beans-tui`, the interactive terminal UI carried over from upstream).

Each binary is independent; you only need `beans-serve` if you want the web UI, and `beans-tui` if you want the retained terminal UI, alongside the required `beans` CLI.

## Building from source

Clone the fork, let mise install the declared toolchain, install dependencies and generated code, then run the repository's canonical build:

```sh
git clone https://github.com/xRiErOS/beans.git
cd beans

mise install
mise run setup
mise run build
```

The build pipeline compiles the Svelte frontend into `internal/web/dist` before building the Go entrypoints, and stamps version, commit, and build date into the binaries. A plain `go build ./cmd/beans` also compiles, because the repository tracks a placeholder in `internal/web/dist` that satisfies the embed directive, but it produces an unstamped binary, and a `beans-serve` built that way embeds no web assets and serves 404 for the browser UI.

`mise run build` writes the stamped `beans`, `beans-serve`, and `beans-tui` binaries to the repository root. Install them in a user-owned directory on `PATH`:

```sh
install -d "$HOME/.local/bin"
install -m 755 beans beans-serve beans-tui "$HOME/.local/bin/"
export PATH="$HOME/.local/bin:$PATH"
```

Persist the `PATH` addition in your shell configuration if `$HOME/.local/bin` is not already present.

## Verifying the build

Confirm that the installed CLI is reachable and carries the version, commit, and build date injected by the build task:

```sh
beans version
beans --help
```

A plain local `go build` does not inject those linker values (it also has no embedded frontend, see the `go install` section above); use `mise run build` for a fully stamped binary with the web UI, or one of the release-artifact channels above.

## License

The fork is distributed under the Apache License 2.0, the same license as the upstream project; see `LICENSE` in the repository root.

## Related documentation

- [Quick Start](quick-start.md)
- [Configuration](configuration.md)
- [Separate Binaries](commands/separate-binaries.md)
- [Web UI and API](web-ui-and-api.md)
- [TUI Companion](tui-companion.md)
- [Fork Lineage](fork-lineage.md)
- [Compatibility and Upgrading](compatibility-and-upgrading.md)
- [Troubleshooting](troubleshooting.md)
