# Product Architecture

codectx is a documentation package manager for AI-driven development. It manages structured, distributable documentation packages that AI agents consume as operational instructions. The core design principle is metadata-first navigation: AI loads a lightweight data map (YAML) that indexes all available documentation, then selectively loads only what the current task requires. This minimizes token usage while giving AI complete awareness of available context.

For the reasoning behind every architectural decision, see [spec/README.md](spec/README.md).

## Feature Documentation

| Document | Description |
|---|---|
| [Package Format](packages.md) | Package structure, manifest format, entry types, naming, resolution, and plan state tracking |
| [Compilation](compilation.md) | Compile process, content-addressed storage, heuristics, decomposition, and lock file |
| [Configuration](configuration.md) | `codectx.yml` settings, activation, conflict handling, deduplication, and directory layout |
| [AI Integration](ai-integration.md) | Entry point linking, the loading protocol, supported tools, and watch mode |
| [Design Decisions](spec/README.md) | Reasoning behind every architectural choice |

## Core Concepts

### Data Map

Every layer of the system has a data map file that serves as a navigation index for AI. Source packages use `package.yml`; the compiled output uses `manifest.yml`. The data map lists every documentation entry, its dependencies, its loading rules, and its file path. AI reads the data map first and loads documentation on demand.

### Content-Addressed Objects

Documentation files are stored by their content hash (16-character SHA256 prefix). Identical content across packages produces the same hash, giving natural deduplication. This means installing multiple packages that share common documentation incurs no storage or token overhead.

### Precedence and Deduplication

When multiple packages provide entries with the same ID, conflicts are resolved by precedence (local package wins, then config order) and content hash (identical content = silent dedup; different content = warning, precedence wins).

### Manifest Decomposition

Large documentation sets are automatically split into per-section sub-manifests. AI loads the root manifest (with always-load foundation entries) first, then loads sub-manifests on demand. This keeps the initial token cost low regardless of total documentation size.

## Validation Schemas

| Schema | Validates |
|---|---|
| [codectx.schema.json](../schemas/codectx.schema.json) | `codectx.yml` |
| [package.schema.json](../schemas/package.schema.json) | `package.yml` |
| [state.schema.json](../schemas/state.schema.json) | `state.yml` (plan state) |
| [compiled.schema.json](../schemas/compiled.schema.json) | Compiled `manifest.yml` |
| [heuristics.schema.json](../schemas/heuristics.schema.json) | `heuristics.yml` |
