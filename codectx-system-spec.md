# codectx — Documentation Compiler for AI-Driven Development

## System Specification & Implementation Guide (v2)

---

## Phase 1: System Overview & Philosophy

### What This System Is

codectx is a CLI tool and package manager that compiles human-written markdown documentation into an AI-optimized knowledge structure. The compiler transforms raw documentation into chunked, indexed, taxonomy-enriched artifacts that AI coding tools can search, navigate, and consume with minimal token overhead.

### The Core Problem

AI-driven development tools (Claude Code, Cursor, Copilot, etc.) consume documentation by loading files into their context window. This approach has three fundamental failures:

1. **Token waste**: Loading entire files when only a small section is relevant burns context window budget on irrelevant content. A 5,000-token file where only 500 tokens matter wastes 90% of the allocation.

2. **Retrieval imprecision**: Without a structured search mechanism, the AI either loads everything (expensive) or guesses which files to load (unreliable). There is no principled way to match a developer's query to the most relevant documentation section.

3. **Context loss between sessions**: When an AI session ends or context resets, all understanding of the project's documentation is lost. The next session starts from zero with no way to resume where it left off.

### Why Not Vector RAG

The initial design question was whether to use vector embeddings (RAG) or a file-based approach with structured search. The decision was to avoid vector RAG for several reasons:

- **Accuracy**: Practitioners report that even optimized vector retrieval pipelines achieve correct chunk retrieval below 60% of the time. Vector embeddings are a lossy compression of meaning — they capture semantic similarity but lose structural relationships, ordering, and precise instructions.

- **Context fragmentation**: Chunking documents for vector storage destroys the structural context that makes instructions coherent. A chunk that says "use the factory pattern" loses its significance without knowing it's in the "Service Initialization" section of the "Architecture" document.

- **Black box retrieval**: When vector search returns wrong results, debugging requires understanding embedding distances in high-dimensional space. There's no human-readable explanation for why a chunk was or wasn't retrieved.

- **Lossy translation**: Vector search finds content that is "similar in meaning but different in context," which reduces accuracy for instructional documentation where precision matters.

### The Alternative: Compiled Documentation with Keyword Search

The chosen architecture treats documentation like source code — it gets compiled into optimized artifacts. The compilation pipeline:

1. Parses markdown into structural components
2. Strips human-formatting overhead the AI doesn't need
3. Splits content into normalized, token-counted chunks
4. Builds a BM25 keyword search index over chunks
5. Extracts a taxonomy of terms with aliases for query translation
6. Generates manifests describing chunk relationships and navigation paths
7. Assembles always-loaded foundation documents into a pre-compiled context file

The AI interacts with the compiled output through the CLI, not by reading raw files.

### Design Principles

- **Source of truth is always the raw markdown**: Compiled artifacts are derived and fully reconstructable. Never edit compiled output directly.
- **Token counting is a first-class primitive**: Every size-related decision (chunk boundaries, context budgets, foundation assembly) is measured in tokens, not words or characters.
- **Deterministic over probabilistic**: Every retrieval decision is traceable. BM25 scores are explainable. Taxonomy mappings are inspectable. No black-box embedding distances.
- **Incremental compilation**: Content hashes track changes. Only modified documents trigger recompilation. The expensive LLM pass only processes new or changed terms. System instruction changes trigger targeted full regeneration of the affected artifact (taxonomy, bridges, or context) via instruction hash comparison.
- **Model-agnostic artifacts, model-aware presentation**: Compiled packages work with any AI model. Model-specific behavior is handled at the presentation layer through configuration.
- **Transparent AI instructions**: The instructions that govern how the compiler's AI behaves (taxonomy generation, bridge summaries, etc.) are themselves documentation files the user can read, understand, and modify. The compiler's behavior is documentation-driven.

---

## Phase 2: Package Structure

### Key Distinction: Project vs. Package

A **project** is the full documentation environment in a repository. It contains authored documentation, compiler configuration, AI instructions, installed dependency packages, and compiled output.

A **package** is a publishable subset — just curated documentation content. Packages contain only the four content directories (foundation, topics, plans, prompts) plus a codectx.yaml for identity and versioning. No compiler configuration, no AI instructions, no tooling state. When a consumer installs a package, their local compiler processes the package content using their local system/ instructions and compilation settings. This means package authoring has near-zero friction — you just write good markdown in the right directory structure.

**Publishable as a package:**
- `foundation/`
- `topics/`
- `plans/`
- `prompts/`
- `codectx.yaml` (identity and version only)

**Project-level only (never published):**
- `docs/codectx.lock` — resolved dependency versions
- `docs/system/` — compiler AI instructions
- `docs/.codectx/` — all tooling state, config, compiled output, installed packages

### Directory Layout

> **Note**: `docs/` is the default root directory. This is configurable via the `root` field in codectx.yaml. If `docs/` conflicts with existing content, any directory name can be used. All paths throughout this specification are relative to the configured root.

```
[project-root]/
  docs/                          # Default root — configurable via codectx.yaml root field
    system/
      foundation/
        compiler-philosophy/
          README.md            # General principles governing compilation behavior
          README.spec.md
      topics/
        taxonomy-generation/
          README.md            # Instructions the AI follows during alias generation
          README.spec.md       # Reasoning behind the default instructions
        bridge-summaries/
          README.md            # Instructions for generating chunk boundary bridges
          README.spec.md
        context-assembly/
          README.md            # Instructions for how foundations get assembled
          README.spec.md
      plans/                   # Empty on init — available for compiler migration plans
      prompts/                 # Default automation scripts for compiler-adjacent tasks
    foundation/
      [topic]/
        README.md              # Entry point for the topic
        [topic-breakdown].md   # Sub-topic breakdowns as needed
        README.spec.md         # Reasoning behind the documentation
        [topic-breakdown].spec.md
    topics/
      [topic]/
        README.md
        [topic-breakdown].md
        README.spec.md
        [topic-breakdown].spec.md
    plans/
      [topic]/
        README.md
        plan.yaml              # State tracking for resumable plans
        [topic-breakdown].md
        README.spec.md
        [topic-breakdown].spec.md
    prompts/
      [topic]/
        README.md
        [topic-breakdown].md
        README.spec.md
        [topic-breakdown].spec.md
    codectx.yaml               # Package identity, deps, session config
    codectx.lock               # Resolved dependency versions (auto-generated, checked in)
    .codectx/
      ai.yaml                  # AI model and behavior config (checked in)
      ai.local.yaml            # API keys, local endpoints (gitignored)
      preferences.yaml         # Compiler and pipeline config (checked in)
      packages/                # Installed dependency packages (gitignored)
        react-patterns@community:2.1.0/
        tailwind-guide@designteam:latest/
      compiled/                # All compiled output (gitignored)
        context.md             # Pre-assembled always-loaded foundations
        metadata.yaml          # Document relationship graph
        taxonomy.yaml          # Extracted term tree with aliases
        manifest.yaml          # Chunk navigation map with token counts
        hashes.yaml            # Content hashes for incremental compilation
        heuristics.yaml        # Compilation report — diagnostics, stats, timing
        objects/               # Instruction/documentation chunks
          [hash].[seq].md
        specs/                 # Reasoning/spec chunks (.spec.md sources)
          [hash].[seq].md
        system/                # System/compiler documentation chunks
          [hash].[seq].md
        bm25/
          objects/             # BM25 index over instruction chunks
          specs/               # BM25 index over reasoning chunks
          system/              # BM25 index over system/compiler chunks
```

### Directory Semantics

