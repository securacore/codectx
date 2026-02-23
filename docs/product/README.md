# Product Architecture

codectx is a documentation package manager for AI-driven development. It manages structured, distributable documentation packages that AI agents consume as operational instructions. The core design principle is metadata-first navigation: AI loads a lightweight data map (YAML) that indexes all available documentation, then selectively loads only what the current task requires. This minimizes token usage while giving AI complete awareness of available context.

For the reasoning behind the architecture, see [spec/README.md](spec/README.md). For validation schemas, see [schemas/](../schemas/).

## Data Map Concept

Every layer of the system has a `package.yml` file that serves as a data map. This file is the navigation index for AI: it lists every documentation entry, its dependencies, its loading rules, and its file path. AI reads the data map first and loads documentation on demand.

The loading flow for any AI session:

1. AI opens the entry point file (e.g., `CLAUDE.md`). It contains a single line pointing to the compiled data map.
2. AI loads the compiled `package.yml`. This is a small YAML file that indexes all available documentation.
3. AI loads foundation documents marked `load: always`. This is the minimal initialization context.
4. As the task progresses, AI consults the data map to locate and load relevant topics, prompts, or plans.
5. For plans, AI reads `state.yml` first to assess status without loading the full plan.

This approach ensures AI never loads documentation blindly. The data map is the navigation layer that makes documentation consumption token-efficient.

## Package Format

Every documentation package follows the same structure, whether it is the project's local documentation, an installed dependency, or the compiled output.

<rules>

- Every package has a `package.yml` at its root. This file is the data map. It is validated by [package.schema.json](../schemas/package.schema.json).
- `package.yml` contains package metadata (name, author, version, description) and documentation entries organized into four sections: foundation, topics, prompts, plans.
- Each entry in the data map has a unique ID, a file path relative to the package root, a description, and dependency relationships (`depends_on`/`required_by`).
- Foundation entries have a `load` field that controls when they are loaded into AI context (`always` or `documentation`).
- Plan entries have a `state` field pointing to a `state.yml` file for lightweight status tracking.
- Documentation content lives in optional directories within the package: `foundation/`, `topics/`, `prompts/`, `plans/`.
- Not every package contains all directories. A package may provide only topics, or only prompts, or any combination.

</rules>

Package directory structure:

```text
[package]/
  package.yml
  foundation/
  topics/
  prompts/
  plans/
    [plan-name]/
      README.md
      state.yml
```

## Configuration

`codectx.yml` at the repository root is the sole source of truth for a project's documentation setup. It declares package dependencies, activation state, and build settings. It is validated by [codectx.schema.json](../schemas/codectx.schema.json).

<rules>

- One `codectx.yml` per project, at the repository root.
- `config.docs_dir` sets the documentation source directory. Default: `docs`.
- `config.output_dir` sets the compiled output directory. Default: `.codectx`.
- `packages` is an array of package dependencies. Each entry declares name, author, version, optional source, and activation state.
- The `active` field on each package controls what is included in the compiled output: `all` activates every entry, `none` installs but activates nothing, or a granular object lists specific IDs per section (foundation, topics, prompts, plans).
- When `active` is omitted, the default is `none`. Activation is always explicit.

</rules>

## Directory Layout

A project using codectx has three distinct areas: source documentation, compiled output, and the lock file.

```text
project-root/
  codectx.yml                       # sole source of truth
  codectx.lock                      # resolved state from compile
  CLAUDE.md                         # single-line entry point (created by codectx link)
  AGENTS.md                         # single-line entry point (created by codectx link)

  .codectx/                         # compiled documentation set (checked into git)
    package.yml                     # unified data map
    foundation/
    topics/
    prompts/
    plans/

  docs/                             # documentation source
    package.yml                     # local package data map
    schemas/                        # validation schemas
      codectx.schema.json
      package.schema.json
      state.schema.json
    packages/                       # installed packages
      [name]@[author]/
        package.yml
        topics/
        ...
    foundation/                     # local foundation docs
    topics/                         # local topic docs
    prompts/                        # local prompts
    plans/                          # local plans
```

<rules>

