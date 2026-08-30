# Installation

This page covers building the xRiErOS fork of Beans from source and explains the difference between its three binaries. It reflects the fork's current build configuration, not the upstream project's published release channel.

## Origin and current distribution state

Beans was originally created by [hmans](https://github.com/hmans/beans); this fork, maintained at [xRiErOS/beans](https://github.com/xRiErOS/beans), is an actively maintained independent continuation of that project.

The fork's `go.mod` still declares its module path as `github.com/hmans/beans`, so `go install github.com/xRiErOS/beans/cmd/beans@latest` does not work: Go's module resolver would fetch the fork's source under a path that does not match its own module declaration and fail. Build from a local clone instead, as shown below.

The repository's `.goreleaser.yaml` builds all three binaries for Linux, macOS, and Windows and configures no Homebrew tap; a tag push runs that pipeline through GitHub Actions and attaches the archives to a GitHub Release of this fork. The fork's [Releases page](https://github.com/xRiErOS/beans/releases) carries no artifacts yet, so building from source as described below is currently the only installation path.

## Prerequisites

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

A plain local `go build` or `go install` does not inject those linker values and is not the documented installation path. Installing the fork directly by remote module path also remains invalid while `go.mod` retains `github.com/hmans/beans`.

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