**system/**: Documentation about the compiler itself, using the same subdirectory layout as the standard package structure (foundation/, topics/, plans/, prompts/). Files are documentation in the same format as everything else — README.md as entry point, .spec.md for reasoning. They ship as sensible defaults on `codectx init` and belong to the user from that moment on. Modifying these files changes the compiler's AI behavior on the next compilation.

The system/ directory mirrors the standard layout:
- `system/foundation/`: General compiler philosophy and principles. The AI reads these before running any specific compilation pass, the same way project foundations frame everything in topics.
- `system/topics/`: Specific compiler operations — taxonomy generation, bridge summaries, context assembly. These contain the instructions the compiler's LLM passes follow.
- `system/plans/`: Empty on init. Available for compiler migration plans (e.g., "migrate taxonomy strategy from v1 to v2").
- `system/prompts/`: Default automation scripts for compiler-adjacent tasks — "audit taxonomy quality," "identify documentation gaps from heuristics.yaml," "refactor oversized chunks."

The system/ directory is never included in published packages. The consumer's compiler uses their own system/ instructions to process all content uniformly — both local docs and installed packages. There is one set of compilation rules per project, not per package.

**Compilation behavior**: System .md files (excluding .spec.md) are compiled into `compiled/system/` with the `sys:` prefix and indexed in their own BM25 index (`bm25/system/`). System .spec.md files are compiled into `compiled/specs/` with the `spec:` prefix alongside all other reasoning chunks. The compiler also reads raw system/topics/ markdown files as LLM instructions at compile time — the compiled system/ chunks exist for query-time searchability, while the raw files drive compile-time behavior. These are not conflicting roles; one is an input to the pipeline, the other is an output.

**Default system/ content scaffolded by `codectx init`:**

`system/topics/taxonomy-generation/README.md`:
```markdown
# Taxonomy Alias Generation Instructions

You are generating aliases for canonical terms extracted from documentation.
For each canonical term, generate alternative labels that a developer might
use when searching for the same concept.

## Rules

- Generate common abbreviations (e.g., "authentication" → "auth")
- Generate casual shorthand developers use verbally (e.g., "database" → "db")
- Generate formal alternatives (e.g., "auth" → "identity verification")
- Generate plurals and singular forms
- Generate related acronyms (e.g., "JSON Web Token" → "JWT")
- Do NOT generate antonyms or loosely related concepts
- Do NOT generate more than 10 aliases per term
- Prefer aliases that a developer would actually type into a search query

## Context

Each term is provided with:
- Its parent and child terms in the taxonomy hierarchy
- Example sentences from the source documentation
- The source type (heading, code identifier, or extracted phrase)

Use the hierarchy context to generate aliases that are appropriate to the
term's specificity level. A top-level term like "Authentication" should have
broader aliases than a leaf term like "JWT Refresh Token."
```

`system/topics/taxonomy-generation/README.spec.md`:
```markdown
# Taxonomy Generation Reasoning

The alias generation instructions prioritize search recall — making sure
developers find the right documentation regardless of which synonym they use.

The 10-alias limit prevents taxonomy bloat. Early testing showed that beyond
10 aliases per term, the additional aliases tend to be low-quality and can
cause false positive matches in BM25 search.

The prohibition on antonyms and loose associations prevents the taxonomy
from creating misleading connections. "Error handling" should not alias to
"success response" even though they're conceptually related — a developer
searching for error handling does not want success response documentation.
```

`system/topics/bridge-summaries/README.md`:
```markdown
# Bridge Summary Generation Instructions

You are generating one-line semantic bridge summaries for chunk boundaries.
Each bridge summarizes what the previous chunk established that the next
chunk assumes the reader already knows.

## Rules

- Keep each bridge to a single sentence, under 30 words
- Focus on what knowledge carries forward, not what was covered in detail
- Use specific terms from the content, not vague summaries
- Do NOT repeat the heading or title of the previous chunk
- Write in past tense ("Established...", "Defined...", "Covered...")

## Example

Good: "Defined the JWT token structure including header, payload, and
signature fields with RS256 signing requirements."

Bad: "The previous section was about JWT tokens." (too vague)
Bad: "JWT Token Structure: this section covered the structure." (repeats heading)
```

`system/topics/context-assembly/README.md`:
```markdown
# Context Assembly Instructions

You are assembling foundation documents into a single coherent engineering
context document that the AI reads at the start of every session.

## Rules

- Preserve the full content of each foundation document
- Add brief transition sentences between documents when topics shift abruptly
- Do NOT summarize or abbreviate the source content
- Do NOT add commentary or interpretation
- Maintain the order specified in codectx.yaml session.always_loaded
- Use consistent heading levels: H2 for each foundation document's title,
  H3 and below for internal structure
```

**Published package directory layout** (what gets published to the registry):

```
[package-name]/
  codectx.yaml             # Name, org, version, description, dependencies only
  foundation/              # Optional
    [topic]/
      README.md
      README.spec.md
  topics/                  # Optional
    [topic]/
      README.md
      [breakdown].md
      README.spec.md
  plans/                   # Optional
    [topic]/
      README.md
      plan.yaml
      README.spec.md
  prompts/                 # Optional
    [topic]/
      README.md
      README.spec.md
```

**foundation/**: Technology-agnostic guidance that survives a tech stack change. Engineering principles, architectural philosophy, coding standards, design conventions, general best practices. The portability test: if this guidance would still apply after switching frameworks, it belongs in foundation. When the AI is deciding between two valid implementation approaches, foundation docs establish which one aligns with this project's values. Foundation docs may also contain general guidance for a technology-specific package — like high-level guidelines that frame the topics within it.

**topics/**: Technology-specific documentation. How specific systems, components, frameworks, or libraries work within this project. Examples: `react/`, `tailwind/`, `nextjs/`, `postgres/`. Same structural conventions as foundation but scoped to specific technologies. This is where the AI goes to understand what exists and how to work with it — API contracts, data models, module documentation, implementation patterns.

**plans/**: Living documentation that tracks work in progress. Same structure as foundation and topics, but with a `plan.yaml` state file that enables resumable AI-driven development. As plans are executed, the AI updates the plan state so that if it loses context, stops mid-task, or continues on another machine, it can resume where it left off. Plans reference which documentation they depend on so the AI can reload the right context efficiently.

**prompts/**: Natural language scripts — pre-crafted instructions the AI should execute and follow. Same structural format as other directories. These codify tribal knowledge about how to instruct the AI effectively for specific task types within this project. Prompts with their .spec.md files become maintainable tools rather than magic incantations.

### Naming Conventions

- Topic directories use lowercase kebab-case: `error-handling/`, `jwt-auth/`, `database-migrations/`
- The directory name is the human-readable description of its purpose
- Every topic directory must contain a `README.md` as its entry point (mirrors GitHub's directory rendering convention)
- Sub-topic breakdowns use lowercase kebab-case `.md` files: `refresh-tokens.md`, `middleware-chain.md`
- Reasoning files mirror their parent with `.spec.md` suffix: `README.spec.md`, `refresh-tokens.spec.md`

### The .spec.md Convention

Every documentation file can have a corresponding `.spec.md` file that captures the reasoning behind the documentation.

**Purpose**: When the AI encounters a situation the documentation doesn't explicitly cover, the .spec.md files give it the *intent* behind the instructions, enabling reasoning by analogy rather than guessing.

**Example**: If `error-handling/README.md` says "always use structured error types," the corresponding `README.spec.md` explains *why* — perhaps because the team needs machine-parseable error output for their monitoring system, or because they plan to internationalize error messages later. When the AI encounters a new error scenario not covered by the docs, the spec tells it what the error handling strategy is *trying to achieve*, which guides correct extrapolation.

**For AI-authored documentation**: The .spec.md files are particularly valuable when AI generates or maintains documentation. They record the reasoning that produced each decision, creating an audit trail that both humans and AI can reference during maintenance and updates.

**Compilation behavior**: The compiler routes content to three separate output directories and BM25 indexes based on source type:

| Source | Compiled to | Prefix | BM25 Index |
|--------|-------------|--------|------------|
| `.md` files (non-system) | `compiled/objects/` | `obj:` | `bm25/objects/` |
| `.spec.md` files (all sources) | `compiled/specs/` | `spec:` | `bm25/specs/` |
| `system/**/*.md` files (non-spec) | `compiled/system/` | `sys:` | `bm25/system/` |

Each type gets its own BM25 index so they can be searched and scored independently. The manifest cross-references spec chunks to their parent instruction chunks via heading path alignment, so `codectx query` can surface relevant reasoning alongside instruction results without the types diluting each other's relevance scores. System chunks are searchable separately so project queries don't get polluted with compiler meta-documentation.

### codectx.yaml

This file serves two contexts with overlapping but distinct schemas: as a project manifest (full configuration) and as a published package manifest (identity and dependencies only).

**Project codectx.yaml** — the source of truth for package identity, dependencies, session context, and registry configuration:

```yaml
# Documentation root directory (default: "docs")
# Change if docs/ is already in use for other purposes
# All paths in this file and all CLI operations are relative to this root
root: "docs"

# Package identity
name: "my-project-docs"
org: "myteam"
version: "1.2.0"
description: "Engineering documentation for the my-project codebase"

# Always-loaded session context
# These are compiled into .codectx/compiled/context.md
# and pointed to by CLAUDE.md / AGENTS.md
# Order matters — documents appear in this order in the compiled context
# Supports local paths and package references
session:
  always_loaded:
    # Local foundation docs
    - foundation/coding-standards
    - foundation/architecture-principles
    - foundation/error-handling
    # Specific topic from an installed package
    - react-patterns@community/foundation/component-principles
    # Entire package as session preamble
    - company-standards@acme
  # Maximum token budget for always-loaded context
  # Compiler warns if assembled content exceeds this
  budget: 30000

# Package dependencies
dependencies:
  react-patterns@community:latest:
    active: true
  company-standards@acme:2.0.0:
    active: true
  tailwind-guide@designteam:2.1.0:
    active: true
  legacy-api-docs@internal:1.0.0:
    active: false  # Inactive — excluded from compiled output

# Package manager settings
# Packages are GitHub repos using the codectx-[name] naming convention
# e.g., react-patterns@community resolves to github.com/community/codectx-react-patterns
registry: "github.com"
```

**Published package codectx.yaml** — minimal identity and dependency declaration:

```yaml
# Package identity
name: "react-patterns"
org: "community"
version: "2.3.1"
description: "React component and hook patterns for AI-driven development"

# Other packages this package's documentation references
# Uses semver ranges — consumer's codectx.lock resolves exact versions
dependencies:
  javascript-fundamentals@community: ">=1.0.0"
```

Published packages do NOT include `session`, `active` flags on dependencies, or `registry`. The consumer controls all session context and active/inactive decisions. Published dependencies use semver ranges to declare compatibility; the consumer's `codectx.lock` pins exact versions.

**Fields by context:**

| Field | Project | Published | Purpose |
|-------|---------|-----------|---------|
| root | Project-only | — | Documentation root directory (default: "docs") |
| name, org, version, description | Required | Required | Package identity |
| session.always_loaded | Project-only | — | What the AI loads at session start |
| session.budget | Project-only | — | Token budget for session context |
| dependencies | With active/inactive flags | With semver ranges | Package relationships |
| registry | Project-only | — | Where to resolve packages |

**Managing session context via CLI:**

```bash
# Add entire package to session context
codectx session add company-standards@acme

# Add specific topic from a package
codectx session add react-patterns@community/foundation/component-principles

# Add local foundation doc
codectx session add foundation/error-handling

# Remove from session context
codectx session remove company-standards@acme

# List current session entries with token costs
codectx session list
```

`codectx session list` output:
```
Always-loaded session context (28,450 / 30,000 tokens):

  foundation/coding-standards                              8,200 tokens
  foundation/architecture-principles                       6,100 tokens
  foundation/error-handling                                5,000 tokens
  react-patterns@community/foundation/component-principles 4,800 tokens
  company-standards@acme                                   4,350 tokens
```

The CLI commands are convenience wrappers that modify the `session.always_loaded` list in codectx.yaml. The file remains the source of truth — hand-editing produces the same result. The commands provide token cost feedback that hand-editing wouldn't.

### ai.yaml

AI model and behavior configuration. Checked into version control (no secrets).

```yaml
# Model used during compilation for alias generation and bridge summaries
compilation:
  model: "claude-sonnet-4-20250514"
  encoding: "cl100k_base"  # Tokenizer encoding for token counting

# Target model for consumption (affects context budgets and formatting)
consumption:
  model: "claude-sonnet-4-20250514"
  context_window: 200000
  results_count: 10  # Default number of results returned by codectx query
                     # Can be overridden per-call with: codectx query --top N "..."

# Context formatting preference for generated output
output_format: "markdown"  # markdown | xml_tags | plain
```

**Note**: ai.yaml does not contain query translation overrides or alias mappings. All term aliasing is handled through the taxonomy, which is governed by the instructions in `docs/system/topics/taxonomy-generation/`. If the taxonomy isn't producing the right aliases, the fix is to improve those instructions and recompile — not to maintain override maps in config files. One mechanism, one place, one pattern.

### ai.local.yaml (always gitignored)

```yaml
# Local-only overrides — API keys, endpoints, personal preferences
api_key: "sk-..."
endpoint: "https://api.anthropic.com"

# Override compilation model locally
compilation:
  model: "claude-sonnet-4-20250514"
```

### preferences.yaml

Compiler and pipeline configuration. Checked into version control.

```yaml
# Chunk compilation settings
chunking:
  target_tokens: 450        # Target chunk size in tokens
  min_tokens: 200            # Minimum chunk size (avoid tiny fragments)
  max_tokens: 800            # Maximum chunk size (hard ceiling)
  flexibility_window: 0.8    # Break after 80% of target if next block would exceed

# BM25 index configuration
bm25:
  k1: 1.2                   # Term frequency saturation
  b: 0.75                    # Document length normalization

# Taxonomy extraction settings
taxonomy:
  min_term_frequency: 2      # Minimum corpus-wide frequency to include a term
  max_alias_count: 10        # Maximum aliases per canonical term
  pos_extraction: true       # Enable POS-based term extraction
  llm_alias_generation: true # Enable LLM pass for alias generation

# Documentation linting / validation
validation:
  require_readme: true       # Every topic directory must have README.md
  require_spec: false        # Spec files are recommended, not required
  max_file_tokens: 10000     # Warn if a single source file exceeds this
  require_headings: true     # Warn if a file has no heading structure
```

### .gitignore additions

```gitignore
# codectx — tooling state and compiled output
docs/.codectx/compiled/
docs/.codectx/packages/
docs/.codectx/ai.local.yaml
```

Force-include checked-in config:
```gitignore
!docs/.codectx/ai.yaml
!docs/.codectx/preferences.yaml
```

---

## Phase 3: Markdown Compilation Pipeline

### Overview

The compilation pipeline transforms raw markdown files into the set of artifacts the AI consumes at runtime. It runs as `codectx compile` and supports both full and incremental compilation via content hash tracking.

### Input Sources

The compiler processes markdown from three locations:
1. `docs/` — local project documentation (foundation, topics, plans, prompts)
2. `docs/system/` — compiler documentation (foundation, topics, plans, prompts for the tooling itself)
3. `.codectx/packages/` — installed dependency packages (active ones only per codectx.yaml)

All three sources are processed through the same pipeline but routed to different output directories. Non-system .md files go to `compiled/objects/`, system .md files go to `compiled/system/`, and all .spec.md files go to `compiled/specs/`. Additionally, the compiler reads raw system/topics/ markdown files as LLM instructions during compile-time passes — compilation and indexing are not conflicting roles.

### Pipeline Stages

```
Stage 1: Parse & Validate
  Source markdown → AST → structural validation

Stage 2: Strip & Normalize
  Remove human-formatting overhead → clean content

Stage 3: Chunk
  Normalized content → token-counted semantic blocks → chunks with context headers

Stage 4: Index
  Chunks → BM25 inverted index

Stage 5: Extract Taxonomy
  Structural terms + POS extraction → candidate terms → deduplicated taxonomy tree

Stage 6: LLM Augmentation
  Taxonomy tree → LLM alias generation (using docs/system/topics/taxonomy-generation/ instructions)
  Chunk boundaries → LLM bridge summaries (using docs/system/topics/bridge-summaries/ instructions)

