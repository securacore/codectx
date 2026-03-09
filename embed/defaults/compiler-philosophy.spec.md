# Compiler Philosophy Reasoning

The "documentation as source code" metaphor is the central design decision.
It means documentation goes through a build pipeline with deterministic,
reproducible output — just like compiling a program.

Token counting as a primitive exists because AI context windows are the
fundamental constraint. Words and characters are poor proxies — a 50-word
code block can be 200 tokens while 50 words of prose might be 65 tokens.
Every budget decision must use the unit the constraint is measured in.

Determinism over probabilistic retrieval was chosen because vector RAG
achieves correct chunk retrieval below 60% of the time in practice. BM25
with taxonomy-augmented queries provides explainable, traceable retrieval
where every ranking decision can be inspected and debugged.

Incremental compilation is essential for developer experience. A project
with hundreds of documentation files should recompile in seconds when only
a few files change, not minutes for a full rebuild every time.
