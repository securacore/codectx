# codectx

## The Problem

AI coding assistants are stateless. Every session starts from zero. They don't know your architecture, your naming conventions, your deployment patterns, or the decisions your team made last quarter. You end up repeating yourself in every prompt, and the AI still hallucinates patterns that don't match your codebase.

This is a documentation problem. If the right context were available at the start of every session, the AI would follow your conventions instead of inventing its own. But getting documentation into AI tools is harder than it sounds:

- **Documentation isn't portable.** Useful conventions, patterns, and instructions that work in one project have to be manually recreated in every new project. There's no install/import mechanism for documentation the way there is for code.
- **Scope creep wastes tokens.** Without a way to selectively load documentation, you either include everything (wasting tokens on irrelevant context) or maintain separate documentation sets per project (maintenance nightmare).
- **Documentation drifts.** Without shared, versioned sources, each project's documentation evolves independently. The same convention gets documented five different ways across five projects.
- **Not optimized for AI.** Documentation written for humans has redundancy, verbose formatting, and structural patterns that waste tokens. AI needs dense, structured input — not prose.
- **Not reproducible.** If your documentation setup works on your machine but a teammate can't reproduce it, you've just created a new class of "works on my machine" bugs.

codectx solves all of these. It's a documentation package manager and compiler that turns structured Markdown into token-efficient context that AI agents load automatically.

## How It Works

### 1. Write Documentation in Markdown

Organize your project's conventions, architecture decisions, prompts, and plans in a simple directory structure:

```
docs/
  manifest.yml          # Data map: what exists, how to load it
  foundation/          # Always-loaded context (conventions, principles)
  topics/              # On-demand reference (React, Go, API patterns)
  prompts/             # Executable instructions (commit, review, deploy)
  plans/               # Implementation plans with state tracking
```

Each document is a plain Markdown file. A YAML manifest indexes everything and declares relationships between entries.

### 2. Install Shared Packages

Reuse documentation across projects. Community packages provide conventions for frameworks, languages, and tools. Install them like any other dependency:

```bash
codectx add react@org           # Framework conventions
codectx add typescript@org      # Language standards
codectx add go@org              # Go patterns and idioms
```

Packages are versioned with semver. Version ranges, lock files, and conflict detection work the same way you'd expect from any package manager.

### 3. Compile

One command compiles all documentation — local and installed — into a single optimized output:

```bash
codectx compile
```

Compilation produces a `.codectx/` directory containing content-addressed objects, a compiled data map, and heuristics. When CMDX compression is enabled (the default for new projects), compiled objects are encoded into a compact text format that reduces token usage by ~25% on structured content.

### 4. Link to AI Tools

```bash
codectx link       # Connect to Claude, Cursor, Copilot, OpenCode
```

AI loads a lightweight data map first, then pulls in only the documentation relevant to the current task. Large documentation sets are automatically decomposed into sections so AI never loads more than it needs.

## Core Design Principles

**Content-addressed storage.** Documentation files are stored by their content hash. Identical content across packages is deduplicated automatically. Installing multiple packages that share common documentation incurs no storage or token overhead.

**Activation-based scoping.** You choose exactly which entries from each package are active. Unused topics are never compiled, never loaded, never wasted.

**Progressive loading.** AI reads a small data map first, then loads foundation context, then pulls additional topics on demand. The initial token cost is constant regardless of total documentation size.

**Portable packages.** Any project's documentation can be extracted and published as a package. The same format works for local docs, installed dependencies, and compiled output.

**CMDX compression.** A purpose-built text codec compresses Markdown into a compact `@TAG` format with dictionary compression and domain-specific blocks. The output is still human-readable — not binary — and round-trips back to equivalent Markdown. See [Compression](compression.md).

**Reproducibility.** Lock files record exact resolved versions and checksums. `codectx add --lockfile` reproduces the exact package state on any machine.

## Features

- **Content-addressed storage** — identical content across packages is deduplicated automatically
- **CMDX compression** — purpose-built codec reduces token usage by ~25% on structured content
- **Link rewriting** — relative markdown links are rewritten to content-addressed references, with unresolvable references clearly marked
- **Smart compilation** — fingerprint-based change detection skips recompilation when nothing changed
- **Live recompilation** — `codectx watch` monitors documentation and recompiles on every change
- **Manifest decomposition** — large documentation sets split into on-demand sub-manifests
- **Preference management** — `codectx set` for user-local settings (compression, AI provider, model class)
- **Plan state tracking** — AI reads lightweight state files to triage plans without loading full documents
- **Multi-tool support** — generates entry points for Claude Code, Cursor, GitHub Copilot, and OpenCode
- **Interactive package search** — find packages in the registry with fuzzy search
- **Semver resolution** — version ranges, Git tag resolution, and lock file reproducibility
- **Conflict detection** — warns when packages overlap and lets you choose what to activate
- **AI model class targeting** — configure the documentation compatibility baseline for your AI tier

