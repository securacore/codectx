# codectx — Documentation Compiler for AI-Driven Development

**codectx** is a CLI tool and package manager that compiles human-written markdown documentation into an AI-optimized knowledge structure. Your AI coding tools get precise, searchable, token-efficient access to project knowledge — instead of loading entire files and hoping for the best.

---

## The Problem

AI-driven development tools — Claude Code, Cursor, Copilot, and others — consume documentation by loading files into their context window. This approach has three fundamental failures:

**Token waste.** Loading a 5,000-token file when only 500 tokens are relevant burns 90% of the context window budget on irrelevant content. Across a working session, this adds up to thousands of wasted tokens that could carry task-relevant information.

**Retrieval imprecision.** Without structured search, the AI either loads everything (expensive) or guesses which files to load (unreliable). There is no principled way to match a developer's query to the most relevant section of documentation.

**Context loss between sessions.** When an AI session ends or context resets, all understanding of the project's documentation is lost. The next session starts from zero with no way to resume where it left off.

## The Solution

codectx treats documentation like source code — it gets compiled. You write markdown. The compiler transforms it into chunked, indexed, taxonomy-enriched artifacts that AI tools can search and consume with surgical precision.

```
Your markdown docs
       |
       v
  codectx compile
       |
       v
  Token-counted chunks + BM25F search index + taxonomy + session context
       |
       v
  AI searches with `codectx query` → gets ranked results
  AI reads with `codectx generate` → gets assembled documents
```

The AI interacts with the compiled output through the CLI. It searches for what it needs, retrieves exactly the chunks that matter, and consumes them at a fraction of the token cost of loading raw files.

---

## Key Capabilities

### Intelligent Search, Not File Loading

codectx builds a [BM25F search index](search-and-retrieval.md) over your compiled documentation. When the AI needs information, it searches — and gets ranked results scored by relevance, not a directory listing of files to guess from.

The search pipeline includes **taxonomy-based query expansion** (so "auth" finds documentation about "authentication"), **field-weighted scoring** (headings count more than body text), **multi-index fusion** (instructions, reasoning, and system docs scored independently then merged), and **graph-based re-ranking** (adjacent and cross-referenced chunks boost each other).

Every retrieval decision is deterministic and traceable. No embedding models, no vector databases, no black-box similarity scores.

### Compiled, Not Loaded

The [compilation pipeline](how-it-works.md) transforms raw markdown through a multi-stage process: parsing, stripping unnecessary formatting, splitting into token-counted semantic chunks, building search indexes, extracting a controlled vocabulary, generating navigation manifests, and assembling session context.

Only changed files are reprocessed on subsequent compiles. The expensive work scales with your changes, not your total documentation size.

### Token-Aware by Design

Every decision in the system is measured in tokens, not words or characters. Chunk boundaries, session context budgets, search result metadata — all reported in the unit that actually matters for AI context windows.

When you [configure session context](session-context.md), you set a token budget and see exactly how much each document consumes. When the AI receives search results, each chunk includes its token count so it can make budget-aware decisions about what to load.

### Structured Documentation with Intent

codectx organizes documentation into [four content types](documentation-structure.md):

- **Foundation** — Technology-agnostic guidance that survives a stack change. Engineering principles, coding standards, architectural philosophy.
- **Topics** — Technology-specific documentation. How specific systems, frameworks, and libraries work in your project.
- **Plans** — Living documentation that tracks work in progress with [resumable AI workflows](plans.md).
- **Prompts** — Pre-crafted AI instructions that codify tribal knowledge about how to instruct the AI effectively.

Every documentation file can have a companion `.spec.md` file that captures the *reasoning* behind the documentation. When the AI encounters situations the docs don't explicitly cover, the spec files give it the intent behind the instructions — enabling reasoning by analogy rather than guessing.

### Session Context That Persists

Your most important documentation gets compiled into a single [session context document](session-context.md) that the AI reads at the start of every session. Engineering principles, coding standards, architectural guidelines — always loaded, always available, with zero manual effort per session.

The session context is assembled from your `always_loaded` configuration with a token budget you control. Add local docs or documentation from installed packages. The compiler warns if you exceed your budget.

### Package Manager for Documentation

