# codectx — BM25 Query Enhancement Plan

**Scope**: Stage 4 (indexing) and `codectx query` execution
**Goal**: Increase chunk relevance and correlation capability by stacking four complementary techniques on top of the existing BM25 foundation
**Principle**: Each technique is independently valuable and non-breaking. They compose into a single query pipeline.

---

## Overview: The Stacked Query Pipeline

Standard BM25 treats every chunk as a flat bag of words. This discards signal that already exists in the compiled artifacts — heading paths, taxonomy relationships, adjacency links, and spec cross-references.

The enhanced pipeline adds four layers:

```
Query string (raw)
        │
        ▼
[1. Taxonomy Query Expansion]   — broaden query using alias + hierarchy graph
        │
        ▼
[2. BM25F Field-Weighted Scoring] — score heading, body, and code fields separately
        │
        ▼
[3. Reciprocal Rank Fusion]     — merge ranked lists from all three indexes
        │
        ▼
[4. Graph-Based Re-ranking]     — boost chunks connected to high-scoring results
        │
        ▼
Final ranked result set → codectx query output
```

Each layer operates on the output of the previous one. No layer requires changes to the compiled chunk files or the BM25 index structure — they all operate at query time.

**Unified output**: Before this pipeline, `codectx query` returned results per-index or merged them naively. After RRF, the output is a single ranked list where object, spec, and system chunks appear interleaved based on their fused scores. This is a change to the query output format — the AI receives one coherent list rather than three separate result sets.

---

## Layer 1: Taxonomy Query Expansion

### What It Does

Before touching the BM25 index, expand the raw query terms using the taxonomy tree already built during compilation. A query for "auth" becomes a query that also includes "authentication", its narrower terms ("jwt", "oauth", "session-auth"), and related terms ("authorization", "middleware").

This improves **recall** — chunks that use different vocabulary for the same concept become reachable without the developer knowing which exact term the documentation uses.

### Expansion Strategy

Three expansion tiers with decreasing weight contribution:

| Tier           | Source                           | Weight Multiplier | Example for "auth"                         |
| -------------- | -------------------------------- | ----------------- | ------------------------------------------ |
| Aliases        | `taxonomy.yaml` altLabel entries | 1.0               | "authn", "authentication", "login"         |
| Narrower terms | `taxonomy.yaml` narrower[]       | 0.7               | "jwt", "oauth", "session-auth", "api-keys" |
| Related terms  | `taxonomy.yaml` related[]        | 0.4               | "authorization", "middleware"              |

Broader terms (parent concepts) are intentionally excluded — a query for "JWT" should not pull in everything under the parent "Authentication" concept.

### Implementation

```go
type TaxonomyTerm struct {
    Canonical string
    Aliases   []string
    Broader   string
    Narrower  []string
    Related   []string
    Chunks    []string
}

type ExpandedQuery struct {
    Original string
    Terms    []WeightedTerm
}

type WeightedTerm struct {
    Text   string
    Weight float64
    Tier   string // "original", "alias", "narrower", "related"
}

func ExpandQuery(raw string, taxonomy map[string]TaxonomyTerm, cfg ExpansionConfig) ExpandedQuery {
    tokens := tokenizeQuery(raw) // same stemming pipeline as index time
    expanded := ExpandedQuery{Original: raw}

    seen := map[string]bool{}

    for _, token := range tokens {
        // Always include original token at full weight
        if !seen[token] {
            expanded.Terms = append(expanded.Terms, WeightedTerm{token, 1.0, "original"})
            seen[token] = true
        }

        term, ok := taxonomy[token]
        if !ok {
            continue
        }

        // Tier 1: aliases at full weight
        for _, alias := range term.Aliases {
            norm := strings.ToLower(alias)
            if !seen[norm] {
                expanded.Terms = append(expanded.Terms, WeightedTerm{norm, cfg.AliasWeight, "alias"})
                seen[norm] = true
            }
        }

        // Tier 2: narrower terms at reduced weight
        for _, narrower := range term.Narrower {
            if child, ok := taxonomy[narrower]; ok {
                norm := strings.ToLower(child.Canonical)
                if !seen[norm] {
                    expanded.Terms = append(expanded.Terms, WeightedTerm{norm, cfg.NarrowerWeight, "narrower"})
                    seen[norm] = true
                }
            }
        }

        // Tier 3: related terms at lower weight
        for _, related := range term.Related {
            if rel, ok := taxonomy[related]; ok {
                norm := strings.ToLower(rel.Canonical)
                if !seen[norm] {
                    expanded.Terms = append(expanded.Terms, WeightedTerm{norm, cfg.RelatedWeight, "related"})
                    seen[norm] = true
                }
            }
        }
    }

    return expanded
}
```

