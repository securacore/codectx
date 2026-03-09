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