## Feature Documentation

| Document | Description |
|---|---|
| [Package Format](packages.md) | Package structure, manifest format, entry types, naming, and resolution |
| [Compilation](compilation.md) | Compile process, content-addressed storage, compression, link rewriting, and decomposition |
| [Compression](compression.md) | CMDX codec: algorithm, tag format, dictionary compression, and benchmarks |
| [Preference Management](set-command.md) | The `codectx set` command and user-local preferences |
| [Configuration](configuration.md) | `codectx.yml` settings, activation, conflict handling, and directory layout |
| [AI Integration](ai-integration.md) | Entry point linking, the loading protocol, model class targeting, and watch mode |
| [Design Decisions](spec/README.md) | Reasoning behind every architectural choice |

## Architecture

The core design principle is **metadata-first navigation**: AI loads a lightweight data map (a YAML file called `manifest.yml`) that indexes all available documentation, then selectively loads only what the current task requires. This minimizes token usage while giving AI complete awareness of available context.

### Understanding the Two Manifests

There are two distinct `manifest.yml` files in a codectx project, and they serve different purposes:

| File | Location | Purpose | Who creates it |
|---|---|---|---|
| **Source manifest** | `docs/manifest.yml` | Indexes your source documentation. Lists entries with file paths relative to `docs/`. Also present in each installed package at `docs/packages/[name]@[author]/manifest.yml`. | You (or `codectx init`). Auto-updated by `codectx compile` when new files are discovered. |
| **Compiled manifest** | `.codectx/manifest.yml` | Indexes the compiled output. Lists entries with content-addressed object paths (e.g., `objects/a1b2c3d4.md`). Includes provenance tracking and dependency edges. | Generated by `codectx compile`. Never edited manually. |

The source manifest is what you author and maintain. The compiled manifest is what AI actually reads. Compilation transforms one into the other, merging all active sources, deduplicating content, rewriting links, and generating provenance metadata.

For installed packages: the package repository may or may not ship a `manifest.yml`. If it does, codectx uses it. If it doesn't, codectx auto-discovers entries from the directory structure during compilation. Either way works.

### Core Concepts

**Content-Addressed Objects.** Documentation files are stored by their content hash (16-character SHA256 prefix). Identical content across packages produces the same hash, giving natural deduplication. Installing multiple packages that share common documentation incurs no storage or token overhead.

**Link Rewriting.** Source documentation files contain relative markdown links (`[hooks](hooks.md)`, `[philosophy](../../foundation/philosophy.md)`). Since compiled objects live in a flat directory, those relative paths would break. During compilation, all markdown links are rewritten to reference other objects by their content hash. Links to files not in the compiled set use the `unresolved:` URI scheme so AI knows the reference exists but the target isn't available. Source files are never modified.

**Precedence and Deduplication.** When multiple packages provide entries with the same ID, conflicts are resolved by precedence (local package wins, then config order) and content hash (identical content = silent dedup; different content = warning, precedence wins).

**Manifest Decomposition.** Large documentation sets are automatically split into per-section sub-manifests. AI loads the root manifest (with always-load foundation entries) first, then loads sub-manifests on demand. This keeps the initial token cost low regardless of total documentation size.

**CMDX Compression.** When compression is enabled, compiled objects are encoded from Markdown into a compact `@TAG` text format before storage. The content-addressed filename uses the `.cmdx` extension instead of `.md`. Compression happens transparently during compilation — source files are never modified. See [Compression](compression.md) for the full technical deep-dive.

## Validation Schemas

| Schema | Validates |
|---|---|
| [codectx.schema.json](../schemas/codectx.schema.json) | `codectx.yml` |
| [manifest.schema.json](../schemas/manifest.schema.json) | Source `manifest.yml` (in `docs/` and packages) |
| `compiled.schema.json` (not yet created) | Compiled `.codectx/manifest.yml` |
| `heuristics.schema.json` (not yet created) | `heuristics.yml` |
| [plan.schema.json](../schemas/plan.schema.json) | `plan.yml` (plan state) |