### Configuration in preferences.yaml

```yaml
query:
  expansion:
    enabled: true
    alias_weight: 1.0 # Aliases are equivalent to original terms
    narrower_weight: 0.7 # Narrower terms are likely relevant
    related_weight: 0.4 # Related terms are possible context
    max_expansion_terms: 20 # Cap to prevent query explosion on broad terms
```

### Key Design Decision: Expansion Before Scoring

Expansion happens before BM25F scoring, not after. The expanded weighted terms are passed into the BM25F scorer as a weighted query vector. This is the correct approach — you want the BM25 IDF calculation to apply to each expanded term independently, not to a pre-combined query string.

---

## Layer 2: BM25F Field-Weighted Scoring

### What It Does

Standard BM25 sees each chunk as one document. BM25F (the F stands for Fields) computes separate BM25 scores for distinct fields within each chunk, then combines them with configurable weights. A term match in a heading counts more than a match in body prose, which counts differently than a match inside a code block.

The chunk context header already extracts the heading path. The only new work is splitting the chunk content into fields at index time and scoring them separately at query time.

### Fields

| Field     | Source                              | Default Weight | Rationale                                                   |
| --------- | ----------------------------------- | -------------- | ----------------------------------------------------------- |
| `heading` | `heading:` line from context header | 3.0            | Headings are the author's signal about what a chunk _is_    |
| `body`    | Prose paragraphs in chunk content   | 1.0            | Baseline field                                              |
| `code`    | Content inside fenced code blocks   | 0.6            | Code identifiers matter but tokenize differently from prose |
| `terms`   | `terms:` list from context header   | 2.0            | Taxonomy-assigned terms are curated signals                 |

### Chunk Parsing for Fields

At index time, parse each chunk into its fields:

````go
type IndexedChunk struct {
    ID      string
    Fields  map[string]string // field name → text content
    Tokens  int
    Source  string
    Type    string // "object", "spec", "system"
}

var codeBlockPattern = regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")

func ParseChunkFields(chunkContent string, meta ChunkMeta) IndexedChunk {
    // Extract code blocks first
    codeMatches := codeBlockPattern.FindAllString(chunkContent, -1)
    codeText := strings.Join(codeMatches, " ")

    // Remove code blocks from body
    bodyText := codeBlockPattern.ReplaceAllString(chunkContent, "")

    // Strip the context header comment block
    bodyText = stripContextHeader(bodyText)

    return IndexedChunk{
        ID: meta.ID,
        Fields: map[string]string{
            "heading": strings.Join(meta.HeadingPath, " "),
            "body":    strings.TrimSpace(bodyText),
            "code":    codeText,
            "terms":   strings.Join(meta.Terms, " "),
        },
        Tokens: meta.Tokens,
        Source: meta.Source,
        Type:   meta.Type,
    }
}
````

### BM25F Scoring

BM25F computes a pseudo-document by concatenating the term frequency contributions from each field, weighted and length-normalized independently per field:

```go
type BM25FConfig struct {
    K1            float64
    FieldWeights  map[string]float64
    FieldB        map[string]float64 // per-field length normalization (b parameter)
}

type FieldIndex struct {
    // Per-field: term → map[chunkID]termFrequency
    TermFreq    map[string]map[string]map[string]float64
    // Per-field: average field length across corpus
    AvgFieldLen map[string]float64
    // Per-field: chunkID → field length
    FieldLen    map[string]map[string]float64
    // Global: term → number of chunks containing term (for IDF)
    DocFreq     map[string]int
    TotalDocs   int
}

func (idx *FieldIndex) Score(chunkID string, query ExpandedQuery, cfg BM25FConfig) float64 {
    var totalScore float64

    for _, term := range query.Terms {
        idf := math.Log(1 + (float64(idx.TotalDocs)-float64(idx.DocFreq[term.Text])+0.5)/
            (float64(idx.DocFreq[term.Text])+0.5))

        var weightedTF float64
        for fieldName, fieldWeight := range cfg.FieldWeights {
            b := cfg.FieldB[fieldName]
            tf := idx.TermFreq[fieldName][term.Text][chunkID]
            avgLen := idx.AvgFieldLen[fieldName]
            docLen := idx.FieldLen[fieldName][chunkID]

            // BM25F per-field normalized TF
            normTF := tf / (1 - b + b*(docLen/avgLen))
            weightedTF += fieldWeight * normTF
        }

        // Apply BM25 saturation and taxonomy expansion weight
        bm25Score := idf * (weightedTF * (cfg.K1 + 1)) / (weightedTF + cfg.K1)
        totalScore += bm25Score * term.Weight // term.Weight from taxonomy expansion tier
    }

    return totalScore
}
```

