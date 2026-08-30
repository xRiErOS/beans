# Separate Binaries

`beans-serve` and `beans-tui` are standalone binaries built from the same repository as `beans`, each wrapping one dedicated application command. The main binary does not advertise runnable `serve` or `tui` subcommands; hidden migration stubs only report that those commands moved to separate executables and exit with an error.

## Why two extra binaries

`beans` registers only the core, file-based subcommands — creation, editing, listing, [validation](validation-and-maintenance.md), and [querying](querying-and-automation.md), among others. `serve` and `tui` are registered separately, each in its own `cmd/` entry point, so that a plain `beans` install carries no web server or terminal-UI dependencies, and a deployment that only needs one of the three doesn't have to ship the others:

| Binary | Wraps | Purpose |
|---|---|---|
| `beans` | every core subcommand | day-to-day CLI use, scripting, agent tool calls |
| `beans-serve` | `serve` | starts the web UI and GraphQL HTTP API — see [Web UI and API](../web-ui-and-api.md) |
| `beans-tui` | `tui` | opens the interactive terminal UI — see [TUI Companion](../tui-companion.md) |

Each binary is a thin `main()` that builds the same root command, registers its one subcommand, and defaults to running that subcommand when invoked with no arguments or with a flag as the first argument — so `beans-serve` alone is equivalent to `beans-serve serve`, and `beans-tui` alone is equivalent to `beans-tui tui`.

Both binaries still accept the two global flags every `beans` command does:

| Flag | Description |
|---|---|
| `--beans-path string` | Path to the data directory (overrides config and `BEANS_PATH`) |
| `--config string` | Path to a config file (default: searches upward for `.beans.yml`) |

Neither binary registers the other's subcommand, and neither registers `version`, `graphql`, `check`, or any other core command — `beans-serve version` and `beans-tui version` both fail with cobra's "unknown command" error. Version and diagnostic commands are only available on the main `beans` binary.

## `beans-serve`

```
$ beans-serve --help
Start the web server

Usage:
  beans-serve serve [flags]

Aliases:
  serve, server, s

Flags:
      --cors-origin strings   Allowed CORS origins (use * to allow all) (default [http://localhost:*,http://127.0.0.1:*])
  -h, --help                  help for serve
  -p, --port int              Port to listen on (default 8080)

Global Flags:
      --beans-path string   Path to data directory (overrides config and BEANS_PATH env var)
      --config string       Path to config file (default: searches upward for .beans.yml)
```

Starting it launches the web UI and GraphQL API on the configured port (default `8080`). Full route list, network binding behavior, and security posture are documented in [Web UI and API](../web-ui-and-api.md); do not run it against a store you don't want reachable from every process on the machine without first reading that page's CORS and origin-checking section.

## `beans-tui`

```
$ beans-tui --help
Opens an interactive terminal user interface for browsing and managing beans.

Usage:
  beans-tui tui [flags]

Flags:
  -h, --help   help for tui

Global Flags:
      --beans-path string   Path to data directory (overrides config and BEANS_PATH env var)
      --config string       Path to config file (default: searches upward for .beans.yml)
```

`beans-tui` opens the retained original full-screen terminal interface. [TUI Companion](../tui-companion.md) explains how this compatibility binary differs from the separately maintained `xRiErOS/beans-tui` project and its `bt` executable.

## Building the separate binaries

Use the repository's canonical source-build pipeline; it generates the embedded frontend before compiling all three entrypoints and stamps version metadata:

```
mise install
mise run setup
mise run build
```

The resulting `beans`, `beans-serve`, and `beans-tui` executables are written to the repository root. See [Installation](../installation.md) for prerequisites and PATH setup.

## Related documentation

- [Web UI and API](../web-ui-and-api.md)
- [TUI Companion](../tui-companion.md)
- [Querying and Automation](querying-and-automation.md)
- [Installation](../installation.md)
