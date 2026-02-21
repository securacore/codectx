# Foundation Specification

Spec for the foundational documentation that governs all work in this repository. For the conventions themselves, see the files listed in the Structure section.

## Purpose

Foundational documentation establishes the baseline conventions that every other document, topic, and AI session depends on. Without it, each new document or AI interaction would reinvent conventions from scratch, leading to inconsistency and drift. Core foundation documents (`load: always`) are loaded into every AI context. Documentation-authoring foundation documents (`load: documentation`) are loaded when the session involves writing or editing documentation.

## Decisions

- **AI-first audience.** AI assistants are the primary consumers in active sessions and need predictable structure for reliable parsing. Engineers benefit from the same clarity. Alternative considered: engineer-first (rejected; AI context loading is the higher-frequency use case and imposes stricter structural requirements).

- **Cohesive foundational documents.** The foundation is split by concern: philosophy, markdown, documentation strategy, and the spec template. Each is a distinct responsibility that rarely changes. Alternative considered: a single foundational document (rejected; exceeds useful token budgets and mixes unrelated concerns). Alternative considered: more granular splits (rejected; the current documents already map cleanly to distinct responsibilities).

- **Philosophy as a separate document.** Guiding principles are isolated from concrete rules. Philosophy informs _why_ rules exist; other documents define the rules themselves. This separation prevents philosophical drift from contaminating actionable conventions.

- **`docs/README.md` as the single index.** One file serves as both entry point and complete documentation index. Alternative considered: a separate concordance file (rejected after initial implementation; it duplicated the README's foundational listing and added a navigation hop without unique content).

- **Foundational docs live in `docs/foundation/`.** Originally placed at the `docs/` root. Moved to `docs/foundation/` to create a consistent pattern where all documentation tiers live in subdirectories, and to enable a `spec/` subdirectory for foundational meta-documentation. The `docs/` root retains only `README.md` as the universal entry point and index.

- **Spec template as a foundational document.** The spec template (`specs.md`) defines the format for all specs across every topic. It governs everything, not a specific technology, making it foundational by nature. Alternative considered: a standalone `docs/specs/` topic directory (rejected after initial implementation; it was a topic directory that wasn't really a topic).

- **Spec subdirectories for meta-documentation.** Each topic directory (including foundation) may contain a `spec/` subdirectory. This captures reasoning separately from conventions, so the conventions stay clean and the reasoning is available when needed for recreation or revision.

- **Cross-model authoring as a foundational document.** Documentation and prompts must work across model capability tiers. Authoring conventions are foundational because they govern how all content is written, not a specific technology. Alternative considered: embedding authoring rules in markdown.md (rejected; markdown.md governs formatting structure, authoring governs linguistic patterns and instructional design. These are complementary but distinct concerns).

- **Review standards as a foundational document.** Every documentation update requires a review pass against the full set of foundational standards: formatting, linguistic, example, semantic marker, token positioning, structural integrity, and integration. Without a codified checklist, these checks are performed ad hoc and inconsistently. The review standards document consolidates all compliance checks into one authoritative checklist that the docs-audit prompt automates. Alternative considered: embedding the checklist in the docs-audit prompt only (rejected; the prompt is automation, not authority. The standards must be readable and referenceable independent of the prompt).

- **Metadata manifest as a foundational document.** The documentation system requires a machine-readable manifest (`docs/metadata.yml`) that maps every document, its relationships, and its entry points. This enables AI agents to perform instant lookups without traversing the file tree, and ensures relationship symmetry (every `depends_on` has a matching `required_by`) is maintained. The manifest has its own JSON Schema (`docs/metadata.schema.json`) for IDE validation. The conventions for maintaining the manifest are foundational because they govern how all documentation is indexed and interconnected. Alternative considered: embedding metadata conventions in `documentation.md` (rejected; the manifest has distinct rules for symmetry, ID namespacing, and maintenance procedures that warrant a standalone document).

- **Tiered context loading for foundation documents.** Not every session requires the full documentation-authoring context. Code implementation sessions need the decision-making framework (`philosophy.md`) and the documentation map (`metadata.yml`) but not markdown formatting rules or review checklists. Loading all foundation documents into every session costs ~10k tokens, which is ~32% of a 32k context window. Splitting foundation docs into `load: always` (core operating context, ~2.5k tokens) and `load: documentation` (authoring context, loaded only when writing or editing docs) reduces the default startup cost by ~76% while preserving full capability for documentation tasks. The `load` field on each foundation entry in `metadata.yml` makes this a mechanical lookup, not a judgment call, so even baseline-tier models can follow it reliably. Alternative considered: loading everything every session (rejected; disproportionate token cost for code-only sessions on smaller context windows). Alternative considered: requiring the model to judge what is relevant without a field (rejected; weaker models fail at relevance judgments, and the `load` field eliminates the need for judgment entirely).

## Dependencies

- [docs/foundation/specs.md](../specs.md): defines the spec template this document follows
- [docs/foundation/ai-authoring.md](../ai-authoring.md): cross-model authoring conventions
- [docs/README.md](../../README.md): documentation entry point and index

## Structure

- `philosophy.md`: guiding principles for decision-making across the codebase
- `markdown.md`: markdown formatting conventions for all documentation files
- `documentation.md`: how documentation is managed, organized, and written
- `specs.md`: specification template and process for all topic docs
- `ai-authoring.md`: cross-model AI authoring conventions
- `review-standards.md`: post-update review checklist for documentation compliance
- `metadata.md`: metadata manifest conventions and maintenance
- `spec/README.md`: this file; meta-documentation for the foundation
