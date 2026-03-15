# Search and Retrieval

codectx uses a four-layer query pipeline that transforms a raw search query into a precisely ranked set of documentation chunks. Each layer builds on the previous one, and every scoring decision is deterministic and traceable.

---

## The Query Pipeline

```
Raw query string
       |
       v
[1. Taxonomy Query Expansion]     Broaden query using aliases and term relationships
       |
       v
[2. BM25F Field-Weighted Scoring] Score heading, body, code, and terms fields separately
       |
       v
[3. Reciprocal Rank Fusion]       Merge ranked lists from all three indexes
       |
       v
[4. Graph-Based Re-ranking]       Boost chunks connected to high-scoring results
       |
       v
Final ranked result set
```

Each layer operates at query time — no changes to compiled indexes or chunk files are needed. The layers compose into a single pipeline that executes in milliseconds.

---

## Layer 1: Taxonomy Query Expansion

Before the query touches the search index, it gets expanded using the taxonomy built during compilation. A search for "auth" becomes a search that also includes "authentication," related terms like "jwt" and "oauth," and associated terms like "authorization."

### Expansion Tiers

Expanded terms receive decreasing weight based on their relationship to the original query:

| Tier | Source | Weight | Example for "auth" |
|------|--------|--------|--------------------|
| Original | Query as typed | 1.0 | "auth" |
| Aliases | Taxonomy aliases | 1.0 | "authentication," "login," "sign-in," "authn" |
| Narrower | Child terms | 0.7 | "jwt," "oauth," "session-auth," "api-keys" |
| Related | Lateral relationships | 0.4 | "authorization," "middleware" |

Broader terms (parent concepts) are intentionally excluded. A query for "JWT" should not pull in everything under the parent "Authentication" concept.

### Why Expansion Before Scoring

Expansion happens before BM25F scoring, not after. The expanded terms are passed into the scorer as a weighted query vector. This is the correct approach — BM25's IDF calculation needs to apply to each expanded term independently, not to a pre-combined query string.

### Configuration

```yaml
# In preferences.yml
query:
  expansion:
    enabled: true
    alias_weight: 1.0
    narrower_weight: 0.7
    related_weight: 0.4
    max_expansion_terms: 20    # Cap to prevent query explosion on broad terms
```

---

## Layer 2: BM25F Field-Weighted Scoring

Standard BM25 treats each chunk as a single document. BM25F (the F stands for Fields) computes separate scores for distinct fields within each chunk, then combines them with configurable weights.

### Fields

Each chunk is split into four fields at index time:

| Field | Source | Default Weight | Rationale |
|-------|--------|---------------|-----------|
| Heading | Heading hierarchy from context header | 3.0 | Headings are the author's signal about what a chunk *is* |
| Terms | Taxonomy-assigned terms | 2.0 | Curated signals, not raw text |
| Body | Prose paragraphs | 1.0 | Baseline field |
| Code | Content inside fenced code blocks | 0.6 | Code identifiers matter but tokenize differently |

A term match in a heading counts three times more than a match in body prose. This reflects the structural importance of headings as topic indicators.

### Per-Field Length Normalization

Each field has its own length normalization parameter (`b`):

- **Headings** (`b: 0.3`) — Low normalization. Headings are intentionally short; don't penalize brevity.
- **Terms** (`b: 0.0`) — No normalization. The terms list is curated, not prose.
- **Body** (`b: 0.75`) — Standard BM25 normalization for prose.
- **Code** (`b: 0.5`) — Moderate. Code blocks vary widely in size.

### Domain-Aware Tokenization

The BM25 tokenizer is designed for technical documentation:

- **Compound terms preserved**: "error-handling" stays whole, not split on the hyphen
- **Code identifiers preserved**: `CreateUser` stays whole
- **Technical stopwords kept**: "null," "void," "async," "err" are meaningful in documentation
- **Standard stopwords removed**: "the," "a," "is," "are" — these carry no signal
- **Snowball stemming**: Applied to improve recall across word forms

### Configuration

```yaml
# In preferences.yml
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
```

---

## Layer 3: Reciprocal Rank Fusion

BM25F runs independently on three separate indexes — objects (instructions), specs (reasoning), and system (compiler docs). Each index produces its own ranked list scored within its own corpus.

Reciprocal Rank Fusion (RRF) merges these lists into one without requiring scores to be on the same scale. For each chunk that appears in any list:

```
RRF(chunk) = Sum of: weight_i / (k + rank_in_list_i)
             for each list where the chunk appears
```

A chunk ranked 1st in the objects index and 5th in the specs index scores higher than a chunk ranked 3rd in objects only. Presence across multiple indexes is itself a relevance signal.

### Index Weights

Instruction chunks are the primary target. Reasoning and system chunks are supporting context:

| Index | Weight | Rationale |
|-------|--------|-----------|
| Objects | 1.0 | Instruction content is what the AI needs most |
| Specs | 0.7 | Reasoning provides high-value context |
| System | 0.3 | Compiler docs are lower priority for most queries |

### Unified Output

The output of RRF is a single ranked list where object, spec, and system chunks appear interleaved based on their fused scores. The AI receives one coherent result set rather than three separate lists.

