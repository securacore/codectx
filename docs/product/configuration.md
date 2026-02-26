# Configuration

`codectx.yml` at the repository root is the single source of truth for your project's documentation setup. It declares package dependencies, what's activated, and where things live. It's validated by [codectx.schema.json](../schemas/codectx.schema.json).

User-local settings like compression, AI provider, and model class are stored separately in `.codectx/preferences.yml` (gitignored). See [Preference Management](set-command.md).

## Settings

| Field | Default | Description |
|---|---|---|
| `name` | -- | Project name (used in compiled output and README) |
| `config.docs_dir` | `docs` | Where your source documentation lives |
| `config.output_dir` | `.codectx` | Where compiled output is written |
| `packages` | `[]` | List of installed package dependencies |

## User Preferences

User preferences are stored in `.codectx/preferences.yml`, managed by `codectx set`:

| Key | Type | Default | Description |
|---|---|---|---|
| `compression` | bool | `true` (new projects) | Encode compiled objects to CMDX format |
| `auto_compile` | bool | `true` (new projects) | Recompile automatically after changes |
| `ai.provider` | string | unset | AI provider (`claude`, `opencode`, `ollama`) |
| `ai.model` | string | unset | AI model name (Ollama only) |
| `ai.class` | string | `gpt-4o-class` (new projects) | Documentation compatibility target |

Preferences are personal and not shared. The `.codectx/` directory is gitignored. See [Preference Management](set-command.md) for detailed documentation.

## Package Dependencies

Each entry in the `packages` array declares an installed package and what to activate from it:

```yaml
name: my-project

config:
  docs_dir: docs
  output_dir: .codectx

packages:
  - name: react
    author: org
    version: "^1.0.0"
    active: all
  - name: typescript
    author: org
    version: "~2.0.0"
    active:
      topics:
        - conventions
        - patterns
```

You don't typically edit this file by hand. `codectx add` and `codectx activate` manage it interactively.

## Activation

The `active` field on each package controls what gets included in the compiled output. There are three modes:

**Activate everything:**

```yaml
active: all
```

**Activate nothing** (package is installed but contributes nothing to the compiled output):

```yaml
active: none
```

This is the default when `active` is omitted.

**Activate specific entries** by listing entry IDs per section:

```yaml
active:
  foundation:
    - core-principles
  topics:
    - conventions
    - patterns
  prompts:
    - commit
```

Activation is always explicit. You choose exactly what documentation is included in your compiled output.

### Interactive Activation

When you add a package with `codectx add`, the CLI reads the package's manifest (or auto-discovers its contents), presents what's available, and prompts you to choose what to activate. You can activate all entries, select specific ones, or activate none and decide later.

## Conflict Handling

Packages are namespaced by `name@author`, so installed packages never create file-level conflicts. Conflicts only arise at the **activation level**: when two packages both provide documentation for the same domain (same entry ID) and both are activated.

During `codectx add`:

1. The CLI inspects the new package's contents
2. If activating an entry would overlap with an already-active entry from another package, the CLI warns you
3. You're prompted to resolve the conflict interactively

### Deduplication During Compilation

During compilation, entries with duplicate IDs across packages are resolved by:

- **Precedence**: your local package always wins, then config order (first listed wins)
- **Content hash**: if the content is byte-for-byte identical (same SHA256), it's silently deduplicated. If the content differs, the precedence winner is used and a warning is emitted.

## Directory Layout

A project using codectx has three distinct areas:

```text
project-root/
  codectx.yml                       # Configuration (your source of truth)
  codectx.lock                      # Resolved state from last compile

  docs/                             # Source documentation
    manifest.yml                    # Local package data map
    schemas/                        # Validation schemas
    packages/                       # Installed packages
      [name]@[author]/              # Each package in its own directory
        manifest.yml                # Package data map (optional, auto-discovered if missing)
        topics/
        foundation/
        ...
    foundation/                     # Your own foundation docs
    topics/                         # Your own topics
    prompts/                        # Your own prompts
    plans/                          # Your own plans

  .codectx/                         # Compiled output (gitignored)
    manifest.yml                    # Compiled data map (what AI reads)
    README.md                       # Generated loading protocol
    heuristics.yml                  # Size and token stats
    preferences.yml                 # User preferences (personal, preserved)
    objects/                        # Content-addressed files (.md or .cmdx)
    ...

  CLAUDE.md                         # AI entry point (created by codectx link)
  AGENTS.md                         # AI entry point (created by codectx link)
```

Whether you check `docs/packages/` into Git is your choice. The lock file ensures reproducibility regardless.

## Related

- [Package Format](packages.md) -- package structure and manifest format
- [Compilation](compilation.md) -- how configuration drives the compile process
- [Compression](compression.md) -- CMDX compression and the `compression` preference
- [Preference Management](set-command.md) -- `codectx set` and user preferences
- [AI Integration](ai-integration.md) -- how compiled output connects to AI tools
- [Design Decisions](spec/README.md) -- reasoning behind configuration choices
