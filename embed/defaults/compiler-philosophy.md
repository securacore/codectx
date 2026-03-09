# Compiler Philosophy

codectx treats documentation like source code — it gets compiled into
optimized artifacts. The compilation pipeline parses, strips, chunks,
indexes, and assembles documentation into a form that AI tools can
search and consume efficiently.

## Core Principles

- **Source of truth is always the raw markdown.** Compiled artifacts are
  derived and fully reconstructable. Never edit compiled output directly.

- **Token counting is a first-class primitive.** Every size-related decision
  (chunk boundaries, context budgets, foundation assembly) is measured in
  tokens, not words or characters.

- **Deterministic over probabilistic.** Every retrieval decision is traceable.
  BM25 scores are explainable. Taxonomy mappings are inspectable. No
  black-box embedding distances.

- **Incremental compilation.** Content hashes track changes. Only modified
  documents trigger recompilation. System instruction changes trigger
  targeted regeneration of affected artifacts.

- **Model-agnostic artifacts, model-aware presentation.** Compiled packages
  work with any AI model. Model-specific behavior is handled at the
  presentation layer through configuration.

- **Transparent AI instructions.** The instructions that govern how the
  compiler's AI behaves are themselves documentation files the user can
  read, understand, and modify.