### Configuration in preferences.yaml

```yaml
bm25f:
  k1: 1.2
  fields:
    heading:
      weight: 3.0
      b: 0.3 # Low length normalization — headings are intentionally short
    terms:
      weight: 2.0
      b: 0.0 # No length normalization — terms list is curated, not prose
    body:
      weight: 1.0
      b: 0.75 # Standard length normalization for prose
    code:
      weight: 0.6
      b: 0.5 # Moderate — code blocks vary widely in size
```

The per-field `b` parameter is important. Heading fields should not be penalized for being short (low b). Code blocks vary too much in length for standard normalization (medium b). Body prose uses standard BM25 normalization.

---

## Layer 3: Reciprocal Rank Fusion

### What It Does

The three BM25F indexes (objects, specs, system) score chunks within their own corpus. A chunk about "JWT authentication" in the objects index competes against other object chunks. The same query run against the specs index returns reasoning chunks that scored highly in _their_ corpus.

Reciprocal Rank Fusion (RRF) merges multiple ranked lists into one without requiring scores to be on the same scale. It's parameter-free beyond a single smoothing constant `k`, and it's robust to outlier scores.

### Algorithm

For each chunk that appears in any ranked list, its RRF score is:

```
RRF(chunk) = Σ  1 / (k + rank_in_list_i)
           for each list i where chunk appears
```

A chunk ranked 1st in the objects list and 5th in the specs list scores higher than a chunk ranked 3rd in objects only. Presence across multiple indexes is itself a relevance signal.

```go
const rrfK = 60 // Standard default; rarely needs tuning

type RankedChunk struct {
    ID    string
    Score float64
    Rank  int
    Type  string // which index it came from
}

type RRFResult struct {
    ID       string
    RRFScore float64
    // Track which indexes contributed and at what rank — useful for explain output
    Sources  map[string]int // indexName → rank
}

func ReciprocRankFusion(lists map[string][]RankedChunk, k float64) []RRFResult {
    scores := map[string]*RRFResult{}

    for indexName, ranked := range lists {
        for rank, chunk := range ranked {
            if _, ok := scores[chunk.ID]; !ok {
                scores[chunk.ID] = &RRFResult{
                    ID:      chunk.ID,
                    Sources: map[string]int{},
                }
            }
            scores[chunk.ID].RRFScore += 1.0 / (k + float64(rank+1))
            scores[chunk.ID].Sources[indexName] = rank + 1
        }
    }

    results := make([]RRFResult, 0, len(scores))
    for _, r := range scores {
        results = append(results, *r)
    }

    sort.Slice(results, func(i, j int) bool {
        return results[i].RRFScore > results[j].RRFScore
    })

    return results
}
```

### Index Weighting

Standard RRF treats all lists equally. For codectx, object chunks are the primary target — instruction content is what the AI needs most. Spec and system chunks are supporting context. Weighted RRF applies a multiplier per index before fusion:

```go
func WeightedRRF(lists map[string][]RankedChunk, weights map[string]float64, k float64) []RRFResult {
    scores := map[string]*RRFResult{}

    for indexName, ranked := range lists {
        w := weights[indexName]
        for rank, chunk := range ranked {
            if _, ok := scores[chunk.ID]; !ok {
                scores[chunk.ID] = &RRFResult{
                    ID:      chunk.ID,
                    Sources: map[string]int{},
                }
            }
            scores[chunk.ID].RRFScore += w * (1.0 / (k + float64(rank+1)))
            scores[chunk.ID].Sources[indexName] = rank + 1
        }
    }

    results := make([]RRFResult, 0, len(scores))
    for _, r := range scores {
        results = append(results, *r)
    }
    sort.Slice(results, func(i, j int) bool {
        return results[i].RRFScore > results[j].RRFScore
    })
    return results
}
```

### Configuration in preferences.yaml

