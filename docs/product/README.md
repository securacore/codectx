# codectx

AI coding assistants lose context between sessions. They forget your architecture decisions, ignore your conventions, and hallucinate patterns that don't match your codebase. You end up repeating yourself in every prompt.

codectx fixes this. It compiles structured documentation packages into a format AI agents load automatically at the start of every session. Your conventions, architecture, workflows, and plans become persistent context that survives across sessions, tools, and team members.

## How It Works

**Write once, use everywhere.** Document your codebase conventions in simple Markdown files organized by a YAML manifest. codectx compiles them into a token-efficient format that AI agents consume on demand.

```
docs/
  package.yml          # Data map: what exists, how to load it
  foundation/          # Always-loaded context (conventions, principles)
  topics/              # On-demand reference (React, Go, API patterns)
  prompts/             # Executable instructions (commit, review, deploy)
  plans/               # Implementation plans with state tracking
```

**Install shared packages.** Reuse documentation across projects. Community packages provide conventions for frameworks, languages, and tools. Install them like any other dependency.

```bash
codectx add react@org           # Framework conventions
codectx add typescript@org      # Language standards
codectx add go@org              # Go patterns and idioms
```

**Compile and link.** One command compiles all documentation (local and installed) into a single optimized output. Another links it to your AI tools.

```bash
codectx compile    # Build .codectx/ from all active documentation
codectx link       # Connect to Claude, Cursor, Copilot, OpenCode
```

AI loads a lightweight data map first, then pulls in only the documentation relevant to the current task. Large documentation sets are automatically decomposed into sections so AI never loads more than it needs.

## Features

- **Content-addressed storage** -- identical content across packages is deduplicated automatically
- **Smart compilation** -- fingerprint-based change detection skips recompilation when nothing changed
- **Live recompilation** -- `codectx watch` monitors your documentation and recompiles on every change, with a polling heartbeat as a safety net for missed filesystem events
- **Manifest decomposition** -- large documentation sets split into on-demand sub-manifests
- **Plan state tracking** -- AI reads lightweight state files to triage plans without loading full documents
- **Multi-tool support** -- generates entry points for Claude Code, Cursor, GitHub Copilot, and OpenCode
- **Interactive package search** -- find packages in the registry with fuzzy search
- **Semver resolution** -- version ranges, Git tag resolution, and lock file reproducibility
- **Conflict detection** -- warns when packages overlap and lets you choose what to activate
- **Background update checks** -- notifies you when a newer version is available without blocking

## Feature Documentation

| Document | Description |
|---|---|
| [Package Format](packages.md) | Package structure, manifest format, entry types, naming, and resolution |
| [Compilation](compilation.md) | Compile process, content-addressed storage, heuristics, and decomposition |
| [Configuration](configuration.md) | `codectx.yml` settings, activation, conflict handling, and directory layout |
| [AI Integration](ai-integration.md) | Entry point linking, the loading protocol, supported tools, and watch mode |
| [Design Decisions](spec/README.md) | Reasoning behind every architectural choice |

## Architecture

The core design principle is metadata-first navigation: AI loads a lightweight data map (YAML) that indexes all available documentation, then selectively loads only what the current task requires. This minimizes token usage while giving AI complete awareness of available context.

### Core Concepts

**Data Map.** Every layer of the system has a data map file that serves as a navigation index for AI. Source packages use `package.yml`; the compiled output uses `manifest.yml`. The data map lists every documentation entry, its dependencies, its loading rules, and its file path. AI reads the data map first and loads documentation on demand.

**Content-Addressed Objects.** Documentation files are stored by their content hash (16-character SHA256 prefix). Identical content across packages produces the same hash, giving natural deduplication. Installing multiple packages that share common documentation incurs no storage or token overhead.

**Precedence and Deduplication.** When multiple packages provide entries with the same ID, conflicts are resolved by precedence (local package wins, then config order) and content hash (identical content = silent dedup; different content = warning, precedence wins).

**Manifest Decomposition.** Large documentation sets are automatically split into per-section sub-manifests. AI loads the root manifest (with always-load foundation entries) first, then loads sub-manifests on demand. This keeps the initial token cost low regardless of total documentation size.

## Validation Schemas

| Schema | Validates |
|---|---|
| [codectx.schema.json](../schemas/codectx.schema.json) | `codectx.yml` |
| [package.schema.json](../schemas/package.schema.json) | `package.yml` |
| [state.schema.json](../schemas/state.schema.json) | `state.yml` (plan state) |
| [compiled.schema.json](../schemas/compiled.schema.json) | Compiled `manifest.yml` |
| [heuristics.schema.json](../schemas/heuristics.schema.json) | `heuristics.yml` |
