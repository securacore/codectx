# Compilation

Running `codectx compile` builds a compiled documentation set from everything you've activated. The compiled output lives in `.codectx/` and is optimized for AI consumption: flat file layout, content-addressed filenames, rewritten cross-references, optional CMDX compression, and a single data map that indexes everything.

The compiled format is validated by `compiled.schema.json` (not yet created).

## What Happens When You Compile

Compilation runs through seven stages, in order. A progress display shows each stage as it executes, with a spinner and per-file details.

### 1. Combination

All active documentation sources are merged into a single unified set:

- Your project's own documentation from `docs/`
- Activated entries from any installed packages in `docs/packages/`

Only entries you've activated in `codectx.yml` are included. If two packages provide an entry with the same ID, the local package always wins. Between installed packages, the one listed first in your config takes precedence. If the content is identical (same SHA256 hash), the duplicate is silently deduplicated. If the content differs, the precedence winner is used and a warning is emitted.

### 2. Content-Addressed Storage

Each documentation file is stored in a flat `objects/` directory. The filename is a 16-character SHA256 prefix of the stored content (e.g., `objects/a1b2c3d4e5f67890.md` or `objects/a1b2c3d4e5f67890.cmdx`).

When compression is disabled, the hash is computed from the original source content, and files use the `.md` extension. When compression is enabled, the hash is computed from the CMDX-encoded content, and files use the `.cmdx` extension. Either way, identical stored content produces the same filename and is naturally deduplicated.

Plan state files are stored separately in `state/` because they're mutable and change between compiles.

### 3. Link Rewriting

Source documentation files contain relative markdown links that reference their original directory structure (e.g., `[hooks](hooks.md)` or `[philosophy](../../foundation/philosophy.md)`). Since compiled objects live in a flat `objects/` directory, those relative paths would break.

During this stage, the compiler rewrites every markdown link target in each compiled object:

- **Resolvable links** are rewritten to the target's content-addressed filename. For example, `[hooks](hooks.md)` becomes `[hooks](bbbb444455556666.md)` (or `.cmdx` when compressed).
- **Unresolvable links** (targets not in the compiled set, such as references to topics you haven't activated) use the `unresolved:` URI scheme. For example, `[TypeScript](../typescript/README.md)` becomes `[TypeScript](unresolved:../typescript/README.md)`. This tells AI the reference exists but the target isn't available in this compilation.
- **HTTP/HTTPS links** are left untouched.
- **Fragment suffixes** (e.g., `#section`) are preserved on rewritten links.
- **Non-markdown links** (JSON, YAML, etc.) are left untouched.

Link rewriting happens before compression. The compiler rewrites links in the Markdown source, then encodes the rewritten content to CMDX (if compression is enabled). This means CMDX-encoded files contain links to content-addressed objects, not to original source paths.

Source files in `docs/` are never modified. Link rewriting only affects the compiled objects in `.codectx/objects/`.

### 4. Manifest Generation

A compiled `manifest.yml` (the "data map") is built from the unified entries. Each entry references its content-addressed object path (e.g., `objects/a1b2c3d4e5f67890.md`) instead of a relative file path, and includes a `source` field for provenance tracking (`"local"` or `"name@author"`).

This is the `.codectx/manifest.yml` -- distinct from the source `docs/manifest.yml`. See the [Package Format](packages.md) docs for more on how these two manifests differ.

### 5. Heuristics

A `heuristics.yml` sidecar is generated containing size estimates, token counts, and per-package statistics. This file is not part of the AI loading protocol; it's used by tooling and by the generated README.

Contents:

- **Totals**: entry count, unique object count, total bytes, estimated tokens, always-load count
- **Sections**: per-section stats (entries, bytes, tokens, always-load count for foundation)
- **Packages**: per-package stats (entries, bytes, tokens) with local package listed first

Token counts use real BPE tokenization via the `o200k_base` encoding (GPT-4o class), matching the [baseline model assumption](../foundation/ai-authoring/README.md#baseline-model-assumption). When compression is enabled, the byte and token counts reflect the compressed content, giving an accurate picture of actual AI context-window consumption.

The compile progress display shows key heuristics after compilation completes: total objects stored, packages compiled, total bytes, and estimated tokens.

### 6. Decomposition

If the documentation set exceeds decomposition thresholds, the manifest is split into per-section sub-manifests stored in `manifests/`. See [Manifest Decomposition](#manifest-decomposition) below.

### 7. README and Lock

A `README.md` is generated from the unified manifest and heuristics (includes token estimates, section summaries, and the loading protocol). When compression is enabled, the README includes a note explaining the `.cmdx` format to AI consumers. A `codectx.lock` file is generated with resolved versions, checksums, and full activation state.

## Compiled Output Format

After compilation, your `.codectx/` directory looks like this:

```text
.codectx/
  manifest.yml                    # Compiled data map (AI reads this)
  README.md                       # Generated loading protocol
  heuristics.yml                  # Size/token stats (tooling only, not loaded by AI)
  preferences.yml                 # User preferences (preserved across compiles)
  objects/                        # Content-addressed file store
    {16-char-sha256}.md           # Uncompressed objects (when compression disabled)
    {16-char-sha256}.cmdx         # Compressed objects (when compression enabled)
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
- **`objects/`** is a flat directory of content-addressed files. Filenames are 16-character SHA256 prefixes with `.md` or `.cmdx` extension, providing 64 bits of collision resistance. Links inside these files are rewritten to reference other objects by hash.
- **`state/`** contains mutable plan state files. These are not content-addressed because they change between compiles.
- **`preferences.yml`** stores user-specific settings (e.g., compression, AI model class). It is preserved across recompiles.

## Fingerprinting

Compilation uses fingerprint-based change detection. The fingerprint incorporates source file hashes and the compression setting, so toggling compression correctly triggers a recompile. If no source files have changed and the compression setting hasn't changed since the last compile, the compiler returns immediately with an up-to-date result. This makes `codectx compile` safe to run as often as you like with negligible cost.

## Heuristics

The `heuristics.yml` sidecar provides aggregate metadata about the compiled documentation set. It is generated during compilation and is not loaded by AI. Validated by `heuristics.schema.json` (not yet created).

Contents:

- **Totals**: entry count, unique object count, total bytes, estimated tokens, always-load count
- **Sections**: per-section stats (entries, bytes, tokens, always-load count for foundation)
- **Packages**: per-package stats (entries, bytes, tokens) with local package listed first

Token counts are computed using `o200k_base` BPE tokenization (the encoding used by GPT-4o-class models) rather than a byte-based estimate. This gives accurate context-window predictions. The tokenizer runs offline with no network access required. See [AI Authoring: Baseline Model Assumption](../foundation/ai-authoring/README.md#baseline-model-assumption) for why this encoding was chosen.

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
- [Compression](compression.md) -- CMDX codec details and performance characteristics
- [Configuration](configuration.md) -- activation settings that control what gets compiled
- [Preference Management](set-command.md) -- compression and auto_compile preferences
- [Design Decisions](spec/README.md) -- reasoning behind compilation choices
