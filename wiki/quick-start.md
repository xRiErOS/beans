# Quick Start

This page initializes a disposable project, builds a milestone/epic/feature hierarchy, and drives it through the core lifecycle. After initialization, every command uses an explicit `--beans-path`, so inherited `BEANS_PATH` values cannot redirect the example.

## Prerequisites

A working `beans` binary on your `PATH`; see [Installation](installation.md) if you have not built one yet.

## Step 1: create a disposable project directory

```bash
mkdir /tmp/beans-quickstart-demo
cd /tmp/beans-quickstart-demo
```

## Step 2: initialize with an explicit profile

`beans init --profile` cannot be combined with `--beans-path`, because a profile writes `.beans.yml` while an explicit store path intentionally skips that file. Run the profiled initialization inside the disposable directory itself.

```bash
beans init --profile classic
```

The `classic` profile defines five types: `milestone`, `epic`, `feature`, `bug`, and `task`, which is what the hierarchy below uses. `beans init --help` lists the other available profiles (`complex`, `simple`, `todo`).

## Step 3: build a small hierarchy

Every following command passes `--beans-path .beans` explicitly, pointing at the directory `init` just created, rather than relying on the config file it wrote or any environment variable.

Create a milestone, then an epic underneath it, then a feature and a bug underneath the epic:

```bash
beans --beans-path .beans create "Launch v1" --type milestone
beans --beans-path .beans create "Onboarding flow" --type epic --parent <milestone-id>
beans --beans-path .beans create "Add welcome email" --type feature --parent <epic-id> --priority high
beans --beans-path .beans create "Fix typo in welcome banner" --type bug --parent <epic-id>
```

Each `create` call prints `Created <id> <filename>`; substitute the generated IDs for `<milestone-id>` and `<epic-id>` in subsequent commands.

## Step 4: list what exists

```bash
beans --beans-path .beans list
```


## Step 5: find the next bean to work on

```bash
beans --beans-path .beans next
```

`next` picks the highest-priority bean that is not blocked and not already in progress, completed, scrapped, or a draft; in this example that is the high-priority feature created in step 3.

## Step 6: start it

```bash
beans --beans-path .beans start beans-quickstart-demo-l5py
```

This marks the bean's `status` as `in-progress` and prints its full contents.

## Step 7: complete it

```bash
beans --beans-path .beans complete beans-quickstart-demo-l5py --summary "Welcome email shipped"
```

`--summary` records what changed; `complete` also accepts `--commit` to record a git ref alongside the completion.

## Step 8: view the roadmap

```bash
beans --beans-path .beans roadmap --format tty --include-done
```


`--include-done` is required because the feature you completed would otherwise be hidden. The terminal view renders the milestone, epic, and both children as a styled tree with a legend, confirming the hierarchy end to end.

## Cleanup

The example store lives entirely under `/tmp/beans-quickstart-demo`; remove it when you are done with `rm -rf /tmp/beans-quickstart-demo`.

## Related documentation

- [Installation](installation.md)
- [Project Setup](commands/project-setup.md)
- [Lifecycle](commands/lifecycle.md)
- [Inspection and Search](commands/inspection-and-search.md)
- [Planning and Reporting](commands/planning-and-reporting.md)
- [Organization and Relations](commands/organization-and-relations.md)
- [Project Profiles](project-profiles.md)
- [Feature Overview](feature-overview.md)
- [Configuration](configuration.md)
