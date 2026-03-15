# Configuration

codectx uses four configuration files, each with a distinct purpose and version control policy.

---

## codectx.yml

The project manifest. Defines package identity, dependencies, session context, and registry configuration. Checked into version control.

```yaml
# Documentation root directory (default: "docs")
root: "docs"

# Package identity
name: "my-project-docs"
org: "myteam"
version: "1.2.0"
description: "Engineering documentation for the my-project codebase"

# Always-loaded session context
session:
  always_loaded:
    - foundation/coding-standards
    - foundation/architecture-principles
    - foundation/error-handling
    - react-patterns@community/foundation/component-principles
    - company-standards@acme
  budget: 30000    # Maximum tokens for session context

# Package dependencies
dependencies:
  react-patterns@community:latest:
    active: true
  company-standards@acme:2.0.0:
    active: true
  legacy-api-docs@internal:1.0.0:
    active: false    # Installed but excluded from compilation

# Package manager settings
registry: "github.com"
```

### Key Fields

| Field | Purpose |
|-------|---------|
| `root` | Documentation root directory. Change if `docs/` conflicts with existing content. |
| `name`, `org`, `version` | Package identity. Used by the package manager and during publishing. |
| `session.always_loaded` | Documents compiled into `context.md`. Order matters. Supports local paths and package references. |
| `session.budget` | Token ceiling for assembled session context. Advisory — compilation warns but doesn't fail on overflow. |
| `dependencies` | Package dependencies with version specifiers and active/inactive flags. |
| `registry` | Where to resolve packages. Default: `github.com`. |

### Published Package codectx.yml

Published packages use a minimal schema — identity and dependency declarations only:

```yaml
name: "react-patterns"
org: "community"
version: "2.3.1"
description: "React component and hook patterns for AI-driven development"

dependencies:
  javascript-fundamentals@community: ">=1.0.0"
```

No `session`, `active` flags, or `registry`. The consumer controls all of these.

---

## ai.yml

AI model and behavior configuration. Checked into version control (no secrets).

Located at `docs/.codectx/ai.yml`.

```yaml
# Model used during compilation for taxonomy and bridge generation
compilation:
  model: "claude-sonnet-4-20250514"
  encoding: "cl100k_base"    # Tokenizer encoding for token counting

# Target model for consumption
consumption:
  model: "claude-sonnet-4-20250514"
  context_window: 200000
  results_count: 30    # Default results returned by codectx query
                       # Override per-call with: codectx query --top N

# Formatting preference for generated output
output_format: "markdown"    # markdown | xml_tags | plain
```

### Key Fields

| Field | Purpose |
|-------|---------|
| `compilation.model` | Model identifier for compile-time AI passes. |
| `compilation.encoding` | Tokenizer encoding for token counting. `cl100k_base` is the default and works within ~10% variance across modern models. |
| `consumption.results_count` | Default number of results from `codectx query`. Can be overridden per-call with `--top N`. |
| `output_format` | How generated documents are formatted: `markdown`, `xml_tags`, or `plain`. |

### Taxonomy and Aliases

`ai.yml` does not contain query translation overrides or alias mappings. All term aliasing is handled through the taxonomy, which is governed by the instructions in `system/topics/taxonomy-generation/`. If the taxonomy isn't producing the right aliases, edit those instructions and recompile. One mechanism, one place, one pattern.

---

## ai.local.yml

Local-only overrides for API keys, endpoints, and personal preferences. Always gitignored — never committed.

Located at `docs/.codectx/ai.local.yml`.

```yaml
# API credentials
api_key: "sk-..."
endpoint: "https://api.anthropic.com"

# Override compilation model locally
compilation:
  model: "claude-sonnet-4-20250514"
```

This file overrides fields from `ai.yml` on the local machine only. Use it for API keys that should never enter version control, or for local model preferences that differ from the team default.

---

## preferences.yml

Compiler and pipeline configuration. Checked into version control.

Located at `docs/.codectx/preferences.yml`.

