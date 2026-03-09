## Phase 5: Taxonomy System

### Design Philosophy

The taxonomy is a compiled artifact derived from the source documentation. It is not manually maintained. It serves as a controlled vocabulary with aliases that bridges the gap between how developers phrase queries and how documentation uses terminology.

The taxonomy follows a SKOS-inspired data model (W3C Simple Knowledge Organization System):
- **prefLabel**: The canonical term the documentation uses
- **altLabel**: Aliases and synonyms for that term
- **broader/narrower**: Hierarchical relationships between terms

**Reference**: W3C SKOS — https://www.w3.org/2004/02/skos/
**Reference**: Wikipedia — Automatic Taxonomy Construction — https://en.wikipedia.org/wiki/Automatic_taxonomy_construction

### Transparent AI Instructions

The LLM alias generation pass is governed by `docs/system/topics/taxonomy-generation/README.md`. This file ships as a sensible default on `codectx init` and is fully editable by the user.

If the taxonomy isn't producing the right aliases, the fix is to improve these instructions and recompile. There are no override maps, no secondary alias configurations, no hidden alias sources. One mechanism, one place, one pattern.

When a package is published, it does NOT include system/ instructions. The consumer's local system/ instructions govern how all documentation — local and from packages — gets its taxonomy generated. One set of compilation rules per project.

### Extraction Pipeline

Covered in detail in Phase 3, Stages 5 and 6. Summary:

1. **Structural extraction** (pure parsing): headings, code identifiers, bold terms, structural positions
2. **Relationship inference** (structural analysis): heading hierarchy → parent/child, cross-references → lateral
3. **POS extraction** (lightweight NLP via `prose`): noun phrases, named entities, compound terms
4. **Deduplication**: merge, score by frequency, filter by threshold
5. **LLM alias generation**: batched by taxonomy branch, governed by system/ instructions

The taxonomy is built from all three content types — instruction (.md), reasoning (.spec.md), and system (system/**/*.md). Reasoning docs often use the same canonical terms in explanatory context, and system docs use terms in meta/tooling context. This breadth improves alias generation — the LLM sees terms used in instructional, explanatory, and tooling sentences, producing broader alias coverage.

---