### Configuration

```yaml
# In preferences.yml
query:
  rrf:
    k: 60
    index_weights:
      objects: 1.0
      specs: 0.7
      system: 0.3
```

---

## Layer 4: Graph-Based Re-ranking

After RRF produces the merged list, a second pass re-ranks the top results using relationship signals from the compiled manifests. Three graph signals apply boost multipliers:

| Signal | Source | Boost | Rationale |
|--------|--------|-------|-----------|
| Adjacent chunk scored | manifest.yml adjacency links | +15% | If chunk 3 is relevant, chunk 4 probably is too |
| Spec chunk scored | manifest.yml spec cross-references | +20% | If both instruction and reasoning scored, the topic is deeply relevant |
| Cross-referenced doc scored | metadata.yml document references | +10% | Cross-referenced documents share topic territory |

### How It Works

The re-ranker builds a set of chunk IDs that scored in the top results, then checks each result's graph connections:

- If a chunk's adjacent neighbor (previous or next in the same file) is also in the scored set, the chunk gets a 15% boost.
- If a chunk's paired spec or parent instruction chunk is also in the scored set, both get a 20% boost.
- If a chunk's source document is cross-referenced by another document that also has scored chunks, the chunk gets a 10% boost.

The maximum combined boost when all three signals fire is +45%.

### Re-ranking Window

The re-ranker considers a window larger than the final output size. At the default `results_count` of 30, the window is approximately 65 candidates (derived as `ceil(results_count * 2.15)`). This gives the re-ranker room to promote graph-connected chunks from outside the initial top-N into the final results.

### Configuration

```yaml
# In preferences.yml
query:
  graph_rerank:
    enabled: true
    adjacent_boost: 0.15
    spec_boost: 0.20
    cross_ref_boost: 0.10
```

---

## Query Output

The final output is a unified ranked list with full metadata. Object, spec, and system chunks appear interleaved based on their fused RRF scores:

```
-> Results for: "jwt refresh token validation"
  Expanded: jwt json-web-token bearer-token refresh-token token-validation

Results (5, bm25f + rrf)
  1. [score: 0.0234] obj:a1b2c3.03 — Authentication > JWT Tokens > Refresh Flow
     Source: docs/topics/authentication/jwt-tokens.md (chunk 3/7, 462 tokens)
     Indexes: objects:#1, specs:#3

  2. [score: 0.0198] obj:a1b2c3.04 — Authentication > JWT Tokens > Validation Rules
     Source: docs/topics/authentication/jwt-tokens.md (chunk 4/7, 488 tokens)
     Indexes: objects:#2

  3. [score: 0.0165] spec:f7g8h9.02 — Authentication > JWT Tokens > Refresh Flow
     Source: docs/topics/authentication/jwt-tokens.spec.md (chunk 2/3, 380 tokens)
     Indexes: specs:#1, objects:#5

  4. [score: 0.0142] obj:a1b2c3.05 — Authentication > JWT Tokens > Error Handling
     Source: docs/topics/authentication/jwt-tokens.md (chunk 5/7, 412 tokens)
     Indexes: objects:#4

  5. [score: 0.0098] sys:m3n4o5.01 — Documentation Protocol > Mandatory Workflow
     Source: system/foundation/documentation-protocol/README.md (chunk 1/10, 107 tokens)
     Indexes: system:#2

  Total: 1,849 tokens across 5 results

Related chunks (adjacent to top results, not scored):
  obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure (488 tokens)

Run "codectx generate" with the top chunk IDs above to read their full content.
Try additional queries with different terms to explore related areas before deciding.
```

Each result includes:
- The chunk ID (with type prefix — `obj:`, `spec:`, or `sys:`)
- Fused RRF score
- Heading hierarchy (using em dash `—` separator)
- Source file and chunk position
- Token count (for budget-aware decisions)
- `Indexes:` metadata showing which indexes contributed and at what rank

The header shows the result count and the active scoring pipeline (`bm25f + rrf`). The total token count across all results helps the AI plan its budget. The footer guides the AI toward the next steps — generating a document from the results or running additional queries.

The "Related chunks" section lists chunks that are adjacent to top-scoring results but didn't score highly enough to appear in the ranked list. These are candidates for the AI to request if it needs more context.

---

## Why BM25 Over Vector Search

Documentation uses consistent terminology. Exact keyword matching with taxonomy-powered synonym expansion is the right approach for instructional content:

- **Deterministic**: Every scoring decision is traceable. You can explain exactly why a chunk ranked where it did.
- **No infrastructure**: No embedding model, no vector database, no GPU. Indexes are files on disk, queries run in-memory.
- **Structural preservation**: BM25F uses the structural signals already in the compiled chunks — headings, taxonomy terms, adjacency. Vector embeddings discard this structure.
- **Debuggable**: When results are wrong, you can inspect field weights, IDF values, and taxonomy mappings. Vector search failures require understanding distances in high-dimensional space.

The taxonomy addresses BM25's one limitation — inability to match semantically similar but lexicographically different terms. "Login flow" won't match documentation titled "Authentication Sequence" in raw BM25. With taxonomy expansion, it does.

---

[Back to overview](README.md)
