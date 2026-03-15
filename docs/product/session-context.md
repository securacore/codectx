# Session Context

Session context is the documentation the AI reads at the start of every session — before it does any work, before it runs any queries. It contains the foundational knowledge that should always be available: engineering principles, coding standards, architectural guidelines, and any other documentation critical enough to warrant permanent presence in the context window.

---

## How It Works

You declare which documents should always be loaded in `codectx.yml` under `session.always_loaded`. When you compile, these documents are assembled into a single `context.md` file that the AI reads top-to-bottom at session start.

```yaml
# In codectx.yml
session:
  always_loaded:
    - foundation/coding-standards
    - foundation/architecture-principles
    - foundation/error-handling
    - react-patterns@community/foundation/component-principles
    - company-standards@acme
  budget: 30000    # Maximum tokens for session context
```

The order matters — documents appear in `context.md` in the order listed. The compiler strips unnecessary formatting and normalizes content but does not chunk the session context. It's optimized for reading, not searching.

---

## Managing Session Context

### Adding Documents

```bash
# Add a local foundation document
codectx session add foundation/coding-standards

# Add a specific topic from an installed package
codectx session add react-patterns@community/foundation/component-principles

# Add an entire package's documentation
codectx session add company-standards@acme
```

Each `add` command reports the token cost of the added entry and the new total against the budget.

### Removing Documents

```bash
codectx session remove company-standards@acme
```

### Viewing Current Context

```bash
codectx session list
```

Output:

```
Always-loaded session context (28,450 / 30,000 tokens):

  foundation/coding-standards                              8,200 tokens
  foundation/architecture-principles                       6,100 tokens
  foundation/error-handling                                5,000 tokens
  react-patterns@community/foundation/component-principles 4,800 tokens
  company-standards@acme                                   4,350 tokens
```

### Direct Editing

The CLI commands are convenience wrappers that modify the `session.always_loaded` list in `codectx.yml`. Hand-editing the file produces the same result. The commands provide token cost feedback that hand-editing does not.

---

## Token Budget

The `session.budget` field sets the maximum token count for assembled session context. The default is 30,000 tokens.

If the assembled content exceeds the budget, the compiler emits a warning identifying which entries consume the most tokens. The compilation still succeeds — the budget is advisory, not enforced.

The budget exists because session context competes with task-specific content for space in the AI's context window. Every token in session context is a token the AI can't use for query results, generated documents, or task instructions. A well-chosen budget ensures session context provides foundational guidance without crowding out the documentation the AI needs for specific tasks.

---

## What Gets Assembled

The compiler resolves each `always_loaded` reference to source markdown:

| Reference Type | Example | Resolution |
|----------------|---------|------------|
| Local path | `foundation/coding-standards` | Resolves under `docs/` |
| Package path | `react-patterns@community/foundation/component-principles` | Resolves under `.codectx/packages/` |
| Bare package | `company-standards@acme` | Resolves all docs from that package |

Each resolved document is:
1. Stripped and normalized (same as compilation Stages 1-2)
2. Kept as flowing prose — not chunked
3. Assembled in the declared order
4. Given consistent heading levels (H2 for each document's title, H3+ for internal structure)

---

## The Compiled Context File

The output is `docs/.codectx/compiled/context.md`:

```markdown
# Project Engineering Context

> This document is automatically compiled from session context entries.
> Source: docs/codectx.yml session.always_loaded
> Token count: 28,450 / 30,000 budget
> Compiled: 2025-03-09T12:00:00Z

## Coding Standards

[Full content from foundation/coding-standards/README.md]

## Architecture Principles

[Full content from foundation/architecture-principles/README.md]

## Error Handling Philosophy

[Full content from foundation/error-handling/README.md]

## React Component Principles

[Full content from react-patterns@community]

## Company Engineering Standards

[Full content from company-standards@acme]
```

This is the one document in the system optimized for reading. Not chunked, not fragmented, not indexed. The AI reads it completely at session start, following the entry point directive in CLAUDE.md or AGENTS.md.

---

## Session Context vs. Query Results

Session context and query results serve different purposes:

| | Session Context | Query Results |
|-|-----------------|---------------|
| **When loaded** | Every session, automatically | On demand, per query |
| **What it contains** | Foundational guidance, standards, principles | Task-specific documentation |
| **How it's formatted** | Flowing prose, reading-optimized | Chunked, metadata-enriched |
| **Token cost** | Counted against budget (permanent) | Counted per request (temporary) |
| **Source** | `session.always_loaded` | `codectx query` results |

Foundation documents are the ideal candidates for session context — they're technology-agnostic guidance the AI needs regardless of what specific task it's working on. Topic-specific documentation is better served through queries, where the AI loads only what's relevant to the current task.

---

[Back to overview](README.md)