Stage 7: Generate Manifests
  Chunks + taxonomy + relationships → manifest.yaml + metadata.yaml + hashes.yaml

Stage 8: Assemble Context
  Always-loaded foundations → context.md (using docs/system/topics/context-assembly/ instructions)

Stage 9: Sync Entry Points
  context.md → CLAUDE.md / AGENTS.md / .cursorrules

Stage 10: Generate Heuristics
  Compilation stats from all stages → heuristics.yaml
```

### Stage 1: Parse & Validate

**Input**: All `.md` files under `docs/foundation/`, `docs/topics/`, `docs/plans/`, `docs/prompts/`, `docs/system/`, and active packages in `.codectx/packages/`.

**Process**:
- Parse each markdown file into an AST (abstract syntax tree)
- Validate structural requirements from `preferences.yaml`:
  - Every topic directory has a `README.md` if `require_readme` is true
  - Every file has at least one heading if `require_headings` is true
  - No file exceeds `max_file_tokens`
- Compute content hash (SHA-256) for each file
- Compare against cached hashes from `.codectx/compiled/hashes.yaml`
- Mark changed files for reprocessing; skip unchanged files in incremental mode

**Output**: Parsed ASTs with change flags. Validation warnings/errors.

**Thought process**: Content hashing enables incremental compilation. A project with 1,000 markdown files where 3 changed only needs to reprocess those 3 files through the expensive stages (POS extraction, chunking). The hash comparison is O(n) on file count but the actual work scales with the number of changes.

### Stage 2: Strip & Normalize

**Input**: Parsed ASTs from Stage 1.

**Process**:
- Remove formatting that exists only for human rendering and provides no semantic value to the AI:
  - Excessive blank lines (normalize to single blank line between blocks)
  - Decorative horizontal rules
  - HTML comments
  - Redundant emphasis/bold that doesn't carry semantic weight
- Preserve all structural elements:
  - Headings (these drive chunk boundaries and taxonomy extraction)
  - Code blocks (indivisible atomic units)
  - Lists (kept whole as atomic units)
  - Tables (kept whole as atomic units)
  - Links and cross-references (drive metadata.yaml relationship graph)
- Normalize heading levels (if a file starts at H3, normalize to H1 for consistent hierarchy)

**Output**: Cleaned ASTs ready for chunking.

**Thought process**: The AI doesn't need three blank lines between paragraphs or decorative dividers. Every unnecessary token in the compiled output is a token the AI can't use for actual task context. This stage typically reduces content by 10-20% in token count with zero information loss.

### Stage 3: Chunk

**Input**: Cleaned ASTs from Stage 2.

**Process**: Semantic block accumulation with a target token window.

**Token counting**: Use `tiktoken-go/tokenizer` package with the encoding specified in `ai.yaml` (default: `cl100k_base`). Token counting is the unit of measurement for all size decisions.

**Go package**: `github.com/tiktoken-go/tokenizer`
- Pure Go implementation, no external dependencies
- Embeds OpenAI vocabulary data (~4MB added to binary)
- Supports cl100k_base (GPT-4), p50k_base, r50k_base, o200k_base
- Interface: `Count(string) (int, error)` for fast token counting

```
Algorithm: Semantic Block Accumulation

1. Parse the cleaned AST into an ordered list of SEMANTIC BLOCKS.
   A semantic block is the smallest meaningful unit:
   - A paragraph
   - A code block (never split)
   - A list (kept whole)
   - A table (kept whole)
   - A blockquote

2. Each block carries metadata:
   - The most recent heading hierarchy above it
     (e.g., "Authentication > JWT > Refresh Flow")
   - Its position in the source file
   - Its token count

3. Walk the block list, accumulating blocks into a CHUNK:
   - Track running token count
   - When adding the next block would exceed target_tokens:
     a. If next block is a HEADING → always break before it
        (headings should start new chunks when possible)
     b. If current chunk is >= flexibility_window (80%) of target →
        break here, start new chunk
     c. If current chunk is < flexibility_window of target →
        include the next block even if it pushes slightly over target
   - Never split within an atomic block (code block, table, list)
   - If a single atomic block exceeds max_tokens on its own →
     it becomes its own oversized chunk, flagged in manifest

4. Generate CONTEXT HEADER for each chunk:
   - Source file path
   - Heading hierarchy at chunk start position
   - Chunk sequence: "chunk N of M" for this source file
   - Token count of the chunk content (excluding header)

5. Generate CHUNK ID:
   - Hash the chunk CONTENT (without context header)
   - ID format: [content-hash].[sequence-index]
   - Content-only hashing means header format changes don't invalidate cache
   - Prefix with type:
     - obj:[hash].[seq] for instruction chunks
     - spec:[hash].[seq] for reasoning chunks
     - sys:[hash].[seq] for system/compiler documentation chunks

6. Route chunk to the appropriate output directory:
   - Chunks from .md files (non-system) → .codectx/compiled/objects/[hash].[seq].md
   - Chunks from .spec.md files (all sources) → .codectx/compiled/specs/[hash].[seq].md
   - Chunks from system/**/*.md (non-spec) → .codectx/compiled/system/[hash].[seq].md
```

**Chunk file format**:
```markdown
<!-- codectx:meta
id: obj:a1b2c3.03
type: object
source: docs/topics/authentication/jwt-tokens.md
heading: Authentication > JWT Tokens > Refresh Flow
chunk: 3 of 7
tokens: 462
-->

The refresh token lifecycle begins when the user authenticates...
[actual chunk content continues]
```

**Spec chunk file format**:
```markdown
<!-- codectx:meta
id: spec:f7g8h9.02
type: spec
source: docs/topics/authentication/jwt-tokens.spec.md
heading: Authentication > JWT Tokens > Refresh Flow
chunk: 2 of 3
tokens: 380
parent_object: obj:a1b2c3.03
-->

The refresh token lifecycle was designed around the constraint that...
[reasoning content continues]
```

The `parent_object` field in spec chunks links reasoning to its corresponding instruction chunk based on heading path alignment. This cross-reference is how `codectx query` surfaces relevant reasoning alongside instruction results.

**System chunk file format**:
```markdown
<!-- codectx:meta
id: sys:m3n4o5.01
type: system
source: system/topics/taxonomy-generation/README.md
heading: Taxonomy Alias Generation Instructions > Rules
chunk: 1 of 2
tokens: 340
-->

Generate common abbreviations (e.g., "authentication" → "auth")...
[system instruction content continues]
```

System chunks are searchable at query time so the AI or developer can ask "how does the compiler generate aliases" and get results. They are also read as raw instructions by the compiler during LLM passes — these two roles don't conflict.

**Handling oversized atomic blocks**: A code block that's 2,000 tokens can't be split. It becomes a single oversized chunk, flagged in the manifest so the AI knows it exists and can budget accordingly. These are exceptions, not the norm — the overall scoring consistency is preserved.

**Why this approach over heading-based chunking**: Someone can write a 5,000-word section under a single H2. Heading-based chunking produces one massive chunk that BM25 penalizes through length normalization even when it's highly relevant. Token-based accumulation with semantic block boundaries produces consistent chunk sizes across the entire corpus. When all chunks are approximately the same size, BM25's length normalization parameter `b` becomes nearly irrelevant — scoring becomes purely about term frequency and inverse document frequency. Every chunk competes on equal footing.

**Output**: Instruction chunks in `.codectx/compiled/objects/`, reasoning chunks in `.codectx/compiled/specs/`, and system chunks in `.codectx/compiled/system/`, each with context headers, type markers, and token counts.

### Stage 4: Index (BM25)

**Input**: All chunk files from Stage 3 (objects/, specs/, and system/).

**Process**:
- Build three separate BM25 indexes:
  - **Objects index**: Tokenize and index all instruction chunks from objects/
  - **Specs index**: Tokenize and index all reasoning chunks from specs/
  - **System index**: Tokenize and index all system/compiler chunks from system/
- Each index has its own IDF calculations scoped to its corpus, so term rarity is meaningful within each type
- Apply BM25 scoring parameters from `preferences.yaml` (k1, b) to all indexes
- Serialize to `.codectx/compiled/bm25/objects/`, `.codectx/compiled/bm25/specs/`, and `.codectx/compiled/bm25/system/`

**Why separate indexes**: Each content type uses different language patterns. Instructions use directive terms ("use," "implement," "always"). Reasoning uses explanatory terms ("because," "the goal is," "we chose"). System docs use meta/tooling terms ("compiler," "generate," "alias," "chunk"). Separate indexes mean each type scores within its own corpus — a system chunk about "how to generate authentication aliases" doesn't compete with an instruction chunk about "how to implement authentication."

**Go packages for BM25** (evaluated during design):

| Package | Strengths | Best For |
|---------|-----------|----------|
| `github.com/crawlab-team/bm25` | Full BM25 variant support (Okapi, L, Plus, Adpt, T). Parallel/batched scoring via goroutines. Direct port of Python rank_bm25. | Most complete option. Recommended starting point. |
| `github.com/covrom/bm25s` | Optimized for short text documents. Auto-adjusts parameters by document length. | If indexing only chunk metadata rather than full content. |
| `github.com/raphaelsty/gokapi` | Disk-backed via Diskv. Searches large document sets without memory constraints. | If index exceeds available memory at scale. |
| `codeberg.org/leejuyuu/bm25s-go` | Pure Go port of bm25s. Precalculated sparse scoring matrices. | Maximum query-time performance. |

**Recommendation**: Start with `crawlab-team/bm25` for completeness. The parallel scoring via goroutines aligns well with Go's concurrency model. Custom tokenizer function lets you integrate domain-aware tokenization.

**Custom tokenizer consideration**: The BM25 tokenizer should be domain-aware:
- Preserve compound terms: "error-handling" stays whole, not split on hyphen
- Preserve code identifiers: `CreateUser` stays whole
- Lowercase for matching but preserve original case in the index
- Remove standard English stopwords but preserve technical ones ("null", "void", "async")

```go
// Example: Domain-aware tokenizer for BM25 indexing
// Passed to crawlab-team/bm25 as the tokenizer function

var technicalStopwords = map[string]bool{
    "null": true, "void": true, "async": true, "await": true,
    "true": true, "false": true, "nil": true, "err": true,
}

var standardStopwords = map[string]bool{
    "the": true, "a": true, "an": true, "is": true, "are": true,
    "was": true, "were": true, "be": true, "been": true, "being": true,
    "have": true, "has": true, "had": true, "do": true, "does": true,
    "did": true, "will": true, "would": true, "could": true, "should": true,
    "may": true, "might": true, "shall": true, "can": true,
    "to": true, "of": true, "in": true, "for": true, "on": true,
    "with": true, "at": true, "by": true, "from": true, "as": true,
    "into": true, "through": true, "during": true, "before": true, "after": true,
    "this": true, "that": true, "these": true, "those": true,
    "it": true, "its": true, "and": true, "or": true, "but": true, "not": true,
}

func tokenize(text string) []string {
    // Split on whitespace and punctuation, but preserve:
    // - Hyphenated compounds: "error-handling" → "error-handling"
    // - Code identifiers: "CreateUser" → "createuser"
    // - Dotted paths: "http.Handler" → "http.handler"
    words := regexp.MustCompile(`[\w][\w\-\.]*[\w]|[\w]+`).FindAllString(text, -1)

    var tokens []string
    for _, word := range words {
        lower := strings.ToLower(word)
        // Skip standard stopwords but keep technical ones
        if standardStopwords[lower] && !technicalStopwords[lower] {
            continue
        }
        tokens = append(tokens, lower)
    }
    return tokens
}

