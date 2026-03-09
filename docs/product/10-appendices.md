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