- `docs/` is the source. It contains the local package (the project's own documentation) and installed packages in `docs/packages/`.
- `.codectx/` is the compiled output. It is a self-contained documentation set that follows the package format. It is checked into git.
- `codectx.lock` pins the exact resolved state (versions, checksums, activation) so the compiled output can be reproduced with `codectx add --lockfile`.
- `docs/packages/` contains installed packages, namespaced as `[name]@[author]/`. Each installed package has its own `package.yml`.
- Whether `docs/packages/` is checked into git is the user's choice. The lock file ensures reproducibility regardless.

</rules>

## Package Naming and Resolution

Packages are identified by name and author. Versions use semver.

<rules>

- Package identifier format: `name@author`. The at-sign separates name from author namespace.
- Version is appended with a colon: `name@author:version`.
- Shorthand forms are accepted by the CLI: `name` (latest, unscoped), `name:version` (versioned, unscoped), `name@author` (latest, scoped), `name@author:version` (fully qualified).
- Versions follow semver. Range syntax is supported: `^1.0.0` (compatible), `~1.0.0` (patch-level), `1.0.0` (exact).
- Package resolution is Git-first. Packages are fetched from Git repositories. The source URL is either specified explicitly in `codectx.yml` or inferred from the name and author.
- The system is designed to support a future package registry without breaking changes to the naming convention or configuration format.
- Versions are resolved from Git tags in semver format (e.g., `v1.0.0`).

</rules>

## Compile Process

The `codectx compile` command builds the compiled documentation set from all active documentation. The process has two phases.

### 1. Combination

Merge documentation from all active sources into the output directory. Active sources include:

- The local package (project's own documentation in `docs/`)
- Activated entries from installed packages in `docs/packages/`

Only entries that are activated in `codectx.yml` are included. Inactive entries are excluded from the output.

### 2. Alignment

Reconcile the data maps from all sources into a single unified `package.yml` in the output directory. The alignment step:

- Merges foundation, topics, prompts, and plans entries from all active sources.
- Adjusts file paths to be relative to the compiled output root.
- Preserves `depends_on`/`required_by` symmetry across the merged data map.
- Records the source package for each entry so provenance is traceable.

The compiled output in `.codectx/` is a complete, self-describing documentation set. Its `package.yml` is the master data map that AI tools consume.

`codectx.lock` is generated as a byproduct of compilation. It contains the resolved versions, checksums, and full activation state so the exact compiled output can be reproduced.

## Activation and Conflict Handling

Packages are namespaced by `name@author`, so installed packages never create file-level conflicts. Conflicts only arise at the activation level: when two packages both provide documentation for the same domain (e.g., two packages both provide React conventions) and both are activated.

<rules>

- During `codectx add`, the CLI reads the new package's `package.yml` to inspect its contents.
- The CLI presents an interactive selection interface where the user can activate all entries, activate specific entries, or activate none.
- If activating an entry would create a domain overlap with an already-active entry from another package, the CLI warns the user and prompts for a resolution.
- Activation state is stored in `codectx.yml` and is the authoritative record of what is included in the compiled output.

</rules>

## Plans and State Tracking

Plans are implementation documents that describe what to build and how. Each plan has a `state.yml` file that provides lightweight status tracking so AI can assess plan state without loading the full plan into context.

<rules>

- Each plan is a directory containing `README.md` (the plan content) and `state.yml` (the state tracker).
- `state.yml` is validated by [state.schema.json](../schemas/state.schema.json).
- `state.yml` contains: plan ID, status (`not_started`, `in_progress`, `completed`, `blocked`), optional timestamps (`started_at`, `updated_at`), and a summary (one to three sentences describing current state).
- AI reads `state.yml` first to triage whether to load the full plan. This avoids loading large plan documents unnecessarily.
- Plans are listed in `package.yml` like any other entry, with `depends_on`/`required_by` for dependency tracking.

</rules>

## CLI Commands

The initial command set is minimal. Additional commands (remove, list, update, search) are deferred until usage patterns emerge.

| Command | Purpose |
|---|---|
| `codectx init` | Create `codectx.yml`, `docs/` directory structure, `docs/package.yml`, and `docs/schemas/`. |
| `codectx add <package>` | Fetch a package from its source, install it to `docs/packages/`, prompt for activation, and update `codectx.yml`. Accepts: `name`, `name:version`, `name@author`, `name@author:version`. |
| `codectx compile` | Build the compiled documentation set in `.codectx/` from all active sources. Generate `codectx.lock`. |
| `codectx link` | Create AI tool entry point files. Backs up any existing entry point files as `[file].[timestamp].bak`, then creates new files with a single line referencing the compiled data map. |
| `codectx version` | Display the CLI version. |

## AI Tool Integration

AI tools (Claude Code, Cursor, GitHub Copilot, OpenCode) each have their own entry point file convention. The `codectx link` command bridges the compiled documentation to these tools.

<rules>

- `codectx link` creates entry point files (e.g., `CLAUDE.md`, `AGENTS.md`) at the repository root.
- Each entry point file contains a single line that references the compiled data map in `.codectx/package.yml`.
- Before creating a new entry point file, `codectx link` renames any existing file to `[file].[timestamp].bak` to preserve the original.
- The entry point file is the only thing the AI tool needs to find the documentation. From the data map, the AI navigates the full documentation set.
- `codectx link` is a separate command from `codectx compile`. It is run once after initial setup or when the output directory changes.

</rules>