// Usage with crawlab-team/bm25:
// bm25.NewBM25Okapi(corpus, tokenize, 1.2, 0.75, nil)
```

**What gets indexed**: The chunk content text, minus the context header metadata. The context header is for AI navigation, not for search scoring. Both instruction and reasoning content feed into the shared taxonomy (Stage 5) since canonical vocabulary is shared across both types.

**Output**: Three serialized BM25 indexes in `.codectx/compiled/bm25/objects/`, `.codectx/compiled/bm25/specs/`, and `.codectx/compiled/bm25/system/`.

### Stage 5: Extract Taxonomy

**Input**: All cleaned ASTs from Stage 2.

**Process**: Multi-pass extraction pipeline, similar to how a compiler builds a symbol table.

**Pass 1 — Structural term extraction (pure parsing, no NLP)**:
- Extract headings at all levels → highest-confidence canonical terms
- Extract code identifiers from code blocks (function names, type names, package names)
- Extract bold/emphasized terms in definition-like patterns
- Extract terms from structured positions (list headers, table headers)
- Record which chunks each term appears in

**Pass 2 — Relationship inference (structural analysis)**:
- Heading hierarchy directly yields parent-child relationships:
  - H2 "OAuth" under H1 "Authentication" → OAuth is a child of Authentication
- Cross-references between documents (markdown links) → lateral relationships
- Import statements and dependency references in code blocks → technical dependency relationships
- All deterministic, no NLP required

**Pass 3 — POS-based term extraction (NLP)**:

**Go package**: `github.com/jdkato/prose/v2`
- Pure Go, no external dependencies
- Tokenization, segmentation, POS tagging, named-entity extraction
- English language support
- POS tags let you specifically extract noun phrases:
  - Filter for patterns: adjective+noun, noun+noun
  - Captures compound technical terms: "dependency injection", "middleware chain"
  - Named-entity extraction catches library names, framework names, service names

```go
// Example: Extracting noun phrases with prose
doc, _ := prose.NewDocument(chunkText)
for _, tok := range doc.Tokens() {
    // tok.Tag gives POS tag (NN, NNP, JJ, etc.)
    // tok.Text gives the word
    // Filter for noun phrases: sequences of JJ* + NN+
}
for _, ent := range doc.Entities() {
    // ent.Text gives named entities
    // ent.Label gives entity type (GPE, ORG, PERSON)
}
```

**Scalability**: POS tagging a 500-word chunk takes single-digit milliseconds. At 1M chunks: 1-3 hours single-threaded, or minutes with goroutine parallelism. Go excels at this kind of embarrassingly parallel workload.

**Pass 4 — Deduplication and normalization**:
- Merge terms extracted from different passes:
  - If POS extraction found "error handling" in body text and "Error Handling" already exists as a heading-derived term → merge under heading form as canonical
  - If "DB migrations" appears in body text and "Database Migrations" is a heading → merge under heading form
- Score terms by corpus-wide frequency:
  - A term in 500 chunks is more taxonomy-worthy than one in 2 chunks
  - Apply `min_term_frequency` threshold from `preferences.yaml`
- This is where 1M chunks' worth of raw extractions collapse into a manageable term set (typically 5,000-50,000 unique terms)

**Output**: Raw taxonomy tree in intermediate format, persisted as part of taxonomy.yaml (serves as its own cache for incremental builds).

### Stage 6: LLM Augmentation

**Input**: Deduplicated taxonomy tree from Stage 5. Chunk boundary pairs from Stage 3. Instructions from `docs/system/`.

**Process**: Two LLM tasks run during compilation. Both use the instructions in `docs/system/` to control the AI's behavior. This means the user can read, understand, and modify how these passes work by editing documentation files — the compiler's AI behavior is transparent and documentation-driven.

**Task 1 — Alias generation**:
- Read instructions from `docs/system/topics/taxonomy-generation/README.md`
- For each canonical term in the taxonomy, generate likely aliases and variations
- The LLM receives the term, its context (parent/child terms, example sentences), and the taxonomy-generation instructions
- Batch efficiently: send 50-100 terms per API call, grouped by taxonomy branch
  - Branch-level batching gives the LLM better context for relevant aliases

**Critical scalability insight**: The LLM pass operates on the DEDUPLICATED TAXONOMY, not on raw chunks. At 1M chunks:
- Raw candidate term instances: 10-30M
- After deduplication: 5,000-50,000 unique terms
- Batched at 50-100 per call: 50-1,000 API calls
- This is cheap, fast, and completely feasible

**Task 2 — Boundary bridge summaries**:
- Read instructions from `docs/system/topics/bridge-summaries/README.md`
- For each pair of adjacent chunks from the same source file, generate a one-line semantic bridge
- The bridge summarizes what the previous chunk established that the next chunk assumes
- Example: "Previous chunk defined the JWT refresh token lifecycle and validation rules"
- Costs ~15-20 tokens per boundary vs. 50-100 for content overlap
- Batched similarly to alias generation

**Caching**: Both outputs are stored in taxonomy.yaml (aliases) and manifest.yaml (bridges). On incremental builds, only new or changed terms/boundaries are reprocessed. The compiled artifacts serve as their own cache.

**Graceful degradation**: The LLM augmentation pass is optional. If it fails (API unavailable, rate limited, misconfigured), or if `llm_alias_generation` is set to false in preferences.yaml, the compilation pipeline completes successfully. The taxonomy will contain only structurally-extracted and POS-extracted terms without LLM-generated aliases — reduced alias coverage but still functional. Bridge summaries will be empty, and the manifest's `bridge_to_next` fields will be null — the AI loses continuity hints but adjacency references still work for navigation. This ensures the compilation never fails due to an external API dependency.

**Why no separate cache/ directory**: The taxonomy.yaml *is* the persisted result of alias generation. The manifest.yaml *is* the persisted result of bridge generation. If a canonical term hasn't changed (its source chunks haven't changed), the existing aliases are retained and the LLM call is skipped. No intermediate cache artifacts needed.

**Output format**: The taxonomy follows a SKOS-inspired schema (W3C Simple Knowledge Organization System). SKOS provides a proven data model for controlled vocabularies with preferred labels, alternative labels, and broader/narrower relationships.

**Reference**: W3C SKOS Primer — https://www.w3.org/2004/02/skos/

```yaml
# .codectx/compiled/taxonomy.yaml
encoding: "cl100k_base"
compiled_with: "claude-sonnet-4-20250514"
compiled_at: "2025-03-09T12:00:00Z"
instructions_hash: "abc123..."  # Hash of system/topics/taxonomy-generation/ for cache invalidation
term_count: 12847

terms:
  authentication:
    canonical: "Authentication"
    aliases:
      - "auth"
      - "login"
      - "sign-in"
      - "sign in"
      - "authn"
      - "identity verification"
      - "credentials"
    broader: null  # top-level term
    narrower:
      - "oauth"
      - "jwt"
      - "session-auth"
      - "api-keys"
    related:
      - "authorization"
      - "middleware"
    chunks:
      - "a1b2c3.01"
      - "a1b2c3.02"
      - "d4e5f6.04"
    source: "heading"  # how this term was discovered

  jwt:
    canonical: "JWT"
    aliases:
      - "JSON Web Token"
      - "json web token"
      - "jwt token"
      - "bearer token"
    broader: "authentication"
    narrower:
      - "refresh-token"
      - "access-token"
    related:
      - "token-validation"
      - "token-expiry"
    chunks:
      - "a1b2c3.03"
      - "a1b2c3.04"
    source: "heading"
```

**Instruction hashing pattern**: Every compiled artifact that's produced by an LLM pass records the hash of the system/ instructions that governed its generation. This enables cache invalidation when instructions change:

| Artifact | Instructions source | Hash field |
|----------|-------------------|------------|
| taxonomy.yaml | `system/topics/taxonomy-generation/` | `instructions_hash` |
| manifest.yaml | `system/topics/bridge-summaries/` | `bridge_instructions_hash` |
| context.md | `system/topics/context-assembly/` | In metadata header |

During incremental compilation, the compiler compares each artifact's stored instruction hash against the current hash of the corresponding system/ directory. If different, that artifact is fully regenerated regardless of whether source content changed. This ensures that editing system instructions always takes effect on the next compile — without it, the compiler would see unchanged source terms/chunks and skip the LLM pass, leaving stale output.

### Stage 7: Generate Manifests

**Input**: Chunks, taxonomy, cross-references from all previous stages.

**Process**: Generate three files in `.codectx/compiled/`.

**manifest.yaml** — Chunk navigation map:
```yaml
# .codectx/compiled/manifest.yaml
total_chunks: 4850
total_object_chunks: 3842
total_spec_chunks: 879
total_system_chunks: 129
total_tokens: 2134500
encoding: "cl100k_base"
bridge_instructions_hash: "sha256:q7r8s9..."  # Hash of system/topics/bridge-summaries/ — change triggers full bridge regeneration

objects:
  obj:a1b2c3.01:
    type: "object"
    source: "docs/topics/authentication/jwt-tokens.md"
    heading: "Authentication > JWT Tokens"
    sequence: 1
    total_in_file: 7
    tokens: 462
    terms: ["authentication", "jwt", "bearer-token"]
    adjacent:
      previous: null
      next: "obj:a1b2c3.02"
    bridge_to_next: "Establishes JWT token structure and signing algorithm requirements"
    spec_chunk: "spec:f7g8h9.01"  # Corresponding reasoning chunk, if exists

  obj:a1b2c3.02:
    type: "object"
    source: "docs/topics/authentication/jwt-tokens.md"
    heading: "Authentication > JWT Tokens > Validation"
    sequence: 2
    total_in_file: 7
    tokens: 488
    terms: ["jwt", "token-validation", "signature-verification"]
    adjacent:
      previous: "obj:a1b2c3.01"
      next: "obj:a1b2c3.03"
    bridge_to_next: "Covered validation rules and signature verification; next section addresses token refresh lifecycle"
    spec_chunk: null  # No corresponding spec file for this section

specs:
  spec:f7g8h9.01:
    type: "spec"
    source: "docs/topics/authentication/jwt-tokens.spec.md"
    heading: "Authentication > JWT Tokens"
    sequence: 1
    total_in_file: 3
    tokens: 380
    terms: ["authentication", "jwt", "design-decision"]
    parent_object: "obj:a1b2c3.01"

system:
  sys:m3n4o5.01:
    type: "system"
    source: "system/topics/taxonomy-generation/README.md"
    heading: "Taxonomy Alias Generation Instructions > Rules"
    sequence: 1
    total_in_file: 2
    tokens: 340
    terms: ["taxonomy", "alias", "generation", "keyword-extraction"]
    adjacent:
      previous: null
      next: "sys:m3n4o5.02"
    bridge_to_next: "Defined alias generation rules including abbreviations, shorthand, and formal alternatives"
```

**metadata.yaml** — Document relationship graph:
```yaml
# .codectx/compiled/metadata.yaml
compiled_at: "2025-03-09T12:00:00Z"

documents:
  docs/topics/authentication/jwt-tokens.md:
    type: "topic"
    title: "JWT Token Management"
    chunks: ["a1b2c3.01", "a1b2c3.02", "a1b2c3.03"]
    total_tokens: 3420
    references_to:
      - path: "docs/foundation/error-handling/README.md"
        reason: "Cross-reference to error handling patterns for token validation failures"
      - path: "docs/topics/authentication/oauth/README.md"
        reason: "OAuth flow depends on JWT token issuance"
    referenced_by:
      - path: "docs/topics/user-service/README.md"
        reason: "User service consumes JWT tokens for authentication"
      - path: "docs/plans/auth-migration/README.md"
        reason: "Auth migration plan references current JWT implementation"
```

**hashes.yaml** — Content hashes for incremental compilation:
```yaml
# .codectx/compiled/hashes.yaml
compiled_at: "2025-03-09T12:00:00Z"

files:
  docs/foundation/coding-standards/README.md: "sha256:a1b2c3d4..."
  docs/foundation/coding-standards/README.spec.md: "sha256:e5f6g7h8..."
  docs/topics/authentication/jwt-tokens.md: "sha256:i9j0k1l2..."

# System instruction hashes — changes trigger full re-run of affected LLM pass
# These are the authoritative source; the per-artifact hashes (instructions_hash in
# taxonomy.yaml, bridge_instructions_hash in manifest.yaml, assembly hash in context.md)
# are compared against these during incremental compilation
system:
  taxonomy-generation: "sha256:m3n4o5p6..."
  bridge-summaries: "sha256:q7r8s9t0..."
  context-assembly: "sha256:u1v2w3x4..."
