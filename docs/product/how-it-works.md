# How It Works

codectx compiles documentation the same way a language compiler processes source code — through a multi-stage pipeline that transforms raw input into optimized output. The input is human-written markdown. The output is a set of indexed, token-counted, taxonomy-enriched artifacts that AI tools consume through the CLI.

---

## The Compilation Pipeline

The pipeline runs as `codectx compile` and consists of ten stages. Each stage transforms the output of the previous one. Only changed files are reprocessed on incremental builds.

```
Stage 1:  Parse & Validate       Source markdown -> ASTs, structural checks
Stage 2:  Strip & Normalize      Remove human-formatting overhead
Stage 3:  Chunk                  Split into token-counted semantic blocks
Stage 4:  Index                  Build BM25F search indexes
Stage 5:  Extract Taxonomy       Build controlled vocabulary from content
Stage 6:  Bridge Generation      Create continuity summaries between chunks
Stage 7:  Generate Manifests     Chunk metadata, relationships, content hashes
Stage 8:  Assemble Context       Compile always-loaded session document
Stage 9:  Link Entry Points      Generate CLAUDE.md, AGENTS.md, .cursorrules
Stage 10: Generate Heuristics    Compilation diagnostics and statistics
```

### Stage 1: Parse and Validate

Every markdown file under `foundation/`, `topics/`, `plans/`, `prompts/`, and `system/` is parsed into an abstract syntax tree (AST). The parser validates structural requirements — every topic directory has a `README.md`, every file has heading structure, no file exceeds the configured token limit.

Each file gets a content hash (SHA-256). On incremental builds, the compiler compares these hashes against the previous compilation. Only changed files continue through the expensive stages.

### Stage 2: Strip and Normalize

The AI doesn't need three blank lines between paragraphs, decorative horizontal rules, HTML comments, or redundant emphasis. This stage removes human-formatting overhead while preserving all structural elements — headings, code blocks, lists, tables, links, and cross-references.

This typically reduces content by 10-20% in token count with zero information loss. Every token saved in compiled output is a token the AI can use for actual task context.

### Stage 3: Chunk

Content is split into semantic chunks using a token-counted accumulation algorithm. Rather than splitting on headings (which produces wildly uneven sizes), the chunker accumulates semantic blocks — paragraphs, code blocks, lists, tables — until reaching a target token count (default: 450 tokens).

Key rules:
- **Atomic blocks are never split.** A code block, list, or table stays whole even if it exceeds the target size.
- **Headings always start new chunks.** This preserves the structural signal that headings carry.
- **Flexibility window.** If a chunk is at 80% of the target and the next block would push it over, the chunker breaks early to keep sizes consistent.

Consistent chunk sizes are critical for search quality. When all chunks are approximately the same length, BM25's document length normalization becomes nearly irrelevant — scoring becomes purely about term relevance. Every chunk competes on equal footing.

Each chunk receives:
- A **context header** with source file, heading hierarchy, sequence position, and token count
- A **content-based ID** (hash of the chunk content, prefixed by type)
- **Type routing** — instruction chunks (`obj:`) go to `compiled/objects/`, reasoning chunks (`spec:`) go to `compiled/specs/`, system chunks (`sys:`) go to `compiled/system/`

### Stage 4: Index

Three separate BM25F search indexes are built — one for each content type:

| Index | Content | Language Pattern |
|-------|---------|-----------------|
| Objects | Instruction chunks | Directive: "use," "implement," "always" |
| Specs | Reasoning chunks | Explanatory: "because," "the goal is," "we chose" |
| System | Compiler documentation | Meta: "compiler," "generate," "alias," "chunk" |

Separate indexes mean each type scores within its own corpus. An instruction chunk about "implementing authentication" doesn't compete with a system chunk about "generating authentication aliases." Each index has its own IDF calculations scoped to its content type.

The indexes use BM25F (field-weighted BM25) with separate scoring for heading, body, code, and taxonomy term fields. See [Search and Retrieval](search-and-retrieval.md) for the full scoring pipeline.

### Stage 5: Extract Taxonomy

The compiler builds a controlled vocabulary from the documentation content — like a symbol table in a traditional compiler. The extraction pipeline runs four passes:

1. **Structural extraction** — Headings, code identifiers, bold terms, and structural positions are harvested as high-confidence canonical terms. Pure parsing, no NLP.
2. **POS extraction** — Natural language processing via part-of-speech tagging extracts compound technical terms ("dependency injection," "middleware chain") and named entities (library names, framework names).
3. **Deduplication** — Terms from different passes are merged under canonical forms. "DB migrations" in body text merges under "Database Migrations" from a heading.
4. **Relationship inference** — Heading hierarchy yields parent-child relationships. Cross-references yield lateral relationships.

