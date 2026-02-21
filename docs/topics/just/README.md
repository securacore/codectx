# Just

Command runner conventions for this repository. This document covers the architecture, file organization, and patterns for Just recipes. It does not cover Just syntax; see the [Just manual](https://just.systems/man/en/) for language reference.

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Operational Directive

<rules>

- Just is the sole interface for all repository operations. Do not invoke package managers, build tools, dev servers, linters, or AI tools directly.
- Use the corresponding Just recipe for every task. If no recipe exists for a task, create one before proceeding.
- Just serves as an abstraction layer that decouples workflows from specific tools. The underlying package manager, runtime, or toolchain may change. The Just interface remains stable.

</rules>

## Architecture

The Just configuration is organized into four layers. Each layer has a distinct responsibility.

<rules>

- **Root module** (`bin/just/root/`): Public recipes the developer runs directly (`just build`, `just save`, `just dev`). Imported into the top-level `Justfile` via `import`, so recipes appear without a namespace prefix.
- **AI dispatch layer** (`bin/just/ai/`): Private recipes that route to the active AI backend. The `ai_bin` variable in `settings.just` determines which backend receives the call. Declared as a `mod` in the `Justfile`, so recipes are namespaced (`just ai save`).
- **AI backends** (`bin/just/claude/`, `bin/just/opencode/`): Provider-specific implementations. Each backend has a `prompt` recipe that knows how to invoke its AI tool against prompt definitions in `docs/prompts/`. Declared as `mod` in the `Justfile`, so recipes are namespaced (`just claude save`, `just opencode save`).
- **Settings** (`bin/just/settings.just`): Shared configuration imported by every module. Defines shell, dotenv loading, and the `ai_bin` variable.

</rules>

### Import vs. Mod

<rules>

- Use `import` for the root module. Root recipes appear at the top level of `just -l` with no namespace prefix.
- Use `mod` for all other modules. This namespaces their recipes under the module name and prevents collisions.

</rules>

### Command Flow

Root recipes delegate to the AI dispatch layer. The AI dispatch layer delegates to whichever backend `ai_bin` is set to. The backend executes the provider-specific implementation.

Example flow for `just save`:

1. `bin/just/root/save.just` calls `just ai save`
2. `bin/just/ai/save.just` calls `just {{ai_bin}} save`
3. `bin/just/claude/save.just` (or `opencode`) calls its `prompt` recipe with the appropriate prompt directory

## File Organization

<rules>

- One recipe per file. The file name matches the recipe name (`save.just` contains the `save` recipe).
- Every module directory has a `.mod.just` file as its entry point. This file imports `settings.just` and all recipe files in the directory.
- `settings.just` lives at `bin/just/settings.just`. Every `.mod.just` imports it.
- All Just files live under `bin/just/`. The only exception is the root `Justfile` at the project root.
- File names use lowercase kebab-case.

</rules>

## Recipe Conventions

<rules>

- Root recipes are public. They are the user-facing interface.
- AI module recipes are private. Mark them with `[private]`.
- Every recipe has a comment on the line above it describing what it does. This comment appears in `just -l` output.
- Every recipe has a `[group()]` annotation. The group name matches the module name (`root`, `ai`, `claude`, `opencode`).

</rules>

## Shell and Environment

<rules>

- All recipes execute through the devbox shell (`set shell := ["devbox", "run", "-q", "--"]` in `settings.just`).
- Environment variables load from `.env.development` via `set dotenv-filename`.
- Recipes that need bash scripting use a `#!/usr/bin/env bash` shebang with `set -euo pipefail`.

</rules>

## AI Provider Integration

<rules>

- The `ai_bin` variable in `settings.just` controls which AI backend is active. Valid values: `"claude"`, `"opencode"`.
- AI backends reference prompt definitions in `docs/prompts/`. The recipe name matches the prompt directory name (`save` invokes `docs/prompts/save/README.md`).
- Each backend has a `prompt` recipe that accepts a directory name and optional context string. This is the single point of integration between Just and the AI tool.

</rules>

### Adding a New AI Provider

1. Create a new directory under `bin/just/` named after the provider (e.g., `bin/just/newprovider/`).
2. Create a `.mod.just` that imports `settings.just` and the recipe files.
3. Create a `prompt.just` that invokes the provider's CLI with the prompt file from `docs/prompts/`.
4. Create recipe files (`save.just`, `docs-audit.just`, etc.) that call the `prompt` recipe.
5. Add a `mod` declaration in the root `Justfile`.
6. Set `ai_bin` in `settings.just` to the new provider name to make it the default.

### Adding a New AI Recipe

1. Create the prompt definition in `docs/prompts/<name>/README.md`.
2. Create `<name>.just` in each AI backend directory, calling the `prompt` recipe with the prompt directory name.
3. Create `<name>.just` in `bin/just/ai/` to dispatch to `just {{ai_bin}} <name>`.
4. Import the new file in each `.mod.just`.
5. Optionally create a root recipe in `bin/just/root/` if the command should be user-facing.

## Configuration Reference

All runtime configuration is defined in `bin/just/settings.just`. Per [philosophy.md](../../foundation/philosophy.md), when this document and configuration conflict, configuration wins.
