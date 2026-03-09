# codectx — Documentation Compiler for AI-Driven Development

## System Specification & Implementation Guide (v2)

This specification describes the complete architecture, compilation pipeline, CLI interface, and package manager for codectx — a tool that compiles human-written markdown documentation into an AI-optimized knowledge structure with BM25 keyword search, compiled taxonomy, and token-aware chunk management.

## Document Map

Read these files in order. Each phase builds on concepts from previous phases.

| File | Phase | Contents | Words |
|------|-------|----------|-------|
| [01-system-overview.md](01-system-overview.md) | Phase 1 | Problem statement, why not vector RAG, design principles | ~620 |
| [02-package-structure.md](02-package-structure.md) | Phase 2 | Directory layout, codectx.yaml schemas (project + published), ai.yaml, preferences.yaml, system/ defaults, .spec.md convention, session management, .gitignore | ~2,775 |
| [03-compilation-pipeline.md](03-compilation-pipeline.md) | Phase 3 | All 10 pipeline stages: parse, strip, chunk, BM25 index, taxonomy extraction, LLM augmentation, manifests, context assembly, entry point sync, heuristics. Includes chunking algorithm, token counting, chunk file formats (obj/spec/sys), manifest schema, metadata schema, hashes schema, heuristics schema, context.md format, CLAUDE.md template | ~4,120 |
| [04-bm25-search.md](04-bm25-search.md) | Phase 4 | How BM25 works, scoring formula, why normalized chunks help, query flow with taxonomy translation | ~650 |
| [05-taxonomy-system.md](05-taxonomy-system.md) | Phase 5 | SKOS-inspired design, transparent AI instructions, extraction pipeline summary | ~325 |
| [06-cli-interface.md](06-cli-interface.md) | Phase 6 | All CLI commands with output samples: init, compile, sync, query, generate, install, update, search, publish, session add/remove/list, plan status/resume. Generated file format with obj/spec/sys sections | ~1,500 |
| [07-plan-state-tracking.md](07-plan-state-tracking.md) | Phase 7 | plan.yaml schema with per-step queries/chunks, dependency hash drift detection, resumption flow (instant replay vs guided re-search), context audit trail | ~840 |
| [08-package-manager.md](08-package-manager.md) | Phase 8 | GitHub registry with codectx-[name] convention, publishing/consuming, active/inactive toggle, transitive dependencies, codectx.lock schema, flat resolution strategy | ~870 |
| [09-implementation-roadmap.md](09-implementation-roadmap.md) | Phase 9 | 14-step build order, Go packages summary, design decisions log (20 entries) | ~1,350 |
| [10-appendices.md](10-appendices.md) | Appendices | Token counting reference, SKOS data model reference, external references and links | ~465 |

## Key Go Packages

| Package | Purpose |
|---------|---------|
| `github.com/tiktoken-go/tokenizer` | Token counting (cl100k_base) |
| `github.com/crawlab-team/bm25` | BM25 search indexing |
| `github.com/jdkato/prose/v2` | POS tagging, named-entity extraction |

## Quick Reference: Compiled Output Structure

```
.codectx/compiled/
  context.md          # Session preamble (always-loaded docs)
  metadata.yaml       # Document relationship graph
  taxonomy.yaml       # Term tree with aliases (SKOS-inspired)
  manifest.yaml       # Chunk navigation map
  hashes.yaml         # Content hashes for incremental builds
  heuristics.yaml     # Compilation diagnostics
  objects/            # Instruction chunks (obj: prefix)
  specs/              # Reasoning chunks (spec: prefix)
  system/             # Compiler documentation chunks (sys: prefix)
  bm25/
    objects/           # BM25 index over instructions
    specs/             # BM25 index over reasoning
    system/            # BM25 index over system docs
```

## Quick Reference: CLI Commands

```
codectx init                              # Scaffold new documentation package
codectx compile [--incremental]           # Run compilation pipeline
codectx sync                              # Regenerate CLAUDE.md / AGENTS.md
codectx query "search terms" [--top N]    # Search documentation
codectx generate "obj:id,spec:id,sys:id"  # Assemble chunks into readable doc
codectx install                           # Install dependencies from codectx.yaml
codectx update                            # Re-resolve to latest versions + recompile
codectx search "query"                    # Find packages on GitHub
codectx publish                           # Tag and push to GitHub
codectx session add|remove|list           # Manage always-loaded session context
codectx plan status|resume [name]         # Plan state and context reconstruction
```
