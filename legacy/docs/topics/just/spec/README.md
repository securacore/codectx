# Just Specification

Spec for the Just command runner conventions. For the conventions themselves, see [README.md](../README.md).

## Purpose

The Just configuration has a non-obvious architecture: a four-layer dispatch system with provider abstraction, one-recipe-per-file granularity, and a coupling between recipe names and prompt directory names. Without documentation, an AI agent or engineer adding a recipe would not discover this structure from the `Justfile` alone and would likely add recipes in the wrong location or break the dispatch pattern.

## Decisions

- **Just as sole operational interface.** Repository operations must go through Just rather than invoking tools directly. Just decouples the developer and AI workflow from specific tool choices. If the package manager changes from Bun to something else, or the linter changes from Biome, the Just recipes absorb the change while the commands remain the same. This also creates a single place to audit and control what tools are invoked. Alternative considered: allowing direct tool invocation alongside Just (rejected; defeats the abstraction and creates inconsistent workflows that diverge over time).

- **One recipe per file.** Mirrors the one-export-per-module pattern from the TypeScript conventions. Each file has a single, unambiguous purpose. Diffs are granular. File names directly communicate what the file does. Alternative considered: grouping related recipes in one file (rejected; obscures individual recipe purpose and produces noisier diffs).

- **Four-layer architecture.** Root, AI dispatch, AI backends, and settings. This separates public interface from implementation, and implementation from provider-specific details. The dispatch layer exists so the developer interacts with a stable set of root commands regardless of which AI backend is active. Alternative considered: direct root-to-backend calls without dispatch (rejected; changing the active backend would require editing every root recipe instead of one variable).

- **Import for root, mod for everything else.** Root recipes are the primary interface. Using `import` surfaces them at the top level of `just -l` without a namespace prefix. AI modules use `mod` to namespace their recipes, preventing collisions between identically-named recipes across modules (e.g., `save` exists in root, ai, claude, and opencode). Alternative considered: `mod` for everything (rejected; forces users to type `just root build` instead of `just build`). Alternative considered: `import` for everything (rejected; creates recipe name collisions).

- **AI dispatch via variable.** The `ai_bin` variable in `settings.just` controls which backend receives AI commands. Changing the active AI provider is a single-line edit. Alternative considered: environment variable (rejected; dotenv is already loaded but the choice is a project-level default, not a per-session override). Alternative considered: command-line argument on each call (rejected; adds friction to every invocation).

- **Private AI recipes.** AI module recipes are implementation details. Marking them `[private]` keeps `just -l` output clean and communicates that developers interact through root commands. The namespaced modules (`just claude save`) remain accessible for debugging or direct use.

- **Devbox shell for all recipes.** `set shell := ["devbox", "run", "-q", "--"]` ensures every recipe runs in the devbox-managed environment with consistent tool versions. Alternative considered: relying on PATH (rejected; breaks when devbox tools are not globally installed). Alternative considered: prefixing individual commands with `devbox run` (rejected; repetitive and error-prone).

- **Recipe names match prompt directory names.** The `save` recipe invokes `docs/prompts/save/README.md`. This coupling is intentional: it creates a predictable mapping that both humans and AI can follow without looking up configuration. Alternative considered: a lookup table or mapping file (rejected; indirection without benefit when the convention is simple enough to be a rule).

- **Settings as a shared import.** Every `.mod.just` imports `settings.just` rather than declaring its own settings. This is DRY applied to configuration. A single file controls shell, dotenv, and the AI backend variable. Alternative considered: per-module settings (rejected; duplicates configuration and creates drift risk).

- **Just not managed by devbox.** Just is installed externally rather than through devbox. Just is the entry point that invokes devbox; managing it through devbox would create a circular dependency where the tool runner depends on the environment it is supposed to bootstrap.

## Dependencies

- `Justfile`: root entry point
- `bin/just/settings.just`: shared configuration
- `devbox.json`: environment definition
- [docs/foundation/philosophy](../../../foundation/philosophy/README.md): guiding principles (one logical unit per file, Configuration is Truth)
- [docs/topics/typescript/README.md](../../typescript/README.md): one-export-per-module pattern that this architecture mirrors
- [docs/foundation/specs](../../../foundation/specs/README.md): spec template this document follows

## Structure

- `README.md`: Just conventions (the actionable rules)
- `spec/README.md`: this file; reasoning and decisions behind the conventions