```

### Stage 8: Assemble Context

**Input**: Always-loaded entries declared in `codectx.yaml` under `session.always_loaded`. These may be local foundation paths, specific topics from packages, or entire packages. Assembly instructions from `docs/system/topics/context-assembly/`.

**Process**:
- Resolve each always-loaded reference to its source markdown files:
  - Local paths (e.g., `foundation/coding-standards`) → resolve under `docs/`
  - Package paths (e.g., `react-patterns@community/foundation/component-principles`) → resolve under `.codectx/packages/`
  - Bare package references (e.g., `company-standards@acme`) → resolve all docs from that package
- Process through Stages 2-3 (strip and normalize) but do NOT chunk — keep as flowing prose
- Assemble into a single coherent document in the order declared in `codectx.yaml`
- The assembly may use the LLM pass (governed by `docs/system/topics/context-assembly/` instructions) to smooth transitions between sections
- Tokenize the assembled output and compare against the declared budget
- If over budget: emit warning identifying which entries are the largest consumers
- Write to `.codectx/compiled/context.md`

**context.md format**: This is the one document in the system optimized for reading, not searching. Not chunked, not fragmented, not indexed. The AI reads it top to bottom at session start.

```markdown
# Project Engineering Context

> This document is automatically compiled from session context entries.
> Source: docs/codectx.yaml session.always_loaded
> Assembly instructions: docs/system/topics/context-assembly/ (sha256:u1v2w3...)
> Token count: 28,450 / 30,000 budget
> Compiled: 2025-03-09T12:00:00Z

## Coding Standards

[Full content from foundation/coding-standards/README.md, stripped and normalized]

## Architecture Principles

[Full content from foundation/architecture-principles/README.md]

## Error Handling Philosophy

[Full content from foundation/error-handling/README.md]

## React Component Principles

[Full content from react-patterns@community/foundation/component-principles/README.md]

## Company Engineering Standards

[Full content from company-standards@acme, all docs assembled]
```

### Stage 9: Sync Entry Points

**Input**: Compiled `context.md`. Configuration from `ai.yaml`.

**Process**: Generate or update AI tool entry point files at the repo root.

**CLAUDE.md** (for Claude Code):
```markdown
# Project Instructions

Read the compiled engineering context before any task:
- Context file: docs/.codectx/compiled/context.md

Follow the instructions in that document with absolute priority.
They override any assumptions from your training data for this project.

## Documentation Queries

For any development task, search documentation before writing code:
- Query: `codectx query "your search terms"`
- Generate: `codectx generate "obj:chunk-id,obj:chunk-id,spec:chunk-id,sys:chunk-id"`

Query returns ranked instruction, reasoning, and system results.
Generate assembles selected chunks into a single readable document.
Read the returned files for task-specific guidance.

Never assume knowledge that isn't in the documentation.
```

**Design rationale**: CLAUDE.md is deliberately minimal. Its only job is to bootstrap the AI into the codectx system. The actual instructions live in `context.md`. This maintains single-source-of-truth — foundation doc updates only require recompilation, not manual CLAUDE.md editing.

### Stage 10: Generate Heuristics

**Input**: Statistics collected from all previous stages.

**Process**: Assemble a diagnostic report of the entire compilation into `.codectx/compiled/heuristics.yaml`. Regenerated on every compile — it's a snapshot of the current state, not cumulative history.

```yaml
# .codectx/compiled/heuristics.yaml
compiled_at: "2025-03-09T12:00:00Z"
compiler_version: "0.1.0"
encoding: "cl100k_base"

# Compilation input summary
sources:
  total_files: 342
  local_files: 298
  package_files: 44
  new: 3
  modified: 7
  unchanged: 332
  spec_files: 86

# Chunk output summary
chunks:
  total: 4850
  objects: 3842
  specs: 879
  system: 129
  total_tokens: 2134500
  average_tokens: 440
  min_tokens: 203
  max_tokens: 1847
  oversized: 4  # Chunks exceeding max_tokens due to indivisible blocks

# Taxonomy summary
taxonomy:
  canonical_terms: 12847
  total_aliases: 48203
  average_aliases_per_term: 3.75
  terms_from_headings: 4210
  terms_from_code_identifiers: 1893
  terms_from_pos_extraction: 6744
  aliases_from_llm: 42650

# Session context summary
session:
  total_tokens: 28450
  budget: 30000
  utilization: "94.8%"
  entries:
    - path: "foundation/coding-standards"
      tokens: 8200
    - path: "foundation/architecture-principles"
      tokens: 6100
    - path: "foundation/error-handling"
      tokens: 5000
    - path: "react-patterns@community/foundation/component-principles"
      tokens: 4800
    - path: "company-standards@acme"
      tokens: 4350

# BM25 index summary
bm25:
  objects:
    indexed_terms: 28493
    indexed_chunks: 3842
    average_chunk_length: 452
  specs:
    indexed_terms: 12847
    indexed_chunks: 879
    average_chunk_length: 433
  system:
    indexed_terms: 3210
    indexed_chunks: 129
    average_chunk_length: 428

# Incremental compilation details
incremental:
  full_recompile: false
  stages_skipped: ["taxonomy_extraction"]
  stages_rerun: ["chunking", "bm25_indexing", "manifest_generation"]
  system_instructions_changed:
    taxonomy_generation: false
    bridge_summaries: false
    context_assembly: false

# Timing
timing:
  total_seconds: 47.3
  parse_validate: 2.1
  strip_normalize: 1.8
  chunking: 8.4
  bm25_indexing: 3.2
  taxonomy_extraction: 0.0  # Skipped (incremental)
  llm_augmentation: 22.1
  manifest_generation: 4.5
  context_assembly: 3.8
  sync_entry_points: 1.4
```

**Purpose**: Gives both humans and AI a complete picture of the compiled state. The AI can read heuristics.yaml to orient itself — "this project has 4,850 chunks across 342 files with 12,847 taxonomy terms, session context is at 95% of budget" — before it begins querying. The timing breakdown helps identify pipeline bottlenecks. The incremental details show what was skipped and why. The session summary surfaces budget pressure before it becomes a problem.

---

## Phase 4: BM25 Search Integration

### How BM25 Works

BM25 (Best Matching 25) is a ranking algorithm that scores documents against a search query based on three factors:

1. **Term Frequency (TF)**: How often the search term appears in a document. More occurrences = higher score, but with diminishing returns (controlled by parameter `k1`). Prevents keyword stuffing from gaming rankings.

2. **Inverse Document Frequency (IDF)**: How rare the search term is across the entire corpus. Rare terms carry more weight. If "authentication" appears in 3 of 50 chunks, it's a strong signal. If "the" appears in all 50, it's ignored.

3. **Document Length Normalization**: Shorter, more focused documents get a slight boost over longer ones (controlled by parameter `b`). A short chunk that mentions the search term is more likely focused on that term than a long chunk that mentions it once in passing.

**Why BM25 over vector search for this use case**:
- Deterministic and explainable — you can trace exactly why a chunk ranked where it did
- Development documentation uses consistent terminology, so exact keyword matching is the right default
- No embedding model dependency, no vector database, no GPU requirements
- Scoring is fast: sub-millisecond per query at corpus sizes typical of project documentation
- With normalized chunk sizes, length normalization becomes nearly irrelevant, making scoring purely about term relevance

**BM25's limitation**: Cannot match semantically similar but lexicographically different terms. "Login flow" won't match documentation titled "Authentication Sequence." The taxonomy addresses this — query-time term translation bridges the gap deterministically.

### The Scoring Formula (Conceptual)

```
score(term, document) = IDF(term) * (TF(term, document) * (k1 + 1)) /
                        (TF(term, document) + k1 * (1 - b + b * |document| / avgdl))
```

Where:
- **IDF(term)**: log((N - n(term) + 0.5) / (n(term) + 0.5)) — how rare the term is
- **TF(term, document)**: how often the term appears in this document
- **k1**: term frequency saturation (default 1.2)
- **b**: document length normalization (default 0.75)
- **|document|**: document length in tokens
- **avgdl**: average document length across the corpus

### Why Normalized Chunk Sizes Help BM25

When all chunks are approximately the same length, `|document| / avgdl` approaches 1.0 for every chunk. The denominator simplifies to approximately `TF + k1`, removing document length as a scoring variable. Scoring becomes purely about term frequency and inverse document frequency — the two most meaningful signals for documentation retrieval.

### Query Flow with Taxonomy Translation

```
1. Developer's query arrives (via codectx query or AI-initiated)
   "How do I handle login failures?"

2. Load taxonomy.yaml internally (CLI handles this, not the AI)
   - "login" → maps to canonical "authentication" (alias match)
   - "failures" → maps to canonical "error-handling" (alias match)

3. Expand query with canonical terms and their aliases
   Original: "login failures"
   Expanded: "authentication login sign-in auth error-handling failures errors"

4. Run expanded query against ALL THREE BM25 indexes
   - Objects index → ranked instruction chunk IDs with scores
   - Specs index → ranked reasoning chunk IDs with scores
   - System index → ranked system/compiler chunk IDs with scores
   - Each index returns up to results_count results (default 10, configurable in ai.yaml)
   - Override per-call with: codectx query --top 20 "login failures"

5. Return grouped results to the AI with manifest metadata
   - Instructions section: ranked object chunks with scores, sources, token counts
   - Reasoning section: ranked spec chunks with scores, sources, token counts
   - System section: ranked system chunks with scores, sources, token counts
   - Related section: adjacent chunks not scored but potentially useful

6. AI selects which chunks to request via codectx generate
   - Can mix obj:, spec:, and sys: chunks in a single generate call
   - Makes token-budget-aware decisions using reported token counts
