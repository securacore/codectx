# Documentation Metadata

Conventions for maintaining `docs/metadata.yml`, the documentation manifest that governs relationships, portability, and AI-first lookup across all documentation.

<rules>

- `docs/metadata.yml` is the single source of truth for documentation structure, relationships, and entry points.
- `docs/metadata.schema.json` validates the manifest. IDE YAML extensions can reference it for inline validation.
- All paths in the manifest are relative to the `docs/` directory.

</rules>

## Structure

The manifest has three top-level sections: `foundation`, `topics`, and `prompts`. Each section is an array of entries.

<rules>

- **Foundation entries** represent documents in `docs/foundation/`. Each has a `load` field controlling when it is loaded into AI context.
- **Topic entries** represent directories in `docs/topics/`. Each topic has a README.md entry point and an optional spec. Topics are loaded on-demand based on the task.
- **Prompt entries** represent directories in `docs/prompts/`. Prompts are freeform and do not follow topic conventions. Prompts are loaded when the specific prompt is invoked.

</rules>

## Loading Context

Foundation documents have a `load` field that determines when they are loaded into AI context. This controls startup token cost by deferring documents that are only relevant to documentation tasks.

<rules>

- `load: always` entries are loaded at the start of every session, regardless of task type. These are the core operating context: the documentation map and the decision-making framework.
- `load: documentation` entries are loaded only when the task involves writing, editing, reviewing, or restructuring any file under `docs/`. Reading documentation to follow conventions during code work does not trigger loading these entries.
- Before making any changes to `metadata.yml`, load `metadata.schema.json` into context. Do not edit the manifest without the schema loaded.
- Topics and prompts do not have a `load` field. They are loaded on-demand when the task requires them, using the `depends_on` graph in the manifest to determine what is relevant.

</rules>

```yaml
# Correct: philosophy is always needed for decision-making
- id: philosophy
  path: foundation/philosophy.md
  load: always

# Correct: markdown formatting rules only needed when writing docs
- id: markdown
  path: foundation/markdown.md
  load: documentation

# Incorrect: documentation-authoring context loaded in a code-only session
# The agent is implementing a React component. It reads docs/topics/react/components.md
# to follow conventions, but it does NOT need to load markdown.md, ai-authoring.md,
# or review-standards.md because it is not writing or editing documentation.
```

## Identifiers

Every entry has a unique `id`. IDs form a shared namespace across all three sections.

<rules>

- IDs are unique across the entire manifest. No foundation entry, topic entry, or prompt entry shares an `id` with another.
- IDs are lowercase, hyphen-separated, and match the filename (without extension) or directory name of the entry. `philosophy` for `philosophy.md`, `react` for `topics/react/`.
- `depends_on` and `required_by` reference IDs, not paths. An entry in any section can reference an entry in any other section.

</rules>

```yaml
# Correct: IDs match filenames/directory names
- id: philosophy
  path: foundation/philosophy.md

- id: react
  path: topics/react/README.md
  depends_on: [typescript, philosophy]

# Incorrect: ID does not match the filename
- id: phil              # WRONG: must be "philosophy" to match philosophy.md
  path: foundation/philosophy.md

# Incorrect: duplicate ID across sections
# foundation:
- id: save              # WRONG: conflicts with prompt entry "save"
# prompts:
- id: save
```

## Relationship Symmetry

`depends_on` and `required_by` are bidirectional. Every relationship is maintained explicitly in both directions for instant AI lookup without graph traversal.

<rules>

- If entry A lists B in `depends_on`, then entry B must list A in `required_by`. No exceptions.
- If entry A lists B in `required_by`, then entry B must list A in `depends_on`. No exceptions.
- When adding, removing, or renaming an entry, update both sides of every affected relationship.
- Empty arrays are explicit: `depends_on: []` and `required_by: []` indicate no relationships, not missing data.

</rules>

```yaml
# Correct: symmetric relationship
# In foundation section:
- id: philosophy
  depends_on: []
  required_by: [react]

# In topics section:
- id: react
  depends_on: [philosophy]
  required_by: []

# Incorrect: asymmetric relationship
# In foundation section:
- id: philosophy
  depends_on: []
  required_by: []          # WRONG: react depends on philosophy but philosophy
                           # does not list react in required_by

# In topics section:
- id: react
  depends_on: [philosophy]
  required_by: []
```

## Topic Files

Topic entries with concordance structure (README.md linking to sub-files) list their sub-files in the `files` array. Single-file topics use an empty array.

<rules>

- The `files` array lists all convention sub-files for the topic, relative to `docs/`.
- The `files` array does not include the topic's README.md (that is the `path` field) or the spec (that is the `spec` field).
- Single-file topics (where README.md is the entire documentation) use `files: []`.

</rules>

```yaml
# Correct: concordance topic with sub-files
- id: react
  path: topics/react/README.md
  spec: topics/react/spec/README.md
  files:
    - topics/react/components.md
    - topics/react/hooks.md
    - topics/react/state.md

# Correct: single-file topic
- id: typescript
  path: topics/typescript/README.md
  spec: topics/typescript/spec/README.md
  files: []

# Incorrect: README included in files
- id: react
  path: topics/react/README.md
  files:
    - topics/react/README.md       # WRONG: already in path field
    - topics/react/components.md
```

## Maintenance

<rules>

- **Adding a topic:** Add the entry to the `topics` section. Set `depends_on` based on what the topic references. Update `required_by` on every entry listed in `depends_on`. Add the topic to the `docs/README.md` Topics table.
- **Adding a foundation document:** Add the entry to the `foundation` section. Follow the same symmetry rules. Add the document to the `docs/README.md` Foundational table.
- **Adding a prompt:** Add the entry to the `prompts` section. Set `depends_on` if the prompt references specific foundation or topic docs. Update `required_by` on referenced entries. Add the prompt to the `docs/README.md` Prompts table.
- **Removing an entry:** Remove the entry. Remove its ID from every `depends_on` and `required_by` list across the manifest. Remove it from the corresponding `docs/README.md` table.
- **Renaming an entry:** Update the `id`, `path`, and any `depends_on`/`required_by` references across the entire manifest.
- **Validation:** The `docs/metadata.schema.json` file validates structure. Relationship symmetry is verified by the docs-audit prompt.

</rules>
