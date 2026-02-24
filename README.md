<p align="center">
  <img src=".assets/logo.svg" alt="codectx" width="180" />
</p>

<h1 align="center">codectx</h1>

<p align="center">
  The package manager for AI code documentation.
</p>

---

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
- **Live recompilation** -- `codectx watch` monitors your documentation and recompiles on every change
- **Manifest decomposition** -- large documentation sets split into on-demand sub-manifests
- **Plan state tracking** -- AI reads lightweight state files to triage plans without loading full documents
- **Multi-tool support** -- generates entry points for Claude Code, Cursor, GitHub Copilot, and OpenCode
- **Interactive package search** -- find packages in the registry with fuzzy search
- **Semver resolution** -- version ranges, Git tag resolution, and lock file reproducibility
- **Conflict detection** -- warns when packages overlap and lets you choose what to activate
- **Background update checks** -- notifies you when a newer version is available without blocking

## Install

**Shell (Linux / macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | sh
```

**Go:**

```bash
go install github.com/securacore/codectx@latest
```

Binaries are published for Linux and macOS on `amd64` and `arm64`. The install script detects your architecture, downloads the correct binary, and verifies its SHA256 checksum.

## Quick Start

```bash
codectx init my-project          # Scaffold project structure
cd my-project
# Write documentation in docs/foundation/, docs/topics/, etc.
codectx add react@org            # Install a shared package
codectx compile                  # Compile everything into .codectx/
codectx link                     # Create AI tool entry points
```

Your AI assistant now loads your documentation automatically.

## Commands

| Command | Description |
|---|---|
| `codectx init [name]` | Scaffold a new documentation project |
| `codectx add <package>` | Install and activate a documentation package |
| `codectx compile` | Compile all active documentation into `.codectx/` |
| `codectx link` | Create entry point files for AI tools |
| `codectx search [query]` | Search the package registry |
| `codectx watch` | Watch for changes and recompile automatically |
| `codectx version` | Print the installed version |

The `codectx-` prefix is always implied. `react@org` resolves to the `codectx-react` repository owned by `org` on GitHub.

## Documentation

Detailed documentation for each feature:

| Document | Description |
|---|---|
| [Package Format](docs/product/packages.md) | Package structure, manifest format, and entry types |
| [Compilation](docs/product/compilation.md) | Compile process, content addressing, deduplication, and decomposition |
| [Configuration](docs/product/configuration.md) | `codectx.yml` settings, activation, and conflict handling |
| [AI Integration](docs/product/ai-integration.md) | How AI tools load documentation and the data map protocol |
| [Architecture](docs/product/README.md) | System architecture overview and design principles |
| [Design Decisions](docs/product/spec/README.md) | Reasoning behind every architectural choice |

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, testing, and release process.