```

**Token budget impact**: The taxonomy.yaml is loaded by the CLI internally, not by the AI. The AI never loads taxonomy.yaml directly, keeping the taxonomy's token cost at zero for the AI's context window.

---

## Phase 5: Taxonomy System

### Design Philosophy

The taxonomy is a compiled artifact derived from the source documentation. It is not manually maintained. It serves as a controlled vocabulary with aliases that bridges the gap between how developers phrase queries and how documentation uses terminology.

The taxonomy follows a SKOS-inspired data model (W3C Simple Knowledge Organization System):
- **prefLabel**: The canonical term the documentation uses
- **altLabel**: Aliases and synonyms for that term
- **broader/narrower**: Hierarchical relationships between terms

**Reference**: W3C SKOS — https://www.w3.org/2004/02/skos/
**Reference**: Wikipedia — Automatic Taxonomy Construction — https://en.wikipedia.org/wiki/Automatic_taxonomy_construction

### Transparent AI Instructions

The LLM alias generation pass is governed by `docs/system/topics/taxonomy-generation/README.md`. This file ships as a sensible default on `codectx init` and is fully editable by the user.

If the taxonomy isn't producing the right aliases, the fix is to improve these instructions and recompile. There are no override maps, no secondary alias configurations, no hidden alias sources. One mechanism, one place, one pattern.

When a package is published, it does NOT include system/ instructions. The consumer's local system/ instructions govern how all documentation — local and from packages — gets its taxonomy generated. One set of compilation rules per project.

### Extraction Pipeline

Covered in detail in Phase 3, Stages 5 and 6. Summary:

1. **Structural extraction** (pure parsing): headings, code identifiers, bold terms, structural positions
2. **Relationship inference** (structural analysis): heading hierarchy → parent/child, cross-references → lateral
3. **POS extraction** (lightweight NLP via `prose`): noun phrases, named entities, compound terms
4. **Deduplication**: merge, score by frequency, filter by threshold
5. **LLM alias generation**: batched by taxonomy branch, governed by system/ instructions

The taxonomy is built from all three content types — instruction (.md), reasoning (.spec.md), and system (system/**/*.md). Reasoning docs often use the same canonical terms in explanatory context, and system docs use terms in meta/tooling context. This breadth improves alias generation — the LLM sees terms used in instructional, explanatory, and tooling sentences, producing broader alias coverage.

---

## Phase 6: CLI Interface

### Commands

**`codectx init`**
Initialize a new documentation package in the current directory. Defaults to `docs/` as the root directory. If `docs/` already exists and contains non-codectx content, detects the conflict and prompts the user to choose an alternative root (e.g., `ai-docs/`, `.codectx-docs/`, or a custom path). The chosen root is stored in the `root` field of codectx.yaml. Creates the directory structure with all standard directories, default `codectx.yaml`, `ai.yaml`, `preferences.yaml`, default `system/` documentation with sensible compilation instructions, and `.gitignore` additions.

**CLI discovery**: All codectx commands locate the project by walking up from the current directory looking for a codectx.yaml file (similar to how git finds `.git/`). The `root` field in codectx.yaml determines where all documentation and tooling state lives. All path references throughout this specification are relative to whatever root is configured.

**`codectx compile`**
Run the full compilation pipeline. Supports `--incremental` flag (default: true) to only reprocess changed files. Generates `.codectx/compiled/heuristics.yaml` with full diagnostic report. Prints a summary to stdout:

```
Compiled: 342 files → 4,850 chunks (2,134,500 tokens)
Taxonomy: 12,847 terms, 48,203 aliases
Session: 28,450 / 30,000 tokens (94.8%)
Changes: 3 new, 7 modified, 332 unchanged
Time: 47.3s (22.1s LLM augmentation)
```

**`codectx sync`**
Regenerate CLAUDE.md / AGENTS.md / .cursorrules from the current compiled state. Run automatically at the end of `codectx compile`, or standalone after changing `ai.yaml`.

**`codectx query "<search terms>"`**
Search the compiled documentation. Searches all three BM25 indexes (objects, specs, system) and returns results grouped by type. Supports `--top N` flag to override the default `results_count` from ai.yaml.

Internally:
1. Loads taxonomy and translates query terms
2. Runs expanded query against all three BM25 indexes (objects, specs, system)
3. Returns ranked results grouped by type with manifest metadata

Output format (returned to the AI):
```
Results for: "jwt refresh token validation"
Expanded: "jwt json-web-token bearer-token refresh-token token-validation signature-verification"

Instructions:
1. [score: 8.42] obj:a1b2c3.03 — Authentication > JWT Tokens > Refresh Flow
   Source: docs/topics/authentication/jwt-tokens.md (chunk 3/7, 462 tokens)

2. [score: 7.18] obj:a1b2c3.04 — Authentication > JWT Tokens > Validation Rules
   Source: docs/topics/authentication/jwt-tokens.md (chunk 4/7, 488 tokens)

3. [score: 5.91] obj:d4e5f6.02 — Token Service > Refresh Implementation
   Source: docs/topics/token-service/README.md (chunk 2/4, 445 tokens)

Reasoning:
1. [score: 6.85] spec:f7g8h9.02 — Authentication > JWT Tokens > Refresh Flow
   Source: docs/topics/authentication/jwt-tokens.spec.md (chunk 2/3, 380 tokens)

2. [score: 4.21] spec:j1k2l3.01 — Token Validation Strategy
   Source: docs/foundation/error-handling/README.spec.md (chunk 1/2, 290 tokens)

System:
1. [score: 3.12] sys:m3n4o5.01 — Taxonomy Alias Generation Instructions > Rules
   Source: system/topics/taxonomy-generation/README.md (chunk 1/2, 340 tokens)

Related chunks (adjacent to top instruction results, not scored):
  obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure
  obj:a1b2c3.05 — Authentication > JWT Tokens > Error Handling
```

**`codectx generate "<chunk-id>,<chunk-id>,<chunk-id>"`**
Assemble specific chunks into a single coherent reading document. Accepts `obj:`, `spec:`, and `sys:` prefixed chunk IDs in a single call, assembling a unified document with clear type demarcation.

Internally:
1. Load requested chunks (from objects/, specs/, or system/ based on prefix)
2. Sort into natural sequence order (even if requested out of order)
3. Group by type: instruction content first, system content second, reasoning content last
4. Restore heading hierarchy for document coherence
5. Insert bridge summaries at boundaries where chunks from the same file are non-adjacent
6. Format according to `output_format` in `ai.yaml`
7. Write to `/tmp/codectx/[topic-slug].[timestamp].md`
8. Tokenize and report count
9. Append footer listing related chunks not requested but adjacent to those that were

Output (returned to the AI):
```
Generated: /tmp/codectx/authentication-jwt-refresh.1741532400.md (1,772 tokens)
Contains: obj:a1b2c3.03, obj:a1b2c3.04, obj:d4e5f6.02, spec:f7g8h9.02

Related chunks not included:
  obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure (488 tokens)
  obj:a1b2c3.05 — Authentication > JWT Tokens > Error Handling (412 tokens)
  spec:f7g8h9.01 — Authentication > JWT Tokens (380 tokens)
```

**`codectx install`**
Install packages declared in `codectx.yaml`. Resolves `[name]@[org]` references to `github.com/[org]/codectx-[name]`, resolves version tags, clones or fetches to `.codectx/packages/`. If `codectx.lock` exists and `codectx.yaml` hasn't changed, installs from the lock file using pinned commit SHAs (fast, deterministic). If `codectx.yaml` changed, re-resolves affected entries and updates the lock. Resolves transitive dependencies.

**`codectx update`**
Re-resolve all dependencies to their latest compatible versions, update `codectx.lock`, download any changed packages, and recompile if package content changed. This is the command for pulling in newer versions of dependencies — the user should never need flags on `install` for this.

Output:
```
Resolving dependencies...
  react-patterns@community: 2.3.1 → 2.4.0 (updated)
    → github.com/community/codectx-react-patterns@v2.4.0
  company-standards@acme: 2.0.0 (unchanged)
  tailwind-guide@designteam: 2.1.0 (unchanged)
  javascript-fundamentals@community: 1.3.0 (transitive, unchanged)

Updated codectx.lock
Downloaded: react-patterns@community:2.4.0

Recompiling (1 package changed)...
Compiled: 348 files → 4,803 chunks (2,178,200 tokens)
Taxonomy: 12,921 terms, 48,890 aliases
Session: 28,450 / 30,000 tokens (94.8%)
Time: 31.2s
```

**`codectx search "<query>"`**
Search for packages on GitHub using the `codectx-*` naming convention via the GitHub API.

Output:
```
Search results for: "react patterns"

1. react-patterns@community (v2.4.0) ★ 342
   github.com/community/codectx-react-patterns
   React component and hook patterns for AI-driven development

2. react-testing@community (v1.1.0) ★ 89
   github.com/community/codectx-react-testing
   Testing patterns and strategies for React applications

3. react-nextjs@webteam (v3.0.2) ★ 156
   github.com/webteam/codectx-react-nextjs
   Next.js and React integration patterns

Install with: codectx install react-patterns@community:latest
```

**`codectx publish`**
Publish the current package to GitHub. Reads codectx.yaml for name, org, and version. Validates directory structure. Tags the current commit as `v[version]` and pushes to `github.com/[org]/codectx-[name]`. The repo must already exist on GitHub — `codectx publish` handles tagging and validation, not repo creation.

**`codectx session add <reference>`**
Add a local path or package reference to the always-loaded session context. Modifies `session.always_loaded` in `codectx.yaml`. Reports the token cost of the added entry and the new total against the budget.

**`codectx session remove <reference>`**
Remove an entry from the always-loaded session context.

**`codectx session list`**
List all always-loaded session context entries with individual token counts and total against the budget.

**`codectx plan status [plan-name]`**
Report the current state of a plan without loading context. Reads `plan.yaml` and returns the current step, completion percentage, blockers, dependency hash status (changed or unchanged), and stored queries for the current step.

Output:
```
Plan: Authentication System Migration
Status: in-progress (step 3 of 5)
Progress: 2 steps completed, 1 in progress, 2 pending

Current step: Implement token service refactor
  Started: 2025-03-07T09:00:00Z
  Notes: User service and payment service updated. Order service remaining.
  Stored queries:
    - "token service refactor implementation"
    - "order service authentication"

Dependencies:
  ✓ foundation/architecture-principles — unchanged
  ⚠ topics/authentication/jwt-tokens — content changed since last update
  ✓ topics/authentication/oauth — unchanged

Blocked steps:
  Step 4 (Migration testing) — blocked by step 3
  Step 5 (Production rollout) — blocked by step 4
```

**`codectx plan resume [plan-name]`**
Resume a plan by reconstructing its context. Checks dependency hashes against current compiled state. If all hashes match, replays the current step's stored chunks via `codectx generate` for instant context reconstruction. If hashes changed, reports which dependencies drifted and provides the stored queries for the AI to re-run against updated documentation. Returns the assembled context plus plan state.

Output (hashes match — instant replay):
```
Plan: Authentication System Migration
Status: in-progress (step 3 of 5)
Dependencies: all unchanged ✓

Replaying context for step 3...
Generated: /tmp/codectx/auth-migration-step3.1741532400.md (1,847 tokens)
Contains: obj:a1b2c3.04, obj:d4e5f6.02, obj:d4e5f6.03, obj:x9y8z7.01, spec:x9y8z7.01

Current step: Implement token service refactor
Notes: User service and payment service updated. Order service remaining.
```

Output (hashes changed — guided re-search):
```
Plan: Authentication System Migration
Status: in-progress (step 3 of 5)

Documentation changes since last update:
  ⚠ topics/authentication/jwt-tokens — content changed
  ✓ foundation/architecture-principles — unchanged
  ✓ topics/authentication/oauth — unchanged

Stored chunks may be stale. Stored queries for current step:
  - "token service refactor implementation"
  - "order service authentication"

Recommendation: Re-run stored queries to refresh context with updated documentation.
```

### Generated File Format

When `codectx generate` assembles chunks, the output is a coherent reading document with content grouped by type: Instructions first, then System (if any `sys:` chunks requested), then Reasoning. Each section is clearly demarcated so the AI understands what is actionable, what is tooling context, and what is explanatory reasoning:

```markdown
<!-- codectx:generated
chunks: obj:a1b2c3.03, obj:a1b2c3.04, obj:d4e5f6.02, spec:f7g8h9.02
sources:
  - docs/topics/authentication/jwt-tokens.md
  - docs/topics/authentication/jwt-tokens.spec.md
  - docs/topics/token-service/README.md
tokens: 1772
generated: 2025-03-09T12:00:00Z
-->

# Instructions

## Authentication > JWT Tokens

### Refresh Flow

The refresh token lifecycle begins when...
[chunk obj:a1b2c3.03 content]

### Validation Rules

Token validation follows a strict sequence...
[chunk obj:a1b2c3.04 content]

---
> **Context bridge**: The above sections covered JWT token management from the
> authentication documentation. The following section is from a different
> document (Token Service) that implements these patterns.
---

## Token Service

### Refresh Implementation

The token service exposes a refresh endpoint...
[chunk obj:d4e5f6.02 content]

---

# Reasoning

> The following sections contain the reasoning behind the instructions above.
> This is informational context for understanding *why* decisions were made.
> Reason about this content before acting on it.

## Authentication > JWT Tokens > Refresh Flow

The refresh token lifecycle was designed around the constraint that...
[chunk spec:f7g8h9.02 content]

---
<!-- codectx:related
Adjacent chunks not included in this document:
- obj:a1b2c3.02: "Authentication > JWT Tokens > Token Structure" (488 tokens)
- obj:a1b2c3.05: "Authentication > JWT Tokens > Error Handling" (412 tokens)
- spec:f7g8h9.01: "Authentication > JWT Tokens" (380 tokens)
Use: codectx generate "obj:a1b2c3.02,obj:a1b2c3.05,spec:f7g8h9.01" to load these
-->
```

---

## Phase 7: Plan State Tracking

### plan.yaml Schema

```yaml
# docs/plans/auth-migration/plan.yaml
name: "Authentication System Migration"
status: "in-progress"  # draft | in-progress | blocked | completed
created: "2025-03-01T00:00:00Z"
updated: "2025-03-09T12:00:00Z"

