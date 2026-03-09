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

