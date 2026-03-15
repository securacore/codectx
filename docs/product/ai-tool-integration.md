# AI Tool Integration

codectx generates entry point files that tell AI coding tools how to find and use your compiled documentation. These files direct the AI to read session context at startup and use `codectx query` and `codectx generate` for on-demand documentation access.

---

## Entry Point Files

Run `codectx link` to generate entry point files at your repository root:

```bash
codectx link
```

codectx supports four AI tool entry point formats:

| File | AI Tool |
|------|---------|
| `CLAUDE.md` | Claude Code |
| `AGENTS.md` | Generic AI agents |
| `.cursorrules` | Cursor |
| `.github/copilot-instructions.md` | GitHub Copilot |

Each entry point file serves the same purpose — bootstrap the AI into the codectx system:

1. Direct the AI to read `docs/.codectx/compiled/context.md` before any task
2. Instruct the AI to use `codectx query` for searching documentation
3. Instruct the AI to use `codectx generate` for assembling readable documents
4. Establish that compiled documentation overrides assumptions from training data

### CLAUDE.md Example

```markdown
# codectx

STOP. Read [context](docs/.codectx/compiled/context.md) now.
Do not proceed with any task until you have read that document
completely and followed every instruction it contains.
```

The entry point is deliberately minimal. Its only job is to bootstrap — the actual instructions live in `context.md`. This maintains single-source-of-truth: foundation doc updates only require recompilation, not manual entry point editing.

Entry points are regenerated automatically at the end of `codectx compile` and can be regenerated standalone with `codectx link`.

---

## How AI Tools Interact with codectx

### The Session Flow

```
1. AI session starts
2. AI reads entry point file (CLAUDE.md, AGENTS.md, etc.)
3. AI reads docs/.codectx/compiled/context.md (session context)
4. AI has foundational knowledge: coding standards, architecture, principles
5. Developer requests a task
6. AI runs: codectx query "relevant search terms"
7. AI receives ranked chunk results with scores and token counts
8. AI selects chunks based on relevance and token budget
9. AI runs: codectx generate "obj:id1,obj:id2,spec:id3"
10. AI reads the assembled document
11. AI proceeds with the task, informed by precise documentation
```

### Query Results

When the AI runs `codectx query`, it receives:

- Ranked results across instruction, reasoning, and system indexes
- Each result includes chunk ID, score, heading hierarchy, source file, and token count
- Related chunks adjacent to top results (for optional deeper context)
- The expanded query showing taxonomy-driven term expansion

The AI uses token counts to make budget-aware decisions — it knows exactly how many tokens each chunk will consume before requesting it.

### Generated Documents

When the AI runs `codectx generate`, it receives:

- A coherent reading document with content grouped by type
- Heading hierarchies restored for context
- Bridge summaries at non-adjacent chunk boundaries
- A footer listing related chunks not included (with token counts)

The AI can request additional related chunks in subsequent `codectx generate` calls if it needs more context.

### The Prompt Command

`codectx prompt` combines query and generate into one call — the AI searches and immediately gets an assembled document:

```bash
codectx prompt "authentication token validation"
```

This is the fastest path when the AI doesn't need to curate which chunks to include.

---

## Caller Context

codectx automatically detects which AI tool is calling it. This information is recorded in history entries and usage metrics, enabling per-tool tracking.

Detection uses environment variables set by the AI tools:

| Variable | Set By |
|----------|--------|
| `CLAUDE_CODE_ENTRYPOINT` | Claude Code |
| `CLAUDE_CODE_SESSION_ID` | Claude Code |
| `CURSOR_SESSION_ID` | Cursor |
| `ANTHROPIC_MODEL` | Anthropic API clients |

You can also set explicit context via codectx-defined variables:

| Variable | Purpose |
|----------|---------|
| `CODECTX_CALLER` | Override caller identification |
| `CODECTX_SESSION_ID` | Override session identification |
| `CODECTX_MODEL` | Override model identification |

---

## Model-Agnostic Design

Compiled artifacts are model-agnostic. The same compiled output works with Claude, GPT-4, Gemini, or any other model. Model-specific behavior is handled through configuration:

- `ai.yml` specifies the target consumption model and context window size
- `output_format` controls how generated documents are formatted (markdown, XML tags, or plain text)
- Token counting uses `cl100k_base` encoding by default, which is within ~10% variance for English prose across modern models

The AI tool entry points are the only model-specific layer — each file uses the conventions expected by its respective tool. The underlying documentation, compilation, and search system are completely model-independent.

---

## Multiple AI Tools

You can use codectx with multiple AI tools simultaneously. Each tool gets its own entry point file, but they all read the same compiled documentation and use the same CLI commands.

Usage metrics track invocations by caller, so you can see which tools consume the most tokens:

```bash
codectx usage
```

```
  By caller:
    claude: 601,450 tokens  (73.2%)
    cursor: 198,203 tokens  (24.1%)
    unknown: 22,250 tokens   (2.7%)
```

History entries also record caller context, enabling audit trails per tool.

---

[Back to overview](README.md)