# Documentation this plan depends on
# Each dependency tracks a content hash at the time the plan was last updated
# Used to detect documentation drift during plan resumption
dependencies:
  - path: "foundation/architecture-principles"
    hash: "sha256:a1b2c3..."
  - path: "topics/authentication/jwt-tokens"
    hash: "sha256:d4e5f6..."
  - path: "topics/authentication/oauth"
    hash: "sha256:g7h8i9..."

# Plan steps with state, context queries, and chunk references
# queries: the search terms the AI used to find relevant documentation for this step
# chunks: the codectx generate calls the AI made (each entry is one generate call)
#         directly replayable if dependency hashes haven't changed
steps:
  - id: 1
    title: "Audit current JWT implementation"
    status: "completed"
    completed_at: "2025-03-02T14:00:00Z"
    notes: "Found 3 services using deprecated token format"
    queries:
      - "jwt token implementation current"
      - "token validation service audit"
    chunks:
      - "obj:a1b2c3.01,obj:a1b2c3.02,obj:a1b2c3.03,spec:f7g8h9.01"

  - id: 2
    title: "Design new token schema"
    status: "completed"
    completed_at: "2025-03-05T10:00:00Z"
    queries:
      - "jwt token schema design"
      - "refresh token lifecycle"
    chunks:
      - "obj:a1b2c3.03,obj:a1b2c3.04,obj:d4e5f6.02,spec:f7g8h9.02"

  - id: 3
    title: "Implement token service refactor"
    status: "in-progress"
    started_at: "2025-03-07T09:00:00Z"
    notes: "User service and payment service updated. Order service remaining."
    queries:
      - "token service refactor implementation"
      - "order service authentication"
    chunks:
      - "obj:a1b2c3.04,obj:d4e5f6.02,obj:d4e5f6.03"
      - "obj:x9y8z7.01,spec:x9y8z7.01"

  - id: 4
    title: "Migration testing"
    status: "pending"
    blocked_by: [3]

  - id: 5
    title: "Production rollout"
    status: "pending"
    blocked_by: [4]

current_step: 3
```

**Key schema elements:**

- **dependencies with `hash`**: Each dependency records a content hash at the time the plan was last updated. This enables drift detection — the CLI can tell whether the documentation the plan was built against has changed.

- **Per-step `queries`**: The search terms the AI used to find relevant documentation for each step. These are preserved so that if the plan resumes and hashes have changed, the AI has the original search intent to re-run against the updated documentation rather than searching blind.

- **Per-step `chunks`**: Each entry is a comma-delimited string of chunk IDs matching the `codectx generate` input format — directly replayable. Multiple entries mean the AI made multiple generate calls during that step. If dependency hashes haven't changed, these chunk IDs are still valid and can be replayed for instant context reconstruction.

- **Pending steps have no queries or chunks**: These are populated by the AI as it begins working on each step.

### Resumption Flow

`codectx plan resume auth-migration` performs the following:

1. Read plan.yaml, identify current_step (step 3)
2. Check each dependency's current content hash against its `hash`
3. **If all hashes match** (documentation unchanged):
   - Replay step 3's chunks via `codectx generate` for each entry
   - Return assembled context plus plan state
   - AI has exact context reconstruction — instant resumption
4. **If any hashes changed** (documentation drifted):
   - Report which dependencies changed:
     ```
     Plan: Authentication System Migration
     Status: in-progress (step 3 of 5)
     
     Documentation changes since last update:
       ⚠ topics/authentication/jwt-tokens — content changed
       ✓ foundation/architecture-principles — unchanged
       ✓ topics/authentication/oauth — unchanged
     
     Stored queries for current step:
       - "token service refactor implementation"
       - "order service authentication"
     
     Recommendation: Review changes to jwt-tokens before continuing.
     Re-run stored queries to refresh context with updated documentation.
     ```
   - AI uses the stored queries to run `codectx query` for each, selects new chunks from fresh results
   - AI updates plan.yaml with new chunks and hashes once it has re-established context

### Context Audit Trail

Even for completed steps, the stored queries and chunks create a valuable record of *what documentation the AI relied on* to complete that work. If a problem surfaces later, the team can trace back: "Step 2 was completed using these chunks from these queries — did the AI have the right context when it made those decisions?"

This also enables plan handoffs between developers. Developer A starts the plan, their AI finds chunks through certain queries and logs them. Developer B picks it up on a different machine. If hashes match, the exact chunks load. If not, developer B's AI doesn't need to figure out *what to search for* from scratch — the stored queries give it the same search intent, adapted to the current documentation state.

### Design Considerations

- **plan.yaml must be merge-friendly**: Keep it declarative (current state), not accumulative (history log). History lives in git commits.
- **plan.yaml is checked into version control**: This is source content, not compiled output. It's the mechanism for cross-machine, cross-developer continuity.
- **Token cost**: plan.yaml is small enough to load directly. No chunking needed for plan state files.
- **Chunk IDs are stable across machines**: Chunk IDs are content hashes of the chunk text. Same source markdown + same preferences.yaml + same tokenizer encoding = same chunks, same hashes, on every machine. This is what makes chunk-level references safe to check in.
- **Drift detection, not drift prevention**: The system detects when documentation changed under a plan and reports it. It doesn't prevent the developer from continuing — it provides the information for an informed decision.

---

## Phase 8: Package Manager

### What a Package Is

A package is curated documentation content — nothing more. It consists of:
- `foundation/` (optional)
- `topics/` (optional)
- `plans/` (optional)
- `prompts/` (optional)
- `codectx.yaml` (required — name, org, version, description)

No `system/` directory. No `.codectx/` directory. No compiler configuration. No AI instructions. Packages are pure content that gets processed by the consumer's compiler with the consumer's settings.

This means package authoring has near-zero friction. Write markdown. Organize it into the standard directories. Publish. You don't need to understand the compilation pipeline, BM25, or taxonomy extraction. The quality of the compiled output depends on the structure and content of the markdown itself.

### Registry: GitHub

Packages are public GitHub repositories using the `codectx-[name]` naming convention. No custom registry infrastructure is needed — GitHub provides hosting, versioning (git tags), discovery (API search), and built-in quality signals (stars, issues, commit activity).

**Naming convention**: A package named `react-patterns` under org `community` lives at `github.com/community/codectx-react-patterns`. The `codectx-` prefix is the namespace convention that makes packages discoverable. The dependency reference `react-patterns@community` maps to this repo automatically.

**Versioning**: Git tags are versions. Tag `v2.3.1` on the repo corresponds to version `2.3.1` in codectx.yaml. The `latest` reference resolves to the most recent semver tag.

**Discovery**: The GitHub search API finds all repos matching the `codectx-*` pattern. `codectx search "react patterns"` would query the API and return matching packages with their descriptions, stars, and latest versions.

### Publishing and Consuming

**Publishing**:
A package is just a GitHub repo with the `codectx-` prefix that contains the standard directory structure and a codectx.yaml. Publishing is creating or updating the repo and tagging a version.

```bash
codectx publish
# Reads codectx.yaml for name, org, version
# Validates directory structure
# Tags the current commit as v[version]
# Pushes to github.com/[org]/codectx-[name]
# Consumers install by referencing [name]@[org]
```

**Consuming**:
```yaml
# In codectx.yaml
# react-patterns@community → github.com/community/codectx-react-patterns
dependencies:
  react-patterns@community:latest:
    active: true
  company-standards@acme:2.0.0:
    active: true
```

```bash
codectx install
# Resolves [name]@[org] to github.com/[org]/codectx-[name]
# Resolves version tags (latest → highest semver tag)
# Clones/fetches to .codectx/packages/
# Generates or updates codectx.lock with resolved versions and commit SHAs
# Next codectx compile processes them alongside local docs
```

### Active/Inactive Toggle

Packages can be toggled active/inactive in `codectx.yaml`. Inactive packages remain installed in `.codectx/packages/` but are excluded from compilation — their chunks aren't indexed, their taxonomy terms aren't included, and they consume zero tokens at runtime. This lets developers install many reference packages and selectively enable only what's relevant to their current task.

When a direct dependency is deactivated, its transitive-only dependencies also deactivate. If a transitive dependency is also required by another active direct dependency, it stays active.

### Transitive Dependencies

Published packages can declare their own dependencies in their codectx.yaml. When you install a package, its dependencies are resolved and installed transitively. A React patterns package that depends on a JavaScript fundamentals package will cause both to be installed.

**Resolution strategy**: All dependencies — direct and transitive — are flattened into `.codectx/packages/` at the same level. No nesting. If two packages depend on the same package at compatible semver ranges, the resolver picks the highest compatible version and installs it once. If two packages have incompatible version requirements for the same dependency, the installer warns and the developer resolves the conflict manually.

Flat resolution is safe for documentation packages in a way it isn't for executable code. A JavaScript fundamentals package at version 1.2.0 versus 1.3.0 probably added new topics but didn't break existing ones. Documentation content is additive, not behaviorally breaking.

### codectx.lock Schema

The lock file captures the full flattened dependency tree with exact resolved versions. Checked into version control for deterministic reproducibility.

```yaml
# docs/codectx.lock
# Auto-generated by codectx install. Do not edit manually.
lockfile_version: 1
resolved_at: "2025-03-09T12:00:00Z"

packages:
  react-patterns@community:
    resolved_version: "2.3.1"
    repo: "github.com/community/codectx-react-patterns"
    commit: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"  # Git tag v2.3.1
    source: "direct"  # Declared in project codectx.yaml

  company-standards@acme:
    resolved_version: "2.0.0"
    repo: "github.com/acme/codectx-company-standards"
    commit: "d4e5f6g7h8i9d4e5f6g7h8i9d4e5f6g7h8i9d4e5"
    source: "direct"

  javascript-fundamentals@community:
    resolved_version: "1.3.0"
    repo: "github.com/community/codectx-javascript-fundamentals"
    commit: "g7h8i9j0k1l2g7h8i9j0k1l2g7h8i9j0k1l2g7h8"
    source: "transitive"  # Not directly declared — pulled in by a dependency
    required_by:
      - "react-patterns@community:2.3.1"
