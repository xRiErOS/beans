# Project Setup and Introspection

This page covers the commands that initialize a beans project and introspect the CLI itself: `init`, `path`, `prime`, `version`, `help`, and `completion`. Use it to bootstrap a new project, hand the CLI to an AI agent, or wire up shell completions.

## `beans init`

`beans init` creates a `.beans` directory and writes a fresh `.beans.yml` config file in the current directory, giving the project a store to write beans into. Run it once per project root: rerunning it without `--beans-path` overwrites `.beans.yml` with generated defaults, so preserve or review any project-specific configuration first.

Flags: `--profile string` selects a starting type profile (`classic`, `complex`, `simple`, `todo`) that determines the default set of bean types and statuses written into `.beans.yml`; `--json` prints the result as JSON instead of the human-readable confirmation line.

```sh
mkdir demo
cd demo
beans init --profile classic
```

See `../project-profiles.md` for what each profile actually configures.

## `beans path`

`beans path` prints the store path the current invocation resolves to, which is useful for scripts that should not reimplement the CLI's resolution order. An explicit relative `--beans-path` is returned unchanged; config- and default-derived paths are normally absolute. Resolution follows `--beans-path`, then a found `.beans.yml` (including `--config`), then `BEANS_PATH` when no config was found, then the default `.beans` directory.

```
beans --beans-path ./demo/.beans path
```

## `beans prime`

`beans prime` outputs a prompt that primes AI coding agents on how to use the beans CLI to manage project issues, covering recipes for starting work, finding work with `next`, handling blocked beans, completing work, scrapping work, and tagging. It has no flags beyond the global ones and is meant to be pasted into an agent's system prompt or referenced from a project's `AGENTS.md`/`CLAUDE.md` file; see `../agent-integration.md` for the full recipe set and how it is intended to be used.

```
beans prime
```

## `beans version`

`beans version` shows version information for the binary: the version string, the commit, and the build date. `--json` emits the same information as a JSON object with `version`, `commit`, `date`, and `custom_front_matter` fields instead of the two human-readable lines.

```
beans version --json
```

## `beans help`

`beans help [command]` prints the same help text a command would show for `--help`, and `beans help` alone lists every top-level command; it takes no flags of its own beyond `-h`/`--help`.

```
beans help create
```

## `beans completion`

`beans completion [shell]` generates a shell autocompletion script for `bash`, `fish`, `powershell`, or `zsh`; each shell subcommand's own `--help` documents exactly how to source or install the generated script for that shell. The parent command has no flags beyond the global ones and exists only to group the four shell subcommands.

```
source <(beans completion bash)
```

## Related documentation

- [Installation](../installation.md)
- [Quick Start](../quick-start.md)
- [Configuration](../configuration.md)
- [Agent Integration](../agent-integration.md)
- [Project Profiles](../project-profiles.md)
- [Inspection and Search](inspection-and-search.md)
- [Lifecycle](lifecycle.md)
