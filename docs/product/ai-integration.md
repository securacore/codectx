# AI Integration

codectx bridges compiled documentation to AI coding tools. The `codectx link` command creates entry point files that AI tools discover automatically.

## Supported Tools

| Tool | Entry Point |
|---|---|
| Claude Code | `CLAUDE.md` |
| Cursor | `AGENTS.md` |
| GitHub Copilot | `AGENTS.md` |
| OpenCode | `AGENTS.md` |

## How Linking Works

`codectx link` creates entry point files at the repository root. Each file contains a single line referencing `.codectx/README.md`. Before creating a new entry point file, the command renames any existing file to `[file].[timestamp].bak` to preserve the original.

Linking is a separate command from compilation. It is run once after initial setup or when the output directory changes.

## The Loading Protocol

AI tools follow a predictable loading sequence:

1. **Entry point**: AI opens the entry point file (e.g., `CLAUDE.md`). It contains a single line pointing to `.codectx/README.md`.
2. **README**: AI loads `README.md`, which describes the loading protocol and links to `manifest.yml`.
3. **Data map**: AI loads `manifest.yml`. This is the compiled data map indexing all available documentation.
4. **Foundation**: AI loads foundation documents marked `load: always`. This is the minimal initialization context.
5. **On-demand loading**: As the task progresses, AI consults the data map to locate and load relevant topics, prompts, or plans.
6. **Plan triage**: For plans, AI reads `state.yml` first to assess status without loading the full plan.

This approach ensures AI never loads documentation blindly. The data map is the navigation layer that makes documentation consumption token-efficient.

## Watch Mode

`codectx watch` monitors your documentation source directory and recompiles automatically when files change. It uses filesystem events (fsnotify) with debouncing to avoid redundant compilations during rapid edits. New directories created under the watched tree are automatically monitored.

Watch mode is useful during documentation authoring: save a file, and the compiled output updates within seconds.

## Related

- [Package Format](packages.md) -- how documentation is structured for AI consumption
- [Compilation](compilation.md) -- how documentation is compiled into the output format
- [Configuration](configuration.md) -- how to configure output directory and entry points