The result is a SKOS-inspired taxonomy tree with canonical terms, aliases, hierarchical relationships (broader/narrower), and lateral relationships (related). This taxonomy powers [query expansion](search-and-retrieval.md) — the AI searches for "auth" and the system automatically includes "authentication," "login," "sign-in," and related terms.

### Stage 6: Bridge Generation

For each pair of adjacent chunks from the same source file, the compiler generates a one-line bridge summary. The bridge captures what the previous chunk established that the next chunk assumes the reader already knows.

Bridges are generated deterministically using key phrase extraction and heading analysis. They enable the AI to understand continuity when reading non-adjacent chunks — if chunks 2 and 5 from the same file are assembled together, the bridge between them fills in the context gap.

### Stage 7: Generate Manifests

Three manifest files capture the compiled state:

- **manifest.yml** — Chunk navigation map. Every chunk's metadata: source file, heading hierarchy, sequence position, token count, taxonomy terms, adjacent chunk references, bridge summaries, and spec-to-instruction cross-references.
- **metadata.yml** — Document relationship graph. Cross-references between documents, with bidirectional `references_to` and `referenced_by` links.
- **hashes.yml** — Content hashes for every source file plus system instruction hashes. The foundation of incremental compilation.

### Stage 8: Assemble Context

Documents listed in `session.always_loaded` (configured in `codectx.yml`) are assembled into a single `context.md` file. This is the one document optimized for reading, not searching — not chunked, not indexed. The AI reads it top to bottom at session start.

The assembler respects the declared order and checks the total against the configured token budget. See [Session Context](session-context.md) for details.

### Stage 9: Link Entry Points

AI tool entry point files are generated at the repository root — `CLAUDE.md`, `AGENTS.md`, `.cursorrules`, and `copilot-instructions.md`. Each directs the AI to read `context.md` and use `codectx query`/`codectx generate` for documentation access.

See [AI Tool Integration](ai-tool-integration.md) for the entry point format and customization.

### Stage 10: Generate Heuristics

A diagnostic report (`heuristics.yml`) is regenerated on every compile. It captures:

- Source file counts (total, new, modified, unchanged)
- Chunk statistics (total, by type, average/min/max tokens, oversized count)
- Taxonomy statistics (term count, alias count, extraction source breakdown)
- Session context utilization (tokens used vs. budget)
- BM25 index statistics (indexed terms, indexed chunks per type)
- Incremental compilation details (which stages ran, which were skipped)
- Timing breakdown per stage

The AI can read this file to orient itself on the documentation landscape before querying.

---

## Incremental Compilation

Content hashing enables incremental compilation. If your project has 500 markdown files and you change 3, only those 3 files are reprocessed through the expensive stages. The hash comparison is O(n) on file count, but actual work scales with the number of changes.

System instruction hashes are tracked separately. If you edit the taxonomy generation instructions in `system/topics/taxonomy-generation/`, the entire taxonomy is regenerated on the next compile — regardless of whether documentation content changed. Each compiled artifact records the hash of the instructions that produced it, enabling targeted invalidation.

---

## What Gets Compiled Where

| Source | Compiled To | Prefix | BM25 Index |
|--------|-------------|--------|------------|
| `.md` files (non-system) | `compiled/objects/` | `obj:` | `bm25/objects/` |
| `.spec.md` files (all sources) | `compiled/specs/` | `spec:` | `bm25/specs/` |
| `system/**/*.md` (non-spec) | `compiled/system/` | `sys:` | `bm25/system/` |

The compiled output directory structure:

```
.codectx/compiled/
  context.md          # Assembled session context (not chunked)
  manifest.yml        # Chunk navigation map
  metadata.yml        # Document relationship graph
  taxonomy.yml        # Term tree with aliases
  hashes.yml          # Content hashes for incremental builds
  heuristics.yml      # Compilation diagnostics
  objects/            # Instruction chunks
  specs/              # Reasoning chunks
  system/             # Compiler documentation chunks
  bm25/
    objects/           # BM25F index over instructions
    specs/             # BM25F index over reasoning
    system/            # BM25F index over system docs
```

All compiled output is gitignored. It is fully reconstructable from source markdown and configuration.

---

[Back to overview](README.md)
