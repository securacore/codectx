# Preference Management

`codectx set` manages user-local preferences stored in `.codectx/preferences.yml`. These preferences are personal — they're gitignored and not shared with collaborators. Each team member can configure compression, AI providers, and documentation targeting independently.

## Usage

**View all preferences:**

```bash
codectx set
```

Displays every known preference key with its current value and description:

```
Preferences:

  compression      true       Encode compiled objects to CMDX format
  auto_compile     true       Recompile automatically after changes
  ai.provider      claude     AI provider (claude, opencode, ollama)
  ai.model         (unset)    AI model name (ollama only)
  ai.class         gpt-4o-class  Documentation target model class

  Set a value: codectx set key=value
```

**Set a preference:**

```bash
codectx set compression=false
codectx set ai.class=o1-class
codectx set ai.provider=claude
```

## Preference Keys

### `compression`

**Type:** bool | **Default (new projects):** `true`

When enabled, compiled objects are encoded to CMDX format during compilation, reducing token usage by ~25% on structured content. When disabled, compiled objects are stored as plain Markdown.

Existing projects (created before compression was added) have this unset, which is treated as `false` for backward compatibility.

```bash
codectx set compression=true
codectx set compression=false
```

### `auto_compile`

**Type:** bool | **Default (new projects):** `true`

When enabled, certain commands (like `codectx add`) automatically trigger a recompile after making changes. When disabled, you must run `codectx compile` manually.

```bash
codectx set auto_compile=true
codectx set auto_compile=false
```

### `ai.provider`

**Type:** string | **Default:** unset

The AI provider to use for AI-assisted features. Validated against the known provider registry. The provider's binary must be available on PATH.

Known providers: `claude` (Claude Code), `opencode` (opencode), `ollama` (Ollama).

```bash
codectx set ai.provider=claude
codectx set ai.provider=ollama
codectx set ai.provider=           # Clear the provider (also clears model)
```

Setting the provider to an empty string clears the entire AI configuration (provider and model).

### `ai.model`

**Type:** string | **Default:** unset

The model name for providers that require explicit model selection (primarily Ollama). Not validated — any string is accepted because model availability depends on the provider's local state.

```bash
codectx set ai.model=llama3.2:latest
codectx set ai.model=codellama:7b
```

### `ai.class`

**Type:** string | **Default (new projects):** `gpt-4o-class`

The documentation compatibility target. This defines the minimum model capability tier that compiled documentation is written for. It is NOT the model being used — it controls how the AI authoring foundation document adapts its guidance.

Known classes:

| Class | Description |
|-------|-------------|
| `gpt-4o-class` | Mid-tier instruction-following models (GPT-4o, Claude Sonnet, Gemini Pro) |
| `claude-sonnet-class` | Strong reasoning models with extended context |
| `o1-class` | Frontier reasoning models (o1, Claude Opus, Gemini Ultra) |

```bash
codectx set ai.class=gpt-4o-class
codectx set ai.class=o1-class
codectx set ai.class=              # Clear (unset)
```

The `gpt-4o-class` default is the most conservative baseline — documentation written for this tier works at all higher tiers. Setting a higher class allows documentation to use denser language and assume stronger reasoning capabilities.

## Validation

- **Bool keys** (`compression`, `auto_compile`) accept: `true`, `false`, `1`, `0`, `yes`, `no` (case-insensitive).
- **`ai.provider`** is validated against the known provider registry. The provider's binary must also be available on PATH.
- **`ai.class`** is validated against the known model class registry. Unlike providers, no binary check is needed — classes are documentation targets, not executable tools.
- **`ai.model`** accepts any string without validation.
- **Unknown keys** are rejected with an error listing available keys.

## Storage

Preferences are stored in `.codectx/preferences.yml` as a YAML file:

```yaml
compression: true
auto_compile: true
ai:
  provider: claude
  class: gpt-4o-class
```

The `.codectx/` directory is gitignored. Preferences are preserved across recompiles — `codectx compile` never touches `preferences.yml`.

### Pointer Bool Semantics

Boolean preferences use pointer semantics internally (`*bool`). This distinguishes three states:

- **`true`** — Explicitly enabled
- **`false`** — Explicitly disabled
- **`nil` (unset)** — Never configured; the command interprets this based on context (e.g., compression `nil` is treated as `false` for backward compatibility with existing projects)

New projects created with `codectx init` set both booleans to `true` explicitly.

## Related

- [Configuration](configuration.md) — project-level settings in `codectx.yml`
- [Compression](compression.md) — what the `compression` preference controls
- [AI Integration](ai-integration.md) — how `ai.provider` and `ai.class` are used
- [Design Decisions](spec/README.md) — reasoning behind preference design choices