Documentation is shareable. codectx includes a [package manager](package-manager.md) backed by GitHub repositories. Install documentation packages that cover frameworks, patterns, and standards. Publish your own packages for your team or the community.

```bash
codectx search "react patterns"
codectx add react-patterns@community
codectx install
codectx compile
```

Packages contain pure markdown content. When you install a package, your local compiler processes it alongside your own docs — one set of compilation rules for everything. Packages support versioning, transitive dependencies, and a lock file for deterministic builds.

### Resumable Multi-Step Plans

Complex tasks shouldn't lose progress when an AI session ends. [Plans](plans.md) are living documentation with state tracking. Each step records which documentation the AI searched for, which chunks it loaded, and what progress was made.

When a plan resumes, codectx detects whether the underlying documentation has changed. If nothing changed, it replays the exact context instantly. If docs drifted, it tells the AI which dependencies changed and provides the stored search queries for re-execution against updated content.

### Full Audit Trail

Every query and every generated document is [tracked](history-and-caching.md). You can see what the AI searched for, what it loaded, which tool called it, and what session it was part of. Generated documents are cached — requesting the same chunks against the same compilation state returns instantly from cache.

Usage metrics track token consumption by caller and model, with local (per-machine) and global (per-project, version-controlled) scopes.

### Works With Every AI Tool

codectx generates [entry point files](ai-tool-integration.md) for Claude Code (`CLAUDE.md`), GitHub Copilot (`copilot-instructions.md`), Cursor (`.cursorrules`), and generic agents (`AGENTS.md`). Each entry point directs the AI to read the compiled session context and use `codectx query` and `codectx generate` for documentation access.

The compiled artifacts are model-agnostic. The same compiled output works with any AI model. Model-specific behavior is handled through [configuration](configuration.md).

---

## Quick Start

Install codectx and set up your first project in minutes. See the [Getting Started guide](getting-started.md) for a complete walkthrough.

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | bash

# Initialize in your project
codectx init

# Write some documentation, then compile
codectx compile

# Search your compiled docs
codectx query "authentication patterns"

# Assemble specific chunks into a reading document
codectx generate "obj:a1b2c3.01,obj:a1b2c3.02"
```

---

## Documentation

| Topic | Description |
|-------|-------------|
| [Getting Started](getting-started.md) | Install, initialize, compile, and query your first project |
| [How It Works](how-it-works.md) | The compilation pipeline from markdown to searchable chunks |
| [Documentation Structure](documentation-structure.md) | Foundation, topics, plans, prompts, and the .spec.md convention |
| [Search and Retrieval](search-and-retrieval.md) | BM25F scoring, taxonomy expansion, rank fusion, and graph re-ranking |
| [Session Context](session-context.md) | Always-loaded context, token budgets, and context assembly |
| [Package Manager](package-manager.md) | Installing, publishing, and managing documentation packages |
| [Plans](plans.md) | Resumable AI workflows with state tracking and drift detection |
| [History and Caching](history-and-caching.md) | Query/generate history, document caching, and usage metrics |
| [CLI Reference](cli-reference.md) | Complete command reference with flags and examples |
| [Configuration](configuration.md) | codectx.yml, ai.yml, preferences.yml, and local overrides |
| [AI Tool Integration](ai-tool-integration.md) | Entry point files and how AI tools interact with codectx |

---

## Why Not Vector RAG?

codectx deliberately avoids vector embeddings (RAG) for documentation retrieval:

- **Accuracy**: Optimized vector retrieval pipelines achieve correct chunk retrieval below 60% of the time. BM25F with taxonomy expansion achieves deterministic, explainable results.
- **Context preservation**: Vector chunking destroys structural context. codectx preserves heading hierarchies, adjacency relationships, and cross-references.
- **Debuggability**: When BM25 returns wrong results, you can trace exactly why — term frequencies, field weights, taxonomy mappings. Vector search failures require understanding embedding distances in high-dimensional space.
- **Zero infrastructure**: No embedding model, no vector database, no GPU. codectx compiles to files on disk and searches in-memory.

Documentation uses consistent terminology. Exact keyword matching with taxonomy-powered synonym expansion is the right default for instructional content where precision matters.
