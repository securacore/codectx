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

