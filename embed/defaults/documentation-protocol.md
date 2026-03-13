# Documentation Protocol

This project uses codectx — a documentation compiler that transforms markdown into optimized, searchable chunks with BM25 ranking, taxonomy expansion, and bridge summaries. Raw documentation files are source code for this compiler. You MUST NOT read them directly. You MUST use the compiled output exclusively.

## Mandatory Workflow

You MUST follow this workflow for ALL documentation access. There are no exceptions.

### 1. Query

```
codectx query "your search terms"
```

Returns ranked results from three categories: instruction chunks (how-to), reasoning chunks (why), and system chunks (compiler behavior). Each result shows a chunk ID, heading, token count, and relevance score.

ALWAYS query before starting any development task. Query to understand existing patterns, conventions, and constraints before writing or modifying code.

### 2. Generate

```
codectx generate "obj:id1,obj:id2,spec:id3"
```

Assembles requested chunks into a single document. Use chunk IDs from query results. The output includes heading hierarchy, bridge summaries at content gaps, and a list of related chunks not included.

Use `--file <path>` to write the document to a file instead of stdout.

### 3. Act

Use ONLY the generated output to inform your work. NEVER supplement it by reading raw files from `docs/`, `foundation/`, `topics/`, `plans/`, `prompts/`, or `system/` directories.

### 4. Re-query

When your investigation reveals new terms, components, or patterns not covered by the initial query, run `codectx query` again with new search terms. Repeat until you have reviewed all relevant documentation.

### 5. Validate

Before completing a task, query for the areas you changed to confirm your implementation aligns with documented conventions. If the documentation contradicts your approach, follow the documentation.

## Rules

- **MUST** use `codectx query` as the sole method for finding documentation. NEVER use grep, find, cat, or any file-reading tool on documentation directories.
- **MUST** use `codectx generate` to retrieve documentation content. NEVER read raw markdown files from the docs tree.
- **MUST** re-query when new terms or concepts emerge during a task.
- **MUST** validate changes against documentation before finalizing.
- **NEVER** browse or read files under `docs/`, `foundation/`, `topics/`, `plans/`, `prompts/`, or `system/` directories directly. These are compiler source files — not meant for direct consumption.
- **NEVER** skip the query step. Even if you think you know the answer, query first. The compiled chunks contain taxonomy aliases, cross-references, and bridge context that raw files lack.

## Why This Matters

Raw markdown files are unranked, unindexed, and lack the cross-referencing that compilation adds. Reading them directly is:

1. **Slow** — you must manually find and read multiple files instead of getting ranked results.
2. **Incomplete** — you miss taxonomy aliases that would surface related content.
3. **Wasteful** — you consume context window budget on full files when a focused chunk would suffice.
4. **Inaccurate** — you miss bridge summaries that explain relationships between content areas.

The compiled index exists specifically so you do not need to read raw files. Use it.

## History

If the user references earlier context, check `codectx history` first. See the history documentation for details.
