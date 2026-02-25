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
6. **Plan triage**: For plans, AI reads `state.yml` first to assess status without loading the full plan document.

This protocol ensures AI never loads documentation blindly. The data map acts as a table of contents, and AI selectively loads only what the current task requires. Links within compiled documents reference other objects by their content-addressed filenames, so AI can navigate between related documents directly.

## Watch Mode

`codectx watch` monitors your `docs/` directory and recompiles automatically whenever files change. It uses filesystem events (fsnotify) with debouncing to avoid redundant compilations during rapid edits. New directories created under the watched tree are automatically picked up.

Watch mode is useful during documentation authoring: save a file, and the compiled output updates within seconds. It also includes a polling heartbeat as a safety net for filesystem events that get missed.

## Related

- [Package Format](packages.md) -- how documentation is structured for AI consumption
- [Compilation](compilation.md) -- how documentation is compiled into the output format
- [Configuration](configuration.md) -- how to configure output directory and entry points