```yaml
# Chunk compilation settings
chunking:
  target_tokens: 450     # Target chunk size in tokens
  min_tokens: 200        # Minimum chunk size (avoid tiny fragments)
  max_tokens: 800        # Maximum chunk size (hard ceiling)
  flexibility_window: 0.8 # Break after 80% of target if next block would exceed

# BM25F field scoring
bm25f:
  k1: 1.2
  fields:
    heading:
      weight: 3.0
      b: 0.3
    terms:
      weight: 2.0
      b: 0.0
    body:
      weight: 1.0
      b: 0.75
    code:
      weight: 0.6
      b: 0.5

# Query pipeline configuration
query:
  expansion:
    enabled: true
    alias_weight: 1.0
    narrower_weight: 0.7
    related_weight: 0.4
    max_expansion_terms: 20

  rrf:
    k: 60
    index_weights:
      objects: 1.0
      specs: 0.7
      system: 0.3

  graph_rerank:
    enabled: true
    adjacent_boost: 0.15
    spec_boost: 0.20
    cross_ref_boost: 0.10

# Taxonomy extraction settings
taxonomy:
  min_term_frequency: 2
  max_alias_count: 10
  pos_extraction: true

# Scaffold maintenance
scaffold_maintenance: true

# Documentation validation
validation:
  require_readme: true
  require_spec: false
  max_file_tokens: 10000
  require_headings: true

# Prompt command auto-selection
prompt:
  budget_multiplier: 4.0
  budget_delta: 0.0
```

### Chunking Settings

| Field | Default | Purpose |
|-------|---------|---------|
| `target_tokens` | 450 | Target chunk size. Chunks accumulate semantic blocks until reaching this count. |
| `min_tokens` | 200 | Minimum chunk size. Prevents tiny fragments. |
| `max_tokens` | 800 | Hard ceiling. Single atomic blocks (code, tables) can exceed this — they become flagged oversized chunks. |
| `flexibility_window` | 0.8 | Break after this fraction of target if the next block would exceed. Keeps sizes consistent. |

### Query Pipeline Settings

| Section | Purpose |
|---------|---------|
| `query.expansion` | Taxonomy query expansion weights and limits. See [Search and Retrieval](search-and-retrieval.md). |
| `query.rrf` | Reciprocal Rank Fusion parameters. `k` is the smoothing constant. `index_weights` control per-index contribution. |
| `query.graph_rerank` | Graph re-ranking boost multipliers for adjacent, spec, and cross-referenced chunks. |

### Taxonomy Settings

| Field | Default | Purpose |
|-------|---------|---------|
| `min_term_frequency` | 2 | Minimum corpus-wide frequency to include a term in the taxonomy. |
| `max_alias_count` | 10 | Maximum aliases per canonical term. |
| `pos_extraction` | true | Enable POS-based term extraction (noun phrases, named entities). |

### Validation Settings

| Field | Default | Purpose |
|-------|---------|---------|
| `require_readme` | true | Every topic directory must have a README.md. |
| `require_spec` | false | Spec files recommended but not required. |
| `max_file_tokens` | 10000 | Warn if a single source file exceeds this token count. |
| `require_headings` | true | Warn if a file has no heading structure. |

### Prompt Settings

The `codectx prompt` command auto-selects chunks within a computed token budget: `budget = target_tokens x budget_multiplier x (1 + budget_delta)`.

| Field | Default | Purpose |
|-------|---------|---------|
| `budget_multiplier` | 4.0 | Base multiplier. Controls how many chunks worth of content are included. At the default chunk target of 450 tokens, multiplier 4 = ~1,800 tokens (~4 chunks). |
| `budget_delta` | 0.0 | Incremental scaling. `0.1` = +10% budget, `-0.2` = -20%. Override per-call with `codectx prompt --delta 0.5`. |

### Scaffold Maintenance

When `scaffold_maintenance` is `true` (default), `codectx compile` auto-repairs missing directories, restores deleted system default files, and manages `.gitkeep` files in empty content directories. `codectx repair` always runs regardless of this setting.

---

## .gitignore

codectx manages `.gitignore` entries automatically. The standard entries:

```gitignore
# codectx — tooling state and compiled output
docs/.codectx/compiled/
docs/.codectx/packages/
docs/.codectx/history/
docs/.codectx/ai.local.yml
docs/.codectx/usage.yml

# Force-include checked-in config and lifetime metrics
!docs/.codectx/ai.yml
!docs/.codectx/preferences.yml
!docs/.codectx/global_usage.yml
```

Compiled output, installed packages, history, and local secrets are gitignored. Configuration files and project lifetime usage metrics are checked in.

---

[Back to overview](README.md)