```yaml
query:
  rrf:
    k: 60
    index_weights:
      objects: 1.0 # Instruction chunks are the primary target
      specs: 0.7 # Reasoning chunks are high-value context
      system: 0.3 # Compiler docs are lower priority for most queries
```

---

## Layer 4: Graph-Based Re-ranking

### What It Does

After RRF produces a merged ranked list, re-rank the top results using the relationship graph in `manifest.yaml` and `metadata.yaml`. This is a second pass over the top-N candidates — it does not change the index or the BM25 scoring.

Three graph signals apply a boost multiplier to the RRF score:

| Signal                          | Source                                        | Boost | Rationale                                                                                          |
| ------------------------------- | --------------------------------------------- | ----- | -------------------------------------------------------------------------------------------------- |
| Adjacent chunk also scored      | `manifest.yaml` adjacent.previous/next        | +15%  | Adjacency is a strong coherence signal — if chunk 3 of a file is relevant, chunk 4 probably is too |
| Spec chunk also scored          | `manifest.yaml` spec_chunk                    | +20%  | If both the instruction and its reasoning scored, the topic is deeply relevant                     |
| Document cross-reference scored | `metadata.yaml` references_to / referenced_by | +10%  | Cross-referenced documents share topic territory                                                   |

### Implementation

```go
type GraphBoostConfig struct {
    AdjacentBoost    float64
    SpecBoost        float64
    CrossRefBoost    float64
    WindowSize       int // How many top results to consider "scored" for adjacency
}

type ManifestEntry struct {
    ID       string
    Adjacent struct {
        Previous string
        Next     string
    }
    SpecChunk string
    Source    string
    Terms     []string
}

func GraphRerank(
    results []RRFResult,
    manifest map[string]ManifestEntry,
    metadata map[string]DocumentMeta,
    cfg GraphBoostConfig,
) []RRFResult {
    // Build a set of chunk IDs that scored in the top window
    scoredIDs := map[string]bool{}
    for i, r := range results {
        if i >= cfg.WindowSize {
            break
        }
        scoredIDs[r.ID] = true
    }

    // Build a set of source documents that appear in scored results
    scoredDocs := map[string]bool{}
    for id := range scoredIDs {
        if entry, ok := manifest[id]; ok {
            scoredDocs[entry.Source] = true
        }
    }

    // Apply boosts
    boosted := make([]RRFResult, len(results))
    copy(boosted, results)

    for i := range boosted {
        entry, ok := manifest[boosted[i].ID]
        if !ok {
            continue
        }

        multiplier := 1.0

        // Adjacent chunk boost
        if scoredIDs[entry.Adjacent.Previous] || scoredIDs[entry.Adjacent.Next] {
            multiplier += cfg.AdjacentBoost
        }

        // Spec chunk boost — if this chunk's spec also scored (or vice versa)
        if entry.SpecChunk != "" && scoredIDs[entry.SpecChunk] {
            multiplier += cfg.SpecBoost
        }
        // Reverse: if this is a spec chunk and its parent object scored
        // (requires manifest lookup of the parent_object field)
        if parentID := specParentID(boosted[i].ID, manifest); parentID != "" && scoredIDs[parentID] {
            multiplier += cfg.SpecBoost
        }

        // Cross-reference boost — if a document this chunk's source references also scored
        if docMeta, ok := metadata[entry.Source]; ok {
            for _, ref := range docMeta.ReferencesTo {
                if scoredDocs[ref.Path] {
                    multiplier += cfg.CrossRefBoost
                    break // one cross-ref boost per chunk is enough
                }
            }
            for _, ref := range docMeta.ReferencedBy {
                if scoredDocs[ref.Path] {
                    multiplier += cfg.CrossRefBoost
                    break
                }
            }
        }

        boosted[i].RRFScore *= multiplier
    }

    // Re-sort after boosting
    sort.Slice(boosted, func(i, j int) bool {
        return boosted[i].RRFScore > boosted[j].RRFScore
    })

    return boosted
}

func specParentID(chunkID string, manifest map[string]ManifestEntry) string {
    // spec chunks carry a parent_object reference — look it up
    // This requires the manifest to also index spec entries
    // Implementation depends on how manifest is structured in memory
    return ""
}
```

### Configuration in preferences.yaml

