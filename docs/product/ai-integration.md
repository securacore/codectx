# AI Integration

codectx bridges your compiled documentation to AI coding tools. The `codectx link` command creates entry point files that AI tools discover and load automatically.

## Supported Tools

| Tool | Entry Point File | How it works |
|---|---|---|
| Claude Code | `CLAUDE.md` | Claude reads this file at the start of every session |
| Cursor | `AGENTS.md` | Cursor loads agent instructions from this file |
| GitHub Copilot | `AGENTS.md` | Copilot reads repository-level agent instructions |
| OpenCode | `AGENTS.md` | OpenCode loads agent context from this file |

## How Linking Works

Running `codectx link` creates entry point files at your repository root. Each file contains a single line pointing to `.codectx/README.md`, which bootstraps the loading protocol.

If an entry point file already exists (for example, you already have a `CLAUDE.md` with custom instructions), `codectx link` renames it to `[file].[timestamp].bak` before creating the new one. Your original content is never lost.

Linking is a separate step from compilation. You typically run it once after initial setup, or again if you change the output directory.

## The Loading Protocol

When an AI tool starts a session, it follows this sequence:

1. **Entry point**: AI opens the entry point file (e.g., `CLAUDE.md`). It contains a single directive pointing to `.codectx/README.md`.
2. **README**: AI loads `.codectx/README.md`, which describes the loading protocol and links to `manifest.yml`.
3. **Data map**: AI loads `.codectx/manifest.yml` -- the compiled data map that indexes all available documentation.
4. **Foundation**: AI loads foundation documents marked `load: always`. These are the minimum required context for every session.
5. **On-demand loading**: As the task progresses, AI consults the data map to find and load relevant topics, prompts, or plans.
6. **Plan triage**: For plans, AI reads `plan.yml` first to assess status without loading the full plan document.

This protocol ensures AI never loads documentation blindly. The data map acts as a table of contents, and AI selectively loads only what the current task requires. Links within compiled documents reference other objects by their content-addressed filenames, so AI can navigate between related documents directly.

When CMDX compression is enabled, the compiled README includes a note explaining the `.cmdx` format to AI consumers. AI models process the `@TAG` syntax natively without requiring a decoding step.

## AI Provider Configuration

codectx can integrate with AI tools for assisted features. The provider is configured in preferences:

```bash
codectx set ai.provider=claude     # Set the AI provider
codectx set ai.model=llama3        # Set the model (Ollama only)
```

Known providers: `claude` (Claude Code), `opencode` (opencode), `ollama` (Ollama).

Provider validation checks two things:
1. The provider ID must be in the known registry
2. The provider's binary must be available on PATH

If either check fails, `codectx set` rejects the value with a descriptive error.

## Model Class Targeting

The `ai.class` preference defines the documentation compatibility baseline -- the minimum model capability tier that documentation is written for. This controls how the AI authoring foundation document adapts its guidance for writing documentation and prompts.

```bash
codectx set ai.class=gpt-4o-class          # Conservative baseline (default)
codectx set ai.class=claude-sonnet-class    # Strong reasoning models
codectx set ai.class=o1-class              # Frontier reasoning models
```

The model class is a **documentation target**, not the model being used. Setting `ai.class=gpt-4o-class` means your documentation is written to work reliably on mid-tier instruction-following models. Documentation written for this tier also works on all higher tiers — but it uses more explicit, redundant phrasing than would be necessary for frontier models.

Setting a higher class (e.g., `o1-class`) allows documentation to use denser language, assume stronger implicit reasoning, and skip some of the hand-holding required for mid-tier models.

Class validation checks that the ID is in the known registry. Unlike providers, no binary check is needed — classes are documentation targets, not executable tools.

## Watch Mode

`codectx watch` monitors your `docs/` directory and recompiles automatically whenever files change. It uses filesystem events (fsnotify) with debouncing to avoid redundant compilations during rapid edits. New directories created under the watched tree are automatically picked up.

Watch mode is useful during documentation authoring: save a file, and the compiled output updates within seconds. It also includes a polling heartbeat as a safety net for filesystem events that get missed.

## Related

- [Package Format](packages.md) -- how documentation is structured for AI consumption
- [Compilation](compilation.md) -- how documentation is compiled into the output format
- [Compression](compression.md) -- CMDX format that AI reads when compression is enabled
- [Preference Management](set-command.md) -- managing AI provider and model class preferences
- [Configuration](configuration.md) -- how to configure output directory and entry points
