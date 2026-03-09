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