```yaml
query:
  graph_rerank:
    enabled: true
    adjacent_boost: 0.15 # 15% boost if an adjacent chunk also scored
    spec_boost: 0.20 # 20% boost if the paired spec/object chunk also scored
    cross_ref_boost: 0.10 # 10% boost if a cross-referenced document also scored
```

`window_size` is intentionally absent from configuration. It is always derived in code as `2 × results_count`. This keeps the two values in sync automatically — if `results_count` is changed, the re-ranking window scales proportionally without requiring a second config edit. Storing them independently would allow them to drift out of relationship.

---

## Full Pipeline Integration

### Execution Sequencing

Layers 1, 3, and 4 are inherently sequential — each depends on the complete output of the previous layer. Layer 2 is the only embarrassingly parallel stage, which maps directly onto the existing goroutine pattern for parallel index queries. The expanded query from Layer 1 is read-only by the time goroutines fire, so no locks or synchronization are needed beyond the channel collect.

```
Expansion (sequential — produces shared read-only input)
        │
        ├── goroutine: objects BM25F ──┐
        ├── goroutine: specs BM25F ────┤ channel collect
        └── goroutine: system BM25F ───┘
                                       │
                              RRF (sequential — needs all three lists)
                                       │
                              Graph re-rank (sequential — needs fused list)
```

### Query Execution Flow

```go
// windowRatio defines how many candidates the graph re-ranker considers
// relative to the output size. 2.15 = one full output list of headroom
// plus a cherry-picking buffer for graph-promoted candidates.
// Minimum meaningful value is 2.0. Values above 2.5 produce diminishing
// returns as RRF score decay makes tail promotion unlikely without
// correspondingly larger boost multipliers.
const windowRatio = 2.15

type QueryConfig struct {
    Expansion   ExpansionConfig
    BM25F       BM25FConfig
    RRF         RRFConfig
    GraphRerank GraphBoostConfig
    TopN        int
}

func ExecuteQuery(raw string, indexes map[string]*FieldIndex, taxonomy map[string]TaxonomyTerm,
    manifest map[string]ManifestEntry, metadata map[string]DocumentMeta,
    cfg QueryConfig) []RRFResult {

    // Layer 1: Taxonomy expansion — must complete before goroutines fire.
    // Produces a read-only ExpandedQuery consumed concurrently by all three scorers.
    expanded := ExpandQuery(raw, taxonomy, cfg.Expansion)

    // Layer 2: Parallel BM25F scoring across all three indexes.
    // expanded is read-only — no locks needed.
    type indexResult struct {
        name   string
        ranked []RankedChunk
    }

    resultsCh := make(chan indexResult, len(indexes))

    for indexName, index := range indexes {
        go func(name string, idx *FieldIndex) {
            var ranked []RankedChunk
            for chunkID := range idx.AllChunkIDs() {
                score := idx.Score(chunkID, expanded, cfg.BM25F)
                if score > 0 {
                    ranked = append(ranked, RankedChunk{ID: chunkID, Score: score, Type: name})
                }
            }
            sort.Slice(ranked, func(i, j int) bool {
                return ranked[i].Score > ranked[j].Score
            })
            // Pass top candidates to RRF — no need to fuse zero-scorers
            if len(ranked) > cfg.TopN*3 {
                ranked = ranked[:cfg.TopN*3]
            }
            resultsCh <- indexResult{name, ranked}
        }(indexName, index)
    }

    // Collect — block until all three goroutines complete before proceeding to RRF
    lists := make(map[string][]RankedChunk, len(indexes))
    for range indexes {
        r := <-resultsCh
        lists[r.name] = r.ranked
    }

    // Layer 3: Reciprocal Rank Fusion — sequential, depends on all three lists
    fused := WeightedRRF(lists, cfg.RRF.IndexWeights, cfg.RRF.K)

    // Layer 4: Graph-based re-ranking — sequential, depends on fused list.
    // Window size is always derived as ceil(TopN * windowRatio) — never configured
    // independently. At results_count=30 this produces a window of 65.
    windowSize := int(math.Ceil(float64(cfg.TopN) * windowRatio))
    reranked := GraphRerank(fused, manifest, metadata, windowSize, cfg.GraphRerank)

    if len(reranked) > cfg.TopN {
        reranked = reranked[:cfg.TopN]
    }
    return reranked
}
```

### Updated ai.yaml

