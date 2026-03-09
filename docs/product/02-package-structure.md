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

