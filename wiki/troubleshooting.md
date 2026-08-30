# Troubleshooting

This page collects symptom-first fixes for the problems people actually hit running beans. Each entry gives a diagnostic command first, then the corrective one, so you can confirm the cause before you act on it.

## Wrong or missing store

Symptom: a command reports `no .beans directory found at <path> (run 'beans init' to create one)`, or `beans show`/`beans list` returns beans you did not expect.

Run `beans path` first. It prints the store directory the current invocation resolved to, following the same precedence every other command uses (`--beans-path`, then a found `.beans.yml`, then `BEANS_PATH` when no config was found, then the default `.beans` directory), so you see exactly what a real command would have used without touching any data. An explicit relative `--beans-path` is echoed unchanged.

If the printed path is wrong, pass the right one explicitly for a single call: `beans list --beans-path /path/to/.beans`. If the wrong path keeps coming back, check for a stray `.beans.yml` above your working directory (the search walks upward) or an exported `BEANS_PATH` taking effect in a repository that has no config file of its own.

If the path is correct but empty or missing while a `.beans.bak-<timestamp>` directory sits next to it, an interrupted rename left the store mid-swap. `beans check` and every other command detect this automatically: with exactly one backup they repair `.beans` from it and log the recovery; with more than one backup they leave `.beans` as an empty placeholder and print each backup's path plus the exact recovery command, e.g. `rm -rf .beans && mv .beans.bak-1730000000000000000 .beans` (the suffix is a nanosecond timestamp). Inspect each backup's contents with `ls .beans.bak-*` before choosing one — `.beans` is already empty at this point, so nothing is lost by picking; the `rm -rf` in that message only clears the empty placeholder, not real data.

## Config discovery picked the wrong file

Symptom: settings you edited (theme, types, statuses, `require_fields_on`) do not seem to apply.

Diagnose with `beans check`, which prints the resolved configuration checks under a `Configuration` heading, including `Default type '<type>' is valid` and any theme or type mismatch. Config discovery searches upward from the current directory for the nearest `.beans.yml` (`LoadFromDirectory`), so a command run from a subdirectory of a monorepo can pick up a different `.beans.yml` than the one you are editing.

Fix by passing the file explicitly: `beans check --config /path/to/.beans.yml`. To confirm which file that command would have found on its own, run `beans path` from the same working directory — a store path under a directory other than the one holding your edited `.beans.yml` means discovery stopped at a different file first.

## Unexpected BEANS_PATH override

Symptom: `beans` operates on a store you did not intend, even though the current directory has its own `.beans.yml`.

`BEANS_PATH` is commonly exported by direnv and inherited into unrelated repositories. The resolver only lets it win when no `.beans.yml` was found for the current invocation; a `.beans.yml` present at or above the working directory always outranks it, because that file is the repository's own declaration of where its store lives. Diagnose with `beans path` and `env | grep BEANS_PATH` (or `printenv BEANS_PATH`) side by side: if `beans path` matches your repository's `.beans` directory even with the env var set, the config file is already winning and there is nothing to fix.

If `beans path` shows the env var's target instead, the repository has no `.beans.yml` of its own yet — run `beans init` in it, which writes a `.beans.yml` and immediately demotes `BEANS_PATH` for every later invocation in that directory. For a single one-off call without creating a config file, override explicitly: `beans list --beans-path ./.beans`.

## No ready bean

Symptom: `beans next` prints "No ready beans found" (or, filtered, "No ready beans found matching …"), or `beans list --ready` returns nothing.

`--ready` and `beans next` both apply the same filter: not blocked, and excluding `in-progress`, `completed`, `scrapped`, and `draft` status (plus any bean whose status is only implicitly terminal because a scrapped/completed ancestor covers it). Diagnose with the two commands the empty message itself names: `beans list --is-blocked` shows what is blocked and by what, and a bare `beans list` shows every bean's actual status. If you passed filters (`--type`, `--tag`, `--parent`), rerun the equivalent `beans list` with the same flags — the empty-result message echoes them back for exactly this purpose.

Once you have found the bean you expected to see, either clear its blocker (see the next section) or move it out of an excluded status with `beans update <id> --status todo`.

## Blocked or parent relationships are wrong

Symptom: `beans list --is-blocked` or `beans check` reports a circular dependency, or a bean you expect to be blocked is not.

Diagnose with `beans check`, which reports every broken link, self-reference, and cycle in both `parent` and `blocking`/`blocked_by` links under its `Bean Links` section, printing cycles as an arrow-joined path, e.g. `Circular dependency: beans-a1b2 → beans-c3d4 → beans-a1b2 (via blocking)`. `beans show <id>` on any bean in the path lists its own `parent`, `blocking`, and `blocked_by` front matter directly.

Fix a broken or self-referencing link with `beans update <id> --remove-blocked-by <target>` (or `--remove-blocking`, `--remove-parent`), naming the relationship the check reported. Cycles cannot be auto-fixed: `beans check --fix` prints `Cannot auto-fix cycle: <path> (via <link type>)`, then includes that cycle in its final count of issues that `require manual intervention`. Break one edge with the matching `--remove-*` flag and rerun `beans check`.