```yaml
consumption:
  model: "claude-sonnet-4-20250514"
  context_window: 200000
  results_count:
    30 # Unified RRF list — all three indexes fuse into one ranked output.
    # Increased from 10 to give RRF index weights and graph re-ranking
    # room to work across object, spec, and system chunks.
    # Graph re-ranking window is derived automatically as
    # ceil(results_count × 2.15) = 65 at this default.
    # Can be overridden per-call with: codectx query --top N "..."
```

### Updated preferences.yaml (Full Query Block)

```yaml
# BM25F field scoring (replaces flat bm25 block)
bm25f:
  k1: 1.2
  fields:
    heading:
      weight: 3.0
      b: 0.3 # Low — headings are intentionally short, don't penalize brevity
    terms:
      weight: 2.0
      b: 0.0 # None — curated term list, not prose
    body:
      weight: 1.0
      b: 0.75 # Standard BM25 length normalization for prose
    code:
      weight: 0.6
      b: 0.5 # Moderate — code blocks vary widely in size

# Query pipeline configuration
query:
  expansion:
    enabled: true
    alias_weight: 1.0 # Aliases are equivalent to original terms
    narrower_weight: 0.7 # Narrower terms are likely relevant
    related_weight: 0.4 # Related terms are possible context
    max_expansion_terms: 20 # Cap to prevent query explosion on broad terms

  rrf:
    k: 60
    index_weights:
      objects: 1.0 # Instruction chunks are the primary target
      specs: 0.7 # Reasoning chunks are high-value context
      system: 0.3 # Compiler docs are lower priority for most queries

  graph_rerank:
    enabled: true
    # window_size is derived in code as ceil(results_count × 2.15), never configured
    # directly. At results_count=30 the window is 65: one full output list of headroom
    # plus a cherry-picking buffer for graph-promoted candidates.
    adjacent_boost: 0.15 # +15% if an adjacent chunk in the same file also scored
    spec_boost: 0.20 # +20% if the paired spec/object counterpart also scored
    cross_ref_boost: 0.10 # +10% if a cross-referenced document also has scored chunks
    # Maximum combined boost: +45% (all three signals firing simultaneously)
```

---

## Impact on Existing Data Structures

### manifest.yaml — No Changes Required

All signals used by graph re-ranking (`adjacent`, `spec_chunk`, `terms`) are already present in the manifest schema. No new fields needed.

### metadata.yaml — No Changes Required

`references_to` and `referenced_by` are already in the schema. Graph re-ranking reads them as-is.

### taxonomy.yaml — No Changes Required

`aliases`, `narrower`, and `related` are already in the schema. Query expansion reads them directly.

### BM25 Index (compiled/bm25/) — Structural Change

The existing flat BM25 index must be replaced with a per-field index. This is a **compile-time change** to Stage 4. The query interface remains the same from the caller's perspective, but the serialized index format changes to store per-field term frequencies and field lengths.

The three separate indexes (objects, specs, system) are preserved — BM25F applies _within_ each index, and RRF fuses _across_ them.

---

## Implementation Order

The four layers have a natural build sequence. Each is independently valuable and can ship without the others.

**Step 1 — BM25F** replaces the existing flat scorer. This is the highest-impact change — all subsequent layers build on it. Requires updating the Stage 4 indexer and the query scorer.

**Step 2 — Taxonomy Query Expansion** adds recall with no index changes. Reads the already-compiled taxonomy.yaml at query time. Can ship immediately after BM25F without waiting for RRF or graph work.

**Step 3 — RRF** fuses the three existing indexes into one ranked output. Currently the query likely hits each index independently and returns separate result sets. RRF merges them cleanly. Requires changes to the `codectx query` command output assembly.

**Step 4 — Graph Re-ranking** is the final pass. Reads manifest.yaml and metadata.yaml which are already compiled. Can be added to the query pipeline after RRF without touching the indexer.

---

## Go Package Summary

| Purpose                               | Package / Approach                                             |
| ------------------------------------- | -------------------------------------------------------------- |
| BM25F field scoring                   | Implement directly — no library provides per-field BM25F in Go |
| Stemming (shared with index pipeline) | `github.com/kljensen/snowball`                                 |
| RRF fusion                            | Implement directly — trivial algorithm, no library needed      |
| Graph traversal                       | In-memory map traversal over loaded manifest/metadata          |
| Taxonomy expansion                    | In-memory map traversal over loaded taxonomy.yaml              |

The BM25F implementation is the only meaningful build — RRF and graph re-ranking are both simple algorithms over data structures already in memory. Taxonomy expansion is a map lookup loop.
