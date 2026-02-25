# Compilation

Running `codectx compile` builds a compiled documentation set from everything you've activated. The compiled output lives in `.codectx/` and is optimized for AI consumption: flat file layout, content-addressed filenames, rewritten cross-references, and a single data map that indexes everything.

The compiled format is validated by [compiled.schema.json](../schemas/compiled.schema.json).

## What Happens When You Compile

Compilation runs through seven stages, in order.

### 1. Combination

All active documentation sources are merged into a single unified set:

- Your project's own documentation from `docs/`
- Activated entries from any installed packages in `docs/packages/`

Only entries you've activated in `codectx.yml` are included. If two packages provide an entry with the same ID, the local package always wins. Between installed packages, the one listed first in your config takes precedence. If the content is identical (same SHA256 hash), the duplicate is silently deduplicated. If the content differs, the precedence winner is used and a warning is emitted.

### 2. Content-Addressed Storage

Each documentation file is stored in a flat `objects/` directory. The filename is a 16-character SHA256 prefix of the file's **original source content** (e.g., `objects/a1b2c3d4e5f67890.md`). This means identical files across different packages produce the same filename and are naturally deduplicated -- you never store the same content twice.

Plan state files are stored separately in `state/` because they're mutable and change between compiles.

### 3. Link Rewriting

Source documentation files contain relative markdown links that reference their original directory structure (e.g., `[hooks](hooks.md)` or `[philosophy](../../foundation/philosophy.md)`). Since compiled objects live in a flat `objects/` directory, those relative paths would break.

During this stage, the compiler rewrites every markdown link target in each compiled object:

- **Resolvable links** are rewritten to the target's content-addressed filename. For example, `[hooks](hooks.md)` becomes `[hooks](bbbb444455556666.md)` where `bbbb444455556666` is the hash of `hooks.md`.
- **Unresolvable links** (targets not in the compiled set, such as references to topics you haven't activated) use the `unresolved:` URI scheme. For example, `[TypeScript](../typescript/README.md)` becomes `[TypeScript](unresolved:../typescript/README.md)`. This tells AI the reference exists but the target isn't available in this compilation.
- **HTTP/HTTPS links** are left untouched.
- **Fragment suffixes** (e.g., `#section`) are preserved on rewritten links.
- **Non-markdown links** (JSON, YAML, etc.) are left untouched.

The hash used as the filename is always computed from the **raw source content** before link rewriting. This preserves the deduplication property: the same source file always maps to the same object filename regardless of what else is compiled alongside it. The stored content (with rewritten links) may differ from the raw source, but the filename reflects source identity.

Source files in `docs/` are never modified. Link rewriting only affects the compiled objects in `.codectx/objects/`.

### 4. Manifest Generation

A compiled `manifest.yml` (the "data map") is built from the unified entries. Each entry references its content-addressed object path (e.g., `objects/a1b2c3d4e5f67890.md`) instead of a relative file path, and includes a `source` field for provenance tracking (`"local"` or `"name@author"`).

This is the `.codectx/manifest.yml` -- distinct from the source `docs/manifest.yml`. See the [Package Format](packages.md) docs for more on how these two manifests differ.

### 5. Heuristics

A `heuristics.yml` sidecar is generated containing size estimates, token counts, and per-package statistics. This file is not part of the AI loading protocol; it's used by tooling and by the generated README. See [Heuristics](#heuristics) below.

### 6. Decomposition

If the documentation set exceeds decomposition thresholds, the manifest is split into per-section sub-manifests stored in `manifests/`. See [Manifest Decomposition](#manifest-decomposition) below.

### 7. README and Lock

A `README.md` is generated from the unified manifest and heuristics (includes token estimates, section summaries, and the loading protocol). A `codectx.lock` file is generated with resolved versions, checksums, and full activation state.

## Compiled Output Format

After compilation, your `.codectx/` directory looks like this:

```text
.codectx/
  manifest.yml                    # Compiled data map (AI reads this)
  README.md                       # Generated loading protocol
  heuristics.yml                  # Size/token stats (tooling only, not loaded by AI)
  preferences.yml                 # User preferences (preserved across compiles)
  objects/                        # Content-addressed file store
    {16-char-sha256}.md           # Each file named by its source content hash
  state/                          # Mutable plan state files
    {plan-id}.yml
  manifests/                      # Sub-manifests (only present when decomposed)
    foundation.yml
    topics.yml
    prompts.yml
    plans.yml
```

Key properties:

- **`manifest.yml`** is the compiled data map. Each entry has an `object` field (content-addressed path), a `source` field (provenance), and standard `depends_on`/`required_by` edges.
- **`objects/`** is a flat directory of content-addressed files. Filenames are 16-character SHA256 prefixes with `.md` extension, providing 64 bits of collision resistance. Links inside these files are rewritten to reference other objects by hash.
- **`state/`** contains mutable plan state files. These are not content-addressed because they change between compiles.
- **`preferences.yml`** stores user-specific settings (e.g., auto-compile). It is preserved across recompiles.

## Fingerprinting

Compilation uses fingerprint-based change detection. If no source files have changed since the last compile, the compiler returns immediately with an up-to-date result. This makes `codectx compile` safe to run as often as you like with negligible cost.

## Heuristics

The `heuristics.yml` sidecar provides aggregate metadata about the compiled documentation set. It is generated during compilation and is not loaded by AI.

Contents:

- **Totals**: entry count, unique object count, total bytes, estimated tokens, always-load count
- **Sections**: per-section stats (entries, bytes, tokens, always-load count for foundation)
- **Packages**: per-package stats (entries, bytes, tokens) with local package listed first

Token estimates use ~4 characters per token as a rough conversion factor.

## Manifest Decomposition

When the documentation set exceeds any of the following thresholds, the compiled manifest is automatically decomposed into sub-manifests:

- More than 500 entries
- More than 50 KB total size
- More than 100,000 estimated tokens

Decomposition splits at the section level. Each non-empty section gets its own sub-manifest file in `manifests/` (e.g., `manifests/topics.yml`). Foundation entries marked `load: always` remain inlined in the root `manifest.yml` so AI can load them without additional file reads. The root manifest contains references that describe each sub-manifest (section, path, entry count, estimated tokens).

Sub-manifests use the same schema, enabling recursive decomposition at deeper levels if needed in the future.

## Lock File

`codectx.lock` records the exact resolved versions, checksums, and activation state so the compiled output can be reproduced deterministically. Running `codectx add --lockfile` reads the lock file to reproduce the exact package state.

## Related

- [Package Format](packages.md) -- source package structure and the two kinds of manifest
- [Configuration](configuration.md) -- activation settings that control what gets compiled
- [Design Decisions](spec/README.md) -- reasoning behind compilation choices