## Completion policy failures

Symptom: `beans complete`, `beans start`, `beans scrap`, or `beans update --status …` fails with `status "completed" requires front matter field(s) commit: supply them in the same write (e.g. `beans complete <id> --commit HEAD`)`.

This is `beans.require_fields_on` in `.beans.yml`, a fork-specific policy that names front matter fields a bean must carry before it can enter a given status. The check runs before any bean in a batch is written, so a violation on one ID leaves every bean in that call untouched — nothing to roll back. Diagnose the full policy with `beans check`, which lists every bean currently missing a required field under its `Policy` section as a warning (pass `--strict` to also make `check` exit non-zero on these warnings).

Fix by supplying the missing field in the same write that changes status: `beans complete <id> --commit HEAD` (or `--set <field>=<value>` for a non-commit field). If the configured commit field references a SHA, `beans check` separately verifies each referenced commit still exists in the current git repository and warns `<id>: commit <sha> not found in this repository` when it does not — outside a git repository it warns `commit verification skipped (not a git repository)` instead of failing.

## Invalid config or front matter

Symptom: any command exits with `loading config: <path>: unknown beans.anchor "…"` or a similarly worded `beans.require_fields_on[…]` error, or `loading beans: loading <path>: parsing front matter: …`.

Config errors are caught eagerly at load, before any command runs: an unrecognized `beans.anchor` value, an empty field name or reserved schema field (`title`, `status`, `type`, …) inside `beans.require_fields_on`, or an unknown status name there all abort with a message naming the exact key and the config file's path. Diagnose by opening the named file at the reported path; the error text names the offending key directly, so no separate lookup is needed.

A bean file's YAML front matter is validated the same way: one malformed `.md` file (e.g. an unquoted colon inside a title) aborts loading for every command, not just the one touching that bean, with the file's path in the error. Diagnose by opening the exact path named after `loading`; fix the YAML by hand (quote the offending value) and rerun the command — there is no `--fix` for a parse error, since the file cannot be safely interpreted until it parses.

## Server binding or startup failures

Symptom: `beans-serve serve` (or `beans-serve serve --port <n>`) exits with `server error: listen tcp :<port>: bind: address already in use`, or a browser request to the UI fails with a CORS error in its console.

Diagnose a port conflict with `lsof -i :<port>` (the default port is 8080, from `server.port` in `.beans.yml` or the `--port`/`-p` flag) to see what already holds it. Fix by picking a free port explicitly: `beans-serve serve --port 8081`, or by stopping the process holding the current one.

A CORS rejection shows up only in the browser (a blocked-by-CORS error in devtools), not in the server's own log, because the check happens per request against the allowed-origins list. Diagnose by checking the origin the server actually allows: `beans-serve serve` prints `[beans] Allowed origins: …` on startup, which defaults to `http://localhost:*` and `http://127.0.0.1:*` (from `server.cors_origins` in `.beans.yml`, or the `--cors-origin` flag, repeatable). Fix by adding the origin you are calling from: `beans-serve serve --cors-origin http://localhost:5173`, or by setting `server.cors_origins` in `.beans.yml` for a persistent list.

## Profile choice at init

Symptom: `beans init --profile <name>` fails with `unknown profile "<name>" (must be classic, complex, simple, todo)`, or `--profile` together with `--beans-path` fails with `--profile writes a .beans.yml, which --beans-path skips; use one or the other`.

Both errors are validated before any file is written, so a typo never leaves a half-initialized project behind. Diagnose by rereading `beans init --help`, which lists the exact profile names accepted at that moment. Fix an initial setup by rerunning `beans init` with a valid `--profile` name in the intended project directory, without `--beans-path`.

Do not rerun `beans init` over an established project without first preserving `.beans.yml`: the command rewrites that file with generated defaults. To change an existing project's profile or type model, edit and review the `types`, `types_exclusive`, and `beans.default_type` settings in `.beans.yml` deliberately; see [Project profiles](project-profiles.md) and [Configuration](configuration.md).

## Migrating from upstream

This fork tracks [hmans/beans](https://github.com/hmans/beans), the original project, and its `.beans.yml`/bean-file formats read and write compatibly with it in the common case. If a store created by upstream behaves unexpectedly here, start with `beans check`, which surfaces the same configuration and link issues either binary would, plus this fork's own additions (`require_fields_on` policy checks, prefix consistency). A `check` failure that names a key upstream never had (`beans.anchor`, `beans.require_fields_on`) means the store simply predates this fork's fields — those keys are additive and safe to leave unset.

See [Fork Lineage](fork-lineage.md) and [Compatibility and Upgrading](compatibility-and-upgrading.md) for the detailed differences between this fork and upstream, including which fields and behaviors are additions rather than replacements.

## Related documentation

- [Configuration](configuration.md)
- [Command Reference: Validation and Maintenance](commands/validation-and-maintenance.md)
- [Command Reference: Organization and Relations](commands/organization-and-relations.md)
- [Command Reference: Planning and Reporting](commands/planning-and-reporting.md)
- [Fork Lineage](fork-lineage.md)
- [Compatibility and Upgrading](compatibility-and-upgrading.md)
- [Quick Start](quick-start.md)
