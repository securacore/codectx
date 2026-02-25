# Configuration

`codectx.yml` at the repository root is the sole source of truth for a project's documentation setup. It declares package dependencies, activation state, and build settings. It is validated by [codectx.schema.json](../schemas/codectx.schema.json).

## Settings

| Field | Default | Description |
|---|---|---|
| `name` | -- | Project name |
| `config.docs_dir` | `docs` | Documentation source directory |
| `config.output_dir` | `.codectx` | Compiled output directory |
| `packages` | `[]` | Package dependencies |

## Package Dependencies

Each entry in the `packages` array declares a dependency:

```yaml
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

## Activation

The `active` field on each package controls what is included in the compiled output:

- `all` -- activates every entry in the package
- `none` -- installs the package but activates nothing (default when omitted)
- **Granular object** -- lists specific entry IDs per section:

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

Activation is always explicit. When `active` is omitted, the default is `none`.

### Interactive Activation

When a package is added with `codectx add`, the CLI reads the package's `manifest.yml`, presents its contents, and prompts the user to choose what to activate. The user can activate all entries, select specific entries, or activate none.

## Conflict Handling

Packages are namespaced by `name@author`, so installed packages never create file-level conflicts. Conflicts only arise at the activation level: when two packages both provide documentation for the same domain and both are activated.

During `codectx add`:

1. The CLI reads the new package's `manifest.yml` to inspect its contents
2. If activating an entry would create a domain overlap with an already-active entry from another package, the CLI warns the user
3. The user is prompted to resolve the conflict interactively

### Deduplication

During compilation, entries with duplicate IDs across packages are resolved by:

- **Precedence**: local package always wins, then config order
- **Content hash**: identical SHA256 content = silent dedup; different content = warning, precedence wins

## Directory Layout

A project using codectx has three distinct areas:

```text
project-root/
  codectx.yml                       # Configuration (sole source of truth)
  codectx.lock                      # Resolved state from compile

  docs/                             # Documentation source
    manifest.yml                     # Local package data map
    schemas/                        # Validation schemas
    packages/                       # Installed packages
      [name]@[author]/
        manifest.yml
        ...
    foundation/                     # Local foundation docs
    topics/                         # Local topic docs
    prompts/                        # Local prompts
    plans/                          # Local plans

  .codectx/                         # Compiled output (gitignored)
    manifest.yml                    # Compiled data map
    README.md                       # Generated loading protocol
    ...

  CLAUDE.md                         # AI entry point (created by codectx link)
  AGENTS.md                         # AI entry point (created by codectx link)
```

Whether `docs/packages/` is checked into Git is the user's choice. The lock file ensures reproducibility regardless.

## Related

- [Package Format](packages.md) -- package structure and manifest format
- [Compilation](compilation.md) -- how configuration drives the compile process
- [AI Integration](ai-integration.md) -- how compiled output connects to AI tools
- [Design Decisions](spec/README.md) -- reasoning behind configuration choices
