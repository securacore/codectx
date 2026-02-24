# Compilation

The `codectx compile` command builds the compiled documentation set from all active documentation. The compiled output in `.codectx/` uses a distinct format from source packages, optimized for AI consumption. It is validated by [compiled.schema.json](../schemas/compiled.schema.json).

## Compile Process

Compilation has six stages:

### 1. Combination

Merge documentation from all active sources:

- The local package (project's own documentation in `docs/`)
- Activated entries from installed packages in `docs/packages/`

Only entries that are activated in `codectx.yml` are included. Entries with duplicate IDs are resolved by precedence (local wins, then config order) and content hash (identical content = silent dedup; different content = warning, precedence wins).

### 2. Content-Addressed Storage

Each documentation file is stored in the `objects/` directory using a 16-character SHA256 prefix as the filename (e.g., `objects/a1b2c3d4e5f67890.md`). Identical content across packages produces the same hash, giving natural deduplication. Plan state files are stored separately in `state/` because they are mutable.

### 3. Manifest Generation

Build a compiled `manifest.yml` from the unified entries. Each entry references its content-addressed object path (e.g., `objects/a1b2c3.md`) instead of a relative file path, and includes a `source` field for provenance tracking (`"local"` or `"name@author"`).

### 4. Heuristics

Generate `heuristics.yml`, a sidecar metadata file containing size estimates, token counts, and per-package statistics. The heuristics file is not part of the AI loading protocol; it is used by tooling and by the generated README. See [Heuristics](#heuristics).

### 5. Decomposition

If the documentation set exceeds decomposition thresholds, the manifest is split into per-section sub-manifests stored in `manifests/`. See [Manifest Decomposition](#manifest-decomposition).

### 6. README and Lock

Generate `README.md` from the unified manifest and heuristics (includes token estimates, section summaries, and loading protocol). Generate `codectx.lock` with resolved versions, checksums, and full activation state.

## Compiled Output Format

```text
.codectx/
  manifest.yml                    # Compiled data map
  README.md                       # Generated loading protocol
  heuristics.yml                  # Size/token stats (not loaded by AI)
  preferences.yml                 # User preferences (preserved across compiles)
  objects/                        # Content-addressed file store
    {16-char-sha256}.md
  state/                          # Mutable plan state files
    {plan-id}.yml
  manifests/                      # Sub-manifests (only when decomposed)
    foundation.yml
    topics.yml
    prompts.yml
    plans.yml
```

Key properties of the compiled format:

- `manifest.yml` is the compiled data map. Each entry has an `object` field (content-addressed path), a `source` field (provenance), and standard `depends_on`/`required_by` edges.
- `objects/` is a flat directory of content-addressed files. Filenames are 16-character SHA256 prefixes with `.md` extension, providing 64 bits of collision resistance.
- `state/` contains mutable plan state files. These are not content-addressed because they change between compiles.
- `preferences.yml` stores user-specific settings (e.g., auto-compile). It is preserved across recompiles.

## Fingerprinting

Compilation uses fingerprint-based change detection. If no manifest-referenced files have changed since the last compile, the compiler returns immediately with an up-to-date result. This makes `codectx compile` safe to run frequently with negligible cost.

## Heuristics

The `heuristics.yml` sidecar provides aggregate metadata about the compiled documentation set. It is generated during compilation and is not part of the AI loading protocol.

Contents:

- **Totals**: entry count, unique object count, total bytes, estimated tokens, always-load count
- **Sections**: per-section stats (entries, bytes, tokens, always-load count for foundation)
- **Packages**: per-package stats (entries, bytes, tokens) with local package listed first

Token estimates use ~4 characters per token as a rough conversion factor.

## Manifest Decomposition

When the documentation set exceeds any of the following thresholds, the compiled manifest is decomposed into sub-manifests:

- Entries > 500
- Total bytes > 50 KB
- Estimated tokens > 100,000

Decomposition splits at the section level. Each non-empty section gets its own sub-manifest file in `manifests/` (e.g., `manifests/topics.yml`). Foundation entries with `load: always` remain inlined in the root `manifest.yml` so AI can load them without additional file reads. The root manifest contains `ManifestRef` entries that describe each sub-manifest (section, path, entry count, estimated tokens).

Sub-manifests use the same schema, enabling recursive decomposition at deeper levels if needed in the future.

## Lock File

`codectx.lock` records the exact resolved versions, checksums, and activation state so the compiled output can be reproduced deterministically. `codectx add --lockfile` reads the lock file to reproduce the exact package state.

## Related

- [Package Format](packages.md) -- source package structure and manifest
- [Configuration](configuration.md) -- activation settings that control what gets compiled
- [Design Decisions](spec/README.md) -- reasoning behind compilation choices
