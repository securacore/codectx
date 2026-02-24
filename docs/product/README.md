# Product Architecture

codectx is a documentation package manager for AI-driven development. It manages structured, distributable documentation packages that AI agents consume as operational instructions. The core design principle is metadata-first navigation: AI loads a lightweight data map (YAML) that indexes all available documentation, then selectively loads only what the current task requires. This minimizes token usage while giving AI complete awareness of available context.

For the reasoning behind the architecture, see [spec/README.md](spec/README.md). For validation schemas, see [schemas/](../schemas/).

## Data Map Concept

Every layer of the system has a data map file that serves as a navigation index for AI. Source packages use `package.yml`; the compiled output uses `manifest.yml`. The data map lists every documentation entry, its dependencies, its loading rules, and its file path. AI reads the data map first and loads documentation on demand.

The loading flow for any AI session:

1. AI opens the entry point file (e.g., `CLAUDE.md`). It contains a single line pointing to `.codectx/README.md`.
2. AI loads `README.md`, which describes the loading protocol and links to `manifest.yml`.
3. AI loads `manifest.yml`. This is the compiled data map indexing all available documentation.
4. AI loads foundation documents marked `load: always`. This is the minimal initialization context.
5. As the task progresses, AI consults the data map to locate and load relevant topics, prompts, or plans.
6. For plans, AI reads `state.yml` first to assess status without loading the full plan.

This approach ensures AI never loads documentation blindly. The data map is the navigation layer that makes documentation consumption token-efficient.

## Package Format

Every source documentation package follows the same structure, whether it is the project's local documentation or an installed dependency. The compiled output uses a distinct format (see [Compiled Output Format](#compiled-output-format)).

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

  .codectx/                         # compiled documentation set (gitignored)
    manifest.yml                    # compiled data map (AI consumes this)
    README.md                       # generated loading protocol
    heuristics.yml                  # sidecar: size/token stats (not loaded by AI)
    preferences.yml                 # user preferences (preserved across compiles)
    objects/                        # content-addressed file store
      {16-char-sha256}.md
    state/                          # mutable plan state files
      {plan-id}.yml
    manifests/                      # sub-manifests (only when decomposed)
      foundation.yml
      topics.yml
      prompts.yml
      plans.yml

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
- `.codectx/` is the compiled output. It uses a distinct compiled format with content-addressed objects and provenance tracking. It is gitignored.
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

The `codectx compile` command builds the compiled documentation set from all active documentation. The process has six stages.

### 1. Combination

Merge documentation from all active sources. Active sources include:

- The local package (project's own documentation in `docs/`)
- Activated entries from installed packages in `docs/packages/`

Only entries that are activated in `codectx.yml` are included. Entries with duplicate IDs are resolved by precedence (local wins, then config order) and content hash (identical content = silent dedup; different content = warning, precedence wins).

### 2. Content-Addressed Storage

Each documentation file is stored in the `objects/` directory using a 16-character SHA256 prefix as the filename (e.g., `objects/a1b2c3d4e5f67890.md`). Identical content across packages produces the same hash, giving natural deduplication. Plan state files are stored separately in `state/` because they are mutable.

### 3. Manifest Generation

Build a compiled `manifest.yml` from the unified entries. Each entry references its content-addressed object path (e.g., `objects/a1b2c3.md`) instead of a relative file path, and includes a `source` field for provenance tracking (`"local"` or `"name@author"`).

### 4. Heuristics

Generate `heuristics.yml`, a sidecar metadata file containing size estimates, token counts, and per-package statistics. The heuristics file is not part of the AI loading protocol; it is used by tooling and by the generated README for richer context. See [Heuristics](#heuristics).

### 5. Decomposition

If the documentation set exceeds decomposition thresholds (entries > 500, bytes > 50 KB, or tokens > 100k), the manifest is split into per-section sub-manifests stored in `manifests/`. The root `manifest.yml` retains only always-load foundation entries and `ManifestRef` pointers to the sub-manifests. AI loads the root manifest first, then loads sub-manifests on demand. See [Manifest Decomposition](#manifest-decomposition).

### 6. README and Lock

Generate `README.md` from the unified manifest and heuristics (includes token estimates, section summaries, and loading protocol). Generate `codectx.lock` with resolved versions, checksums, and full activation state.

## Compiled Output Format

The compiled output in `.codectx/` uses a distinct format from source packages. It is validated by [compiled.schema.json](../schemas/compiled.schema.json).

<rules>

- `manifest.yml` is the compiled data map. Each entry has an `object` field (content-addressed path), a `source` field (provenance), and standard `depends_on`/`required_by` edges.
- `objects/` is a flat directory of content-addressed files. Filenames are 16-character SHA256 prefixes with `.md` extension. This provides 64 bits of collision resistance.
- `state/` contains mutable plan state files (`{plan-id}.yml`). These are not content-addressed because they change between compiles.
- `heuristics.yml` is a sidecar metadata file. It is not loaded by AI agents. Validated by [heuristics.schema.json](../schemas/heuristics.schema.json).
- `README.md` is dynamically generated with the loading protocol, section summaries, and token estimates.
- `preferences.yml` stores user-specific settings (e.g., auto-compile). It is preserved across recompiles.
- When decomposed, `manifests/` contains per-section sub-manifests. The root `manifest.yml` holds always-load entries and `ManifestRef` pointers.

</rules>

## Heuristics

The `heuristics.yml` sidecar provides aggregate metadata about the compiled documentation set. It is generated during compilation and is not part of the AI loading protocol.

Contents:
- **Totals**: Entry count, unique object count, total bytes, estimated tokens, always-load count.
- **Sections**: Per-section stats (entries, bytes, tokens, always-load count for foundation).
- **Packages**: Per-package stats (entries, bytes, tokens) with local package listed first.

Token estimates use ~4 characters per token as a rough conversion factor.

## Manifest Decomposition

When the documentation set exceeds any of the following thresholds, the compiled manifest is decomposed into sub-manifests:

- Entries > 500
- Total bytes > 50 KB
- Estimated tokens > 100,000

Decomposition splits at the section level (level 1). Each non-empty section gets its own sub-manifest file in `manifests/` (e.g., `manifests/topics.yml`). Foundation entries with `load: always` remain inlined in the root `manifest.yml` so AI can load them without additional file reads. The root manifest contains `ManifestRef` entries that describe each sub-manifest (section, path, entry count, estimated tokens).

Sub-manifests use the same `manifest.yml` schema, enabling recursive decomposition at deeper levels (e.g., by source package within a section) if needed in the future.

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
- Each entry point file contains a single line referencing `.codectx/README.md`. The README then directs AI to `manifest.yml` and the loading protocol.
- Before creating a new entry point file, `codectx link` renames any existing file to `[file].[timestamp].bak` to preserve the original.
- The entry point file is the only thing the AI tool needs to find the documentation. From the README and data map, the AI navigates the full documentation set.
- `codectx link` is a separate command from `codectx compile`. It is run once after initial setup or when the output directory changes.

</rules>
