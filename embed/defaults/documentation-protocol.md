# Documentation Protocol

This project uses codectx — a documentation compiler that transforms markdown into optimized, searchable chunks with BM25 ranking, taxonomy expansion, and bridge summaries. Raw documentation files are source code for this compiler. You MUST NOT read them directly. You MUST use the compiled output exclusively.

The compiled documentation is the authoritative starting point for all work. It establishes the project's patterns, conventions, constraints, and architectural decisions. Always start here. Use what you learn to inform your approach, then apply your engineering knowledge to accomplish the task.

## Mandatory Workflow

You MUST begin every task by consulting the documentation. There are no exceptions.

### 1. Query

```
codectx query "your search terms"
```

Returns ranked results across instruction chunks (how-to), reasoning chunks (why), and system chunks (compiler behavior). Each result shows a chunk ID, heading, token count, and relevance score.

ALWAYS query before starting any development task. Query to understand existing patterns, conventions, and constraints before writing or modifying code.

### 2. Generate

After EVERY query that returns results, you MUST run `codectx generate` on the top results. This is not optional — query and generate are an inseparable pair. Do not skip this step even when chunk headings appear unrelated to your task.

```
codectx generate "obj:id1,obj:id2,spec:id3"
```

Assembles requested chunks into a single document. Use chunk IDs from query results. The output includes heading hierarchy, bridge summaries at content gaps, and a list of related chunks not included.

Use `--file <path>` to write the document to a file instead of stdout.

Query results show only chunk headings and scores — not content. Headings are structural breadcrumbs, not summaries of what the chunk contains. You MUST read the actual generated content before making any determination about whether the documentation covers a topic. Judging by titles alone will cause you to miss relevant content — taxonomy expansion and bridge summaries surface related information that headings do not reveal.

### 3. Act

The generated output establishes the project's conventions. These are authoritative — when the documentation defines a pattern, follow it. Use the documented conventions as the foundation for your work, then apply your engineering knowledge to fill gaps and accomplish the task.

**Before answering, cite your source.** State which `codectx generate` output informed your approach, referencing the chunk IDs. If the documentation covers the topic, your work must align with it. If it does not, say what you queried and found, then proceed using your engineering judgment shaped by the project's documented patterns and philosophy.

**NEVER read raw documentation files.** Do not supplement by reading files from `docs/`, `foundation/`, `topics/`, `plans/`, `prompts/`, or `system/` directories. These are compiler source files — not meant for direct consumption.

### 4. Re-query

When your investigation reveals new terms, components, or patterns not covered by the initial query, run `codectx query` again with new search terms. Repeat until you have reviewed all relevant documentation.

**Before concluding a topic is not documented, try at least 3 queries with different terms and generate the top results from each.** Vary your search terms — use synonyms, related concepts, broader terms, and narrower terms. The taxonomy expansion will surface related content, but only if your query terms are close enough to match. A single failed query is not sufficient evidence that a topic is undocumented. Each query must be followed by `codectx generate` — running queries without generating is equivalent to skipping the documentation step entirely.

### 5. Validate

Before completing a task, query for the areas you changed to confirm your implementation aligns with documented conventions. If the documentation contradicts your approach, follow the documentation.

## Rules

- **MUST** use `codectx query` as the sole method for finding documentation. NEVER use grep, find, cat, or any file-reading tool on documentation directories.
- **MUST** use `codectx generate` to retrieve documentation content. NEVER read raw markdown files from the docs tree.
- **MUST** re-query when new terms or concepts emerge during a task.
- **MUST** validate changes against documentation before finalizing.
- **MUST** cite which generated chunks informed your approach when answering or acting.
- **NEVER** browse or read files under `docs/`, `foundation/`, `topics/`, `plans/`, `prompts/`, or `system/` directories directly. These are compiler source files — not meant for direct consumption.
- **NEVER** skip the query step. Even if you think you know the answer, query first. The compiled chunks contain taxonomy aliases, cross-references, and bridge context that raw files lack.
- **MUST** run `codectx generate` after every `codectx query` that returns results. This pair is atomic — never query without generating.
- **NEVER** skip the generate step. Query results are pointers, not content. You must generate and read the actual chunks before acting or answering. Do not pre-filter results by heading — generate the top results regardless of whether headings seem related to your task.
- When the documentation covers a topic, follow it — do not override documented conventions with general knowledge.
- When the documentation does not cover a topic, say what you queried and found, then proceed using your engineering judgment informed by the project's documented patterns.

## Why This Matters

Raw markdown files are unranked, unindexed, and lack the cross-referencing that compilation adds. Reading them directly is:

1. **Slow** — you must manually find and read multiple files instead of getting ranked results.
2. **Incomplete** — you miss taxonomy aliases that would surface related content.
3. **Wasteful** — you consume context window budget on full files when a focused chunk would suffice.
4. **Inaccurate** — you miss bridge summaries that explain relationships between content areas.

The compiled index exists specifically so you do not need to read raw files. Use it.

## History

If the user references earlier context, check `codectx history` first. See the history documentation for details.