```

The `source` field distinguishes direct dependencies from transitive ones so the developer can see why a package was installed. The `required_by` field traces the dependency chain. The git `commit` SHA pins the exact content — no separate integrity hash is needed since git already provides content-addressed storage.

**Install and update behavior**:
- `codectx install` — if `codectx.lock` exists and `codectx.yaml` hasn't changed, install from lock (fast, deterministic). If `codectx.yaml` changed, re-resolve affected entries and update lock.
- `codectx update` — re-resolve all dependencies to latest compatible versions, update `codectx.lock`, download changed packages, and recompile if content changed.

### Quality Signal

The compiler itself is the quality contract. A package that compiles cleanly with comprehensive taxonomy coverage, well-structured chunks, and complete manifests is a good package. A package with sparse taxonomy, oversized chunks, and validation warnings is visible as lower quality through compilation reports. The ecosystem self-corrects as publishers learn what produces good compiled output.

No enforcement of cross-package terminology consistency is needed. Each package's content feeds into the single project-level taxonomy. The consumer's `system/topics/taxonomy-generation/` instructions govern how aliases are generated for all content uniformly.

---

## Phase 9: Implementation Roadmap

### Recommended Build Order

1. **Package structure and init command**: Create the directory layout, codectx.yaml, ai.yaml, preferences.yaml, default system/ docs. This is the foundation everything else builds on.

2. **Markdown parsing and stripping**: Parse markdown to AST, strip human-formatting overhead. This produces the cleaned content that all subsequent stages consume.

3. **Token counting integration**: Integrate `tiktoken-go/tokenizer`. Needed before chunking since chunk boundaries are token-based.

4. **Chunking algorithm**: Implement semantic block accumulation with token-based windows. Generate chunk files with context headers.

5. **BM25 indexing**: Integrate `crawlab-team/bm25` or equivalent. Build the inverted index over chunks.

6. **Manifest generation**: Generate manifest.yaml with chunk metadata, adjacency information, bridge placeholders, and token counts. Generate metadata.yaml with document relationships. Generate hashes.yaml for incremental compilation. Generate heuristics.yaml with compilation diagnostics.

7. **CLI query and generate commands**: Implement the search and assembly interface the AI will use.

8. **Context assembly and sync**: Compile always-loaded foundations into context.md. Generate CLAUDE.md entry points.

9. **Taxonomy extraction** (structural pass): Extract terms from headings, code identifiers, structural positions.

10. **POS-based extraction**: Integrate `prose` library for noun phrase and named entity extraction.

11. **LLM augmentation**: Implement batched alias generation and boundary bridge summaries, governed by system/ instructions.

12. **Incremental compilation**: Add content hash tracking and change detection for fast recompilation.

13. **Plan state tracking**: Implement plan.yaml schema, plan status command, and plan resume command with dependency hash checking and chunk replay.

14. **Package manager**: Implement install, update, publish, search (via GitHub API), dependency resolution, active/inactive toggling. Packages are GitHub repos with `codectx-[name]` convention, versions are git tags, lock file pins commit SHAs.

### Key Go Packages Summary

| Package | Purpose | Notes |
|---------|---------|-------|
| `github.com/tiktoken-go/tokenizer` | Token counting | Pure Go, embeds vocabulary (~4MB), cl100k_base default |
| `github.com/crawlab-team/bm25` | BM25 search indexing | Full variant support, parallel scoring, port of rank_bm25 |
| `github.com/jdkato/prose/v2` | POS tagging, NER | Pure Go, English, tokenization + POS + entities |
| `github.com/covrom/bm25s` | Short-text BM25 (alt) | Auto-adjusts params by doc length |
| `github.com/raphaelsty/gokapi` | Disk-backed BM25 (alt) | For memory-constrained environments |

### Design Decisions Log

| Decision | Chosen | Rejected | Reasoning |
|----------|--------|----------|-----------|
| Search approach | BM25 keyword search | Vector RAG | Accuracy, traceability, no embedding dependency. RAG retrieval accuracy below 60% in practice. BM25 is deterministic and debuggable. |
| Chunk strategy | Token-counted semantic blocks | Heading-based splitting | Heading-based fails on large single-heading sections. Token-based produces uniform sizes for consistent BM25 scoring. |
| Chunk continuity | Boundary reference map in manifest | Content overlap between chunks | Reference map costs ~15-20 tokens per boundary vs 50-100 for overlap. Every token in the context window is discrete — no deduplication at the model level. Reference loaded once in manifest, not duplicated across chunks. |
| Taxonomy source | Compiled from documentation | Hand-curated | Must be programmatic for package ecosystem. Compiler derives taxonomy like a symbol table. LLM augments at compile time. |
| Taxonomy alias overrides | Edit system/ instructions and recompile | query_overrides in ai.yaml or manual_aliases in preferences.yaml | One mechanism, one place, one pattern. No secondary alias sources. If taxonomy is wrong, fix the instructions that generate it. |
| Compiler AI instructions | Editable markdown in docs/system/ | Hardcoded in Go source | Transparency. Users can read, understand, and modify how the compiler's AI behaves. Instructions version-controlled alongside documentation. |
| Token counting | tiktoken-go cl100k_base | Word count approximation | Different models tokenize differently but cl100k_base is within 10% variance for English prose. Precise enough for budgeting. |
| Model targeting | ai.yaml configuration | Baked into compiled packages | Compiled packages must be model-agnostic for ecosystem portability. Model-specific behavior is a runtime concern. |
| Always-loaded context | Compiled context.md via CLAUDE.md pointer | Foundation docs directly in CLAUDE.md | Single source of truth. Foundation updates only require recompilation, not manual CLAUDE.md editing. |
| metadata.yaml | Fully generated, gitignored | Checked in to avoid recompilation cost | Merge conflict magnet. Regeneration is the cheapest stage in the pipeline. |
| Dependency packages location | .codectx/packages/ | docs/packages/ | docs/ should be exclusively human-authored source content. Dependencies are tooling-managed artifacts that belong under .codectx/ alongside other tooling state. |
| Build cache | No separate cache/ directory | Dedicated cache/ directory | Compiled artifacts serve as their own cache. taxonomy.yaml persists aliases, manifest.yaml persists bridges, hashes.yaml tracks file changes. No intermediate artifacts needed. |
| Package contents | Pure content only (foundation, topics, plans, prompts) | Include system/ instructions and compiler config | One set of compilation rules per project. Consumer's compiler processes all content uniformly. Package authoring stays simple. |
| Query results count | Configurable results_count in ai.yaml with --top CLI override | Vague "conservative"/"aggressive" retrieval_strategy | Concrete, configurable, overridable per-call. The AI or developer controls exactly how many results they get. Default 10. |
| Chunk type separation | Three separate directories (objects/, specs/, system/) with independent BM25 indexes | Mixed chunks in single directory and index | Instructions, reasoning, and system docs use different language patterns with different term frequency distributions. Three indexes keep scoring clean within each type. Enables independent searches per type. |
| System documentation compilation | system/ uses standard subdirectory layout (foundation/, topics/, plans/, prompts/) and compiles into its own chunks and BM25 index | system/ only read as raw instructions, not indexed | Makes compiler documentation searchable through the same mechanism as everything else. The compiler can read raw files as instructions at compile time AND the compiled chunks are searchable at query time — no conflict. |
| Package registry | GitHub repos with `codectx-[name]` convention, discovered via GitHub API | Custom registry infrastructure | Zero infrastructure to build and maintain. Git tags are versions. Commit SHAs are integrity verification. GitHub provides hosting, discovery (API search), and quality signals (stars, issues, activity). Naming convention makes discovery trivial. |
| Spec chunk surfacing | Shown as separate "Reasoning" section in query results | Pre-linked to parent instruction chunks only | Independent BM25 search over specs enables reasoning-focused queries like "why do we use repository pattern" that wouldn't match instruction content. |
| Config file naming | `codectx.yaml` and `codectx.lock` | `package.yaml` and `package.lock` | Tool-named files are self-identifying in directory listings and PR diffs. Avoids collision with other tools that claim the `package.*` namespace. Follows convention of docker-compose.yaml, tsconfig.json, Cargo.toml. |
| Dependency lock file | `codectx.lock` checked into version control | No lock file, or lock file gitignored | Deterministic reproducibility. Two developers running `codectx install` on the same codectx.yaml get identical package versions. Same pattern as package-lock.json, Cargo.lock, Gemfile.lock. |
| Transitive dependencies | Flat resolution, highest compatible version wins | Nested dependencies or no transitive support | Documentation packages are additive, not behaviorally breaking. Flat resolution avoids npm-style nesting complexity. Incompatible versions warn rather than silently break. |
| Session context management | `codectx session add/remove/list` with `session.always_loaded` in codectx.yaml | `codectx context` commands with `context.always_loaded` | "Context" is already overloaded in this system (context window, context headers, context.md, context budget, context bridge). "Session" is unambiguous — it means what the AI gets when it starts working. |
| Published vs project codectx.yaml | Two schemas sharing common identity fields, project adds session/active/registry | Single schema for both | Published packages shouldn't dictate consumer's session context or active/inactive state. Separation keeps package authoring simple and consumer configuration flexible. |
| Plan context tracking | Per-step queries and chunk IDs with dependency hash drift detection | Directory-path-only references or global chunk lists | Per-step keeps context scoped — resuming step 3 doesn't load step 1's chunks. Stored queries enable re-search when docs drift. Stored chunks enable instant replay when docs haven't changed. Hash comparison bridges the two modes automatically. |
| Compilation report | heuristics.yaml in compiled/ — regenerated every compile | No report, or stdout-only logging | Machine-parseable diagnostics let the AI orient itself on the documentation landscape. Humans get timing, budget utilization, and quality signals. Snapshot, not cumulative — always reflects current state. |

---

## Appendix A: Token Counting Reference

### Why Tokens, Not Words or Characters

- The same text produces different token counts depending on the tokenizer
- Code blocks tokenize very differently from prose (each symbol is often its own token)
- A 50-word code block might be 200 tokens
- A 50-word prose paragraph might be 65 tokens
- Word count is not a reliable proxy for context window consumption
- Every token in the context window is a discrete token — there is no deduplication, compression, or reference-counting at the model level. Repeated content is counted and billed twice.

### Encoding Selection

Configured in `ai.yaml` under `compilation.encoding`. Default: `cl100k_base`.

| Encoding | Models | Notes |
|----------|--------|-------|
| cl100k_base | GPT-4, GPT-4 Turbo, GPT-3.5 Turbo | Most widely used. Good baseline. |
| o200k_base | GPT-4o | Newer encoding |
| Claude tokenizer | Claude models | No public Go library; cl100k_base within ~10% for English |

**Practical guidance**: cl100k_base is close enough across modern models for budgeting purposes. The ~10% variance doesn't change chunking decisions. Use exact model-specific tokenization only if precise billing estimation is required.

### Where Token Counts Are Used

1. **Chunk boundary decisions**: Accumulate semantic blocks until reaching `target_tokens`
2. **Session budget enforcement**: Always-loaded session entries vs. declared budget in `codectx.yaml`
3. **Manifest metadata**: Each chunk entry includes its token count
4. **Generate output reporting**: CLI reports token count of assembled documents
5. **Validation warnings**: Files exceeding `max_file_tokens` get flagged

---

## Appendix B: SKOS Data Model Reference

The taxonomy uses a simplified SKOS-inspired schema. Key concepts from SKOS:

- **Concept**: A single idea or term in the taxonomy (a node)
- **prefLabel**: The canonical/preferred label for a concept (one per concept)
- **altLabel**: Alternative labels — synonyms, abbreviations, variations (many per concept)
- **broader**: Parent concept in the hierarchy
- **narrower**: Child concepts in the hierarchy
- **related**: Lateral relationships to other concepts (non-hierarchical)
- **hiddenLabel**: Terms that should match in search but never display — useful for common misspellings or legacy terminology

**Full SKOS reference**: https://www.w3.org/TR/skos-reference/

In codectx, SKOS concepts are serialized as YAML. The data model is borrowed; the Semantic Web serialization infrastructure (RDF, URIs) is not.

---

## Appendix C: References

### Standards and Specifications
- W3C SKOS Simple Knowledge Organization System — https://www.w3.org/2004/02/skos/
- W3C SKOS Reference — https://www.w3.org/TR/skos-reference/
- Automatic Taxonomy Construction (Wikipedia) — https://en.wikipedia.org/wiki/Automatic_taxonomy_construction

### Go Packages
- tiktoken-go/tokenizer — https://github.com/tiktoken-go/tokenizer
- crawlab-team/bm25 — https://github.com/crawlab-team/bm25
- jdkato/prose — https://github.com/jdkato/prose
- covrom/bm25s — https://pkg.go.dev/github.com/covrom/bm25s
- raphaelsty/gokapi — https://github.com/raphaelsty/gokapi
- leejuyuu/bm25s-go — https://pkg.go.dev/codeberg.org/leejuyuu/bm25s-go

### Background Reading
- BM25 algorithm explanation — Elastic Blog "Practical BM25" series — https://www.elastic.co/blog/practical-bm25-part-2-the-bm25-algorithm-and-its-variables
- File-first AI agent approach — Denis Urayev — https://medium.com/@denisuraev/rag-is-dead-before-you-build-it-try-file-first-ai-agent-f51bfe693a55
- RAKE keyword extraction algorithm — documented in prose Go NLP ecosystem
- OpenAI tiktoken cookbook — https://developers.openai.com/cookbook/examples/how_to_count_tokens_with_tiktoken/
