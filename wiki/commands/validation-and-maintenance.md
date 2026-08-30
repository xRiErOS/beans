# Validation and Maintenance

This page documents `beans check`, the fork's completion-policy validation, and `beans archive`, the command that moves finished beans out of the active working set. Run `beans check` before trusting a store's integrity, and `beans archive` to keep a long-lived project's `.beans` directory small.

## `beans check`

`beans check` validates two independent things in one pass: the project's configuration, and the integrity of the bean files it describes.

```
$ beans check
Configuration
  ✓ Statuses defined (5 hardcoded)
  ✓ Default status 'draft' exists
  ✓ Default type 'task' is valid
  ✓ Every bean carries a known type
  ✓ All status colors valid
  ✓ All type colors valid
  ✓ Prefix consistency valid

Bean Links
  ✓ No link issues found

All checks passed
```

Configuration checks cover: that `default_status` and `default_type` name entries that actually exist, that every bean on disk carries a type the current configuration still defines (a config edit or profile switch can leave beans behind with a type nothing recognizes anymore), that every configured status and type `color` names a valid theme tone, and that the configured ID prefix (`beans.prefix`) matches what the bean files on disk actually carry — archived beans are excluded from that comparison, since a project can rename its live prefix without needing to touch history.

Bean-link checks cover three classes of structural problem, all evaluated together:

- **Broken links** — a link field (`blocking`, `blocked_by`, `parent`, and similar) pointing at an ID that does not exist.
- **Self-references** — a bean linking to itself.
- **Circular dependencies** — a cycle in `blocking`/`blocked_by` or `parent` relationships.

`--fix` removes broken links and self-references automatically; cycles cannot be auto-fixed and are always reported for manual resolution, even with `--fix` set. Flags:

| Flag | Description |
|---|---|
| `--fix` | Automatically remove broken links and self-references |
| `--json` | Output as JSON |
| `--strict` | Count policy warnings (see below) as issues, so the command exits 1 when any exist |

`--json` output is the same envelope every command uses, with `check`-specific fields:

```json
{
  "success": true,
  "config_errors": null,
  "bean_issues": { "broken_links": [], "self_links": [], "cycles": [] }
}
```

`config_errors` is a flat list of human-readable strings, one per configuration problem found; `bean_issues` mirrors the link-check sections shown in the plain-text output. On a real failure `success` is `false` and the process exits 1 regardless of `--json`.

## Completion-policy validation

When a project configures `beans.require_fields_on` in `.beans.yml` (see [Configuration Reference](../configuration.md)), `beans check` gains a third section, `Policy`, that re-validates the policy against every bean's current on-disk state:

```
Policy
  ! ae67: status "completed" missing required field(s): summary
```

This section exists because the policy is primarily enforced at write time — `beans complete`, `beans update --status`, and the other status-changing commands refuse the write up front when a required field would be left unset:

```
$ beans complete i9qh
Error: status "completed" requires front matter field(s) summary: supply them in the same write (e.g. `beans complete i9qh --commit HEAD`)
Run 'beans complete --help' for usage.
```

`beans check` catches what the write-time guard cannot: a bean edited directly on disk (bypassing the CLI), or a policy added or tightened after beans already sit in the now-noncompliant status. A policy gap is reported as a warning, not a hard failure, because it describes an existing project state rather than a rejected write — by default it does not affect the process exit code. Pass `--strict` to treat these warnings as issues that fail the command (exit 1), which is the right mode for a CI check that must enforce the policy going forward.

When `beans.commit_field` names a field that also appears in `require_fields_on`, `check` additionally verifies that every recorded value resolves to a real commit in the current git repository; a value that doesn't (or a check run outside a git repository, where verification is skipped entirely) is reported as its own policy warning.

## `beans archive`

`beans archive` moves every bean whose status is configured as an archive status (`completed` or `scrapped` by default) from the active `.beans` directory into `.beans/archive/`.

```
$ beans archive
Archived 1 bean(s) to .beans/archive/
```

Archiving is a filesystem move, not a state change: an archived bean keeps its status, relationships, and full front matter, and remains visible to `beans list`, `beans show`, `beans graphql`, and `beans check` at its path under `archive/`. Archiving keeps the main directory small; it does not delete or hide history. There is no `unarchive` command. Updating an archived bean currently leaves its file in `archive/`; move the file back to the main beans directory manually when you want it active again.

`beans archive --json` returns the standard message envelope (`{"success": true, "message": "Archived N bean(s) to .beans/archive/"}` or `{"success": true, "message": "No beans to archive"}` when nothing qualified).

## Related documentation

- [Querying and Automation](querying-and-automation.md)
- [Configuration Reference](../configuration.md)
- [Web UI and API](../web-ui-and-api.md)
- [Troubleshooting](../troubleshooting.md)
